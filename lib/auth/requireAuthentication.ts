// Server-only helper that verifies the active user session against the Go API (/api/auth/session)
// using the HttpOnly access_token cookie and redirects unauthenticated requests to the root login route (/).
import "server-only"
import { cookies } from "next/headers"
import { redirect } from "next/navigation"

export interface AuthenticatedSession {
  userId: string
  sessionId: string
  email: string
  consented: boolean
  twoFactorEnabled: boolean
  hasHealthProfile: boolean
}

// requireAuthentication reads the access_token cookie, verifies the session with the Go API server,
// and returns { userId, sessionId, email, consented, twoFactorEnabled, hasHealthProfile } or redirects to "/" when unauthenticated.
export async function requireAuthentication(): Promise<AuthenticatedSession> {
  const cookieStore = await cookies()
  const accessToken = cookieStore.get("access_token")?.value

  if (!accessToken) {
    redirect("/")
  }

  const goApiBaseUrl = process.env.GO_API_URL || "http://localhost:8080"
  const response = await fetch(`${goApiBaseUrl}/api/auth/session`, {
    headers: {
      Cookie: `access_token=${encodeURIComponent(accessToken)}`,
    },
    cache: "no-store",
  })

  if (!response.ok) {
    redirect("/")
  }

  const sessionData = (await response.json()) as AuthenticatedSession
  return {
    userId: sessionData.userId,
    sessionId: sessionData.sessionId,
    email: sessionData.email ?? "",
    consented: Boolean(sessionData.consented),
    twoFactorEnabled: Boolean(sessionData.twoFactorEnabled),
    hasHealthProfile: Boolean(sessionData.hasHealthProfile),
  }
}
