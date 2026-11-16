"use client"

import { useEffect, useState } from "react"
import Link from "next/link"
import { useSearchParams } from "next/navigation"
import axios from "axios"
import { CheckCircle2, Loader2, XCircle } from "lucide-react"

type VerificationStatus =
  | { state: "loading" }
  | { state: "success"; title: string; message: string }
  | { state: "error"; title: string; message: string }

// Deduplicates in-flight verification requests per token so React StrictMode's
// mount -> cleanup -> remount cycle does not consume the single-use token twice.
const verificationRequestCache = new Map<string, Promise<VerificationStatus>>()

function verifyEmailTokenOnce(token: string): Promise<VerificationStatus> {
  const existingRequest = verificationRequestCache.get(token)
  if (existingRequest) {
    return existingRequest
  }

  const requestPromise = axios
    .get("/api/auth/verify", { params: { token } })
    .then((response): VerificationStatus => {
      const title =
        typeof response.data?.title === "string"
          ? response.data.title
          : "Email Verified"

      return {
        state: "success",
        title,
        message:
          "Your email address has been verified. You can now log in to your NutriSync account.",
      }
    })
    .catch((err): VerificationStatus => {
      const data = axios.isAxiosError(err) ? err.response?.data : undefined
      const title =
        typeof data?.title === "string" ? data.title : "Verification Failed"
      const message =
        typeof data?.message === "string"
          ? `${data.message}. If you already verified your email, you can return to the login page.`
          : "This verification link is invalid, expired, or has already been used. If you already verified your email, try logging in."

      return { state: "error", title, message }
    })

  verificationRequestCache.set(token, requestPromise)
  return requestPromise
}

export function VerifyEmailStatus() {
  const searchParams = useSearchParams()
  const token = searchParams.get("token")?.trim() ?? ""
  const [status, setStatus] = useState<VerificationStatus>({ state: "loading" })

  useEffect(() => {
    if (!token) {
      return
    }

    let cancelled = false

    void verifyEmailTokenOnce(token).then((resolvedStatus) => {
      if (!cancelled) {
        setStatus(resolvedStatus)
      }
    })

    return () => {
      cancelled = true
    }
  }, [token])

  const effectiveStatus: VerificationStatus = !token
    ? {
        state: "error",
        title: "Missing Verification Token",
        message:
          "This verification link is missing a valid token. Please check your email and try the link again.",
      }
    : status

  return (
    <div className="w-full max-w-md overflow-hidden rounded-2xl border border-border/80 bg-card shadow-sm">
      <div
        aria-hidden="true"
        className="h-1.5 w-full bg-linear-to-r from-[#0B74B8] via-[#18A999] to-[#43B047]"
      />
      <div className="space-y-4 p-6 text-center">
        {effectiveStatus.state === "loading" && (
          <>
            <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-primary/10 text-primary">
              <Loader2 className="size-6 animate-spin" />
            </div>
            <h1 className="text-lg font-semibold text-foreground">
              Verifying your email...
            </h1>
            <p className="text-sm leading-relaxed text-muted-foreground">
              Please wait a moment while we confirm your NutriSync account.
            </p>
          </>
        )}

        {effectiveStatus.state === "success" && (
          <>
            <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-[#43B047]/15 text-[#43B047]">
              <CheckCircle2 className="size-6" />
            </div>
            <h1 className="text-lg font-semibold text-foreground">
              {effectiveStatus.title}
            </h1>
            <p className="text-sm leading-relaxed text-muted-foreground">
              {effectiveStatus.message}
            </p>
            <div className="pt-2">
              <Link
                href="/"
                className="inline-flex h-10 w-full items-center justify-center rounded-lg bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
              >
                Continue to Log In
              </Link>
            </div>
          </>
        )}

        {effectiveStatus.state === "error" && (
          <>
            <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-destructive/15 text-destructive">
              <XCircle className="size-6" />
            </div>
            <h1 className="text-lg font-semibold text-foreground">
              {effectiveStatus.title}
            </h1>
            <p className="text-sm leading-relaxed text-muted-foreground">
              {effectiveStatus.message}
            </p>
            <div className="pt-2">
              <Link
                href="/"
                className="inline-flex h-10 w-full items-center justify-center rounded-lg border border-border bg-background px-4 text-sm font-medium text-foreground transition-colors hover:bg-accent"
              >
                Back to Home
              </Link>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
