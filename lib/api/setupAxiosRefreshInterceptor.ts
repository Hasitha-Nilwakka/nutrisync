import axios, { type InternalAxiosRequestConfig } from "axios"

interface RetryableRequestConfig extends InternalAxiosRequestConfig {
  _retry?: boolean
}

const EXCLUDED_REFRESH_PATHS = [
  "/api/auth/login",
  "/api/auth/register",
  "/api/auth/refresh",
  "/api/auth/logout",
  "/api/auth/passwordreset",
  "/api/auth/2fa/login-verify",
]

export function shouldAttemptTokenRefresh(
  requestUrl: string | undefined,
  statusCode: number | undefined,
  alreadyRetried: boolean | undefined
): boolean {
  if (statusCode !== 401 || alreadyRetried || !requestUrl) {
    return false
  }

  return !EXCLUDED_REFRESH_PATHS.some((excludedPath) =>
    requestUrl.includes(excludedPath)
  )
}

let inFlightRefreshPromise: Promise<boolean> | null = null
let interceptorInstalled = false

function executeSingleFlightRefresh(): Promise<boolean> {
  if (inFlightRefreshPromise) {
    return inFlightRefreshPromise
  }

  inFlightRefreshPromise = axios
    .post("/api/auth/refresh", {}, { withCredentials: true })
    .then(() => true)
    .catch(() => false)
    .finally(() => {
      inFlightRefreshPromise = null
    })

  return inFlightRefreshPromise
}

export function installAxiosRefreshInterceptor(): void {
  if (interceptorInstalled || typeof window === "undefined") {
    return
  }
  interceptorInstalled = true

  axios.interceptors.response.use(
    (response) => response,
    async (error: unknown) => {
      if (!axios.isAxiosError(error) || !error.config) {
        return Promise.reject(error)
      }

      const originalConfig = error.config as RetryableRequestConfig
      if (
        !shouldAttemptTokenRefresh(
          originalConfig.url,
          error.response?.status,
          originalConfig._retry
        )
      ) {
        return Promise.reject(error)
      }

      originalConfig._retry = true
      const refreshed = await executeSingleFlightRefresh()
      if (!refreshed) {
        return Promise.reject(error)
      }

      return axios.request(originalConfig)
    }
  )
}
