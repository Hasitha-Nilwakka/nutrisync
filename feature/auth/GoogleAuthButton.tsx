// Reusable wireframe "Continue with Google" button component wired to the Go API Google OAuth endpoint (/api/auth/oauth/google).
// Uses the global Shadcn Button and Spinner primitives and a monochrome SVG icon to comply with wireframe/no-color rules.
"use client"

import { useEffect, useState } from "react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"

// Maps backend OAuth error query parameters (from HandleGoogleOAuthCallback) to user-facing messages.
const OAUTH_ERROR_MESSAGES: Record<string, string> = {
  oauth_failed: "Unable to start Google sign-in. Please try again.",
  oauth_state_invalid: "Google sign-in session expired or was invalid. Please try again.",
  oauth_code_missing: "Google sign-in authorization code was missing.",
  oauth_exchange_failed: "Failed to exchange Google authorization code.",
  oauth_userinfo_failed: "Failed to retrieve verified profile details from Google.",
  oauth_account_error: "Failed to link or create your account with Google.",
  oauth_session_error: "Failed to establish an authenticated session.",
}

// GoogleAuthButton initiates the Google OAuth 2.0 flow via top-level navigation to /api/auth/oauth/google
// (rewritten by Next.js to the Go API server) and surfaces any OAuth callback redirect errors.
export default function GoogleAuthButton() {
  const [isRedirecting, setIsRedirecting] = useState(false)

  useEffect(() => {
    const searchParams = new URLSearchParams(window.location.search)
    const oauthErrorCode = searchParams.get("error")
    if (!oauthErrorCode || !(oauthErrorCode in OAUTH_ERROR_MESSAGES)) {
      return
    }

    toast.error("Google Sign-In Failed", {
      position: "top-center",
      description: OAUTH_ERROR_MESSAGES[oauthErrorCode],
    })

    searchParams.delete("error")
    const remainingQuery = searchParams.toString()
    const cleanUrl = remainingQuery
      ? `${window.location.pathname}?${remainingQuery}`
      : window.location.pathname
    window.history.replaceState(null, "", cleanUrl)
  }, [])

  // Redirects the browser to the Next.js rewrite endpoint for Google OAuth login initiation.
  const handleGoogleSignIn = () => {
    setIsRedirecting(true)
    window.location.assign("/api/auth/oauth/google")
  }

  return (
    <Button
      type="button"
      variant="outline"
      className="w-full gap-2"
      disabled={isRedirecting}
      onClick={handleGoogleSignIn}
    >
      {isRedirecting ? (
        <Spinner />
      ) : (
        <svg
          aria-hidden="true"
          viewBox="0 0 24 24"
          className="size-4 shrink-0"
          fill="currentColor"
        >
          <path d="M12.48 10.92v3.28h7.84c-.24 1.84-.853 3.187-1.787 4.133-1.147 1.147-2.933 2.4-6.053 2.4-4.827 0-8.6-3.893-8.6-8.72s3.773-8.72 8.6-8.72c2.6 0 4.507 1.027 5.907 2.347l2.307-2.307C18.747 1.44 16.133 0 12.48 0 5.867 0 .307 5.387.307 12s5.56 12 12.173 12c3.573 0 6.267-1.173 8.373-3.36 2.16-2.16 2.84-5.213 2.84-7.667 0-.76-.053-1.467-.173-2.053H12.48z" />
        </svg>
      )}
      <span>{isRedirecting ? "Redirecting to Google..." : "Continue with Google"}</span>
    </Button>
  )
}
