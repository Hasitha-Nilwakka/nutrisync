import fs from "node:fs"
import path from "node:path"
import { describe, expect, it } from "vitest"
import { shouldAttemptTokenRefresh } from "../lib/api/setupAxiosRefreshInterceptor"
import { isJwtExpired } from "../proxy"

const repoRoot = path.resolve(__dirname, "..")

function createUnsignedTestJwt(exp: number): string {
  const header = Buffer.from(JSON.stringify({ alg: "HS256", typ: "JWT" })).toString("base64url")
  const payload = Buffer.from(JSON.stringify({ sub: "user-1", exp })).toString("base64url")
  return `${header}.${payload}.sig`
}

describe("Access & refresh token silent refresh flow", () => {
  it("detects expired and valid JWT expiration timestamps", () => {
    const now = 1_700_000_000
    expect(isJwtExpired(createUnsignedTestJwt(now - 60), now)).toBe(true)
    expect(isJwtExpired(createUnsignedTestJwt(now + 5), now)).toBe(true)
    expect(isJwtExpired(createUnsignedTestJwt(now + 120), now)).toBe(false)
    expect(isJwtExpired("malformed.jwt", now)).toBe(true)
  })

  it("attempts refresh only on non-auth 401s that have not already retried", () => {
    expect(shouldAttemptTokenRefresh("/api/health/dashboard", 401, false)).toBe(true)
    expect(shouldAttemptTokenRefresh("/api/health/dashboard", 401, true)).toBe(false)
    expect(shouldAttemptTokenRefresh("/api/health/dashboard", 403, false)).toBe(false)
    expect(shouldAttemptTokenRefresh("/api/auth/login", 401, false)).toBe(false)
    expect(shouldAttemptTokenRefresh("/api/auth/refresh", 401, false)).toBe(false)
  })

  it("configures proxy.ts for /dashboard and /profile and mounts AuthRefreshProvider in layout.tsx", () => {
    const proxySource = fs.readFileSync(path.join(repoRoot, "proxy.ts"), "utf8")
    expect(proxySource).toContain("/api/auth/refresh")
    expect(proxySource).toContain("/dashboard/:path*")
    expect(proxySource).toContain("/profile/:path*")

    const layoutSource = fs.readFileSync(path.join(repoRoot, "app/layout.tsx"), "utf8")
    expect(layoutSource).toContain("<AuthRefreshProvider")
  })
})
