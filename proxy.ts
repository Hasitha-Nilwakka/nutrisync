import { NextResponse, type NextRequest } from "next/server"

const DEFAULT_GO_API_BASE_URL = "http://localhost:8080"

export function isJwtExpired(token: string, nowSeconds = Math.floor(Date.now() / 1000)): boolean {
  const parts = token.split(".")
  if (parts.length !== 3) {
    return true
  }

  try {
    const base64 = parts[1].replace(/-/g, "+").replace(/_/g, "/")
    const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), "=")
    const payloadJson = atob(padded)
    const payload = JSON.parse(payloadJson) as { exp?: unknown }
    if (typeof payload.exp !== "number" || !Number.isFinite(payload.exp)) {
      return true
    }
    return payload.exp <= nowSeconds + 10
  } catch {
    return true
  }
}

function buildUpdatedCookieHeader(
  request: NextRequest,
  setCookieHeaders: string[]
): string {
  const cookieMap = new Map<string, string>()
  for (const cookie of request.cookies.getAll()) {
    cookieMap.set(cookie.name, cookie.value)
  }

  for (const headerValue of setCookieHeaders) {
    const firstPair = headerValue.split(";")[0]?.trim()
    if (!firstPair) continue
    const eqIndex = firstPair.indexOf("=")
    if (eqIndex <= 0) continue
    const name = firstPair.slice(0, eqIndex).trim()
    const value = firstPair.slice(eqIndex + 1).trim()
    if (name === "access_token" || name === "refresh_token") {
      cookieMap.set(name, value)
    }
  }

  return Array.from(cookieMap.entries())
    .map(([name, value]) => `${name}=${value}`)
    .join("; ")
}

export async function proxy(request: NextRequest) {
  const accessToken = request.cookies.get("access_token")?.value
  const refreshToken = request.cookies.get("refresh_token")?.value

  if (accessToken && !isJwtExpired(accessToken)) {
    return NextResponse.next()
  }

  if (!refreshToken) {
    return NextResponse.next()
  }

  const goApiBaseUrl = process.env.GO_API_URL || DEFAULT_GO_API_BASE_URL
  try {
    const refreshResponse = await fetch(`${goApiBaseUrl}/api/auth/refresh`, {
      method: "POST",
      headers: {
        Cookie: `refresh_token=${refreshToken}`,
      },
      cache: "no-store",
    })

    if (!refreshResponse.ok) {
      const loginRedirect = NextResponse.redirect(new URL("/", request.url))
      loginRedirect.cookies.delete("access_token")
      loginRedirect.cookies.delete("refresh_token")
      return loginRedirect
    }

    const setCookieHeaders = refreshResponse.headers.getSetCookie()
    const requestHeaders = new Headers(request.headers)
    const updatedCookieHeader = buildUpdatedCookieHeader(request, setCookieHeaders)
    if (updatedCookieHeader) {
      requestHeaders.set("cookie", updatedCookieHeader)
    }

    const response = NextResponse.next({
      request: {
        headers: requestHeaders,
      },
    })

    for (const setCookieHeader of setCookieHeaders) {
      response.headers.append("set-cookie", setCookieHeader)
    }

    return response
  } catch {
    return NextResponse.next()
  }
}

export const config = {
  matcher: ["/dashboard/:path*", "/profile/:path*"],
}
