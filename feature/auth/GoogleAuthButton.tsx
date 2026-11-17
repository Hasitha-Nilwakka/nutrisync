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
export default function GoogleAuthButton({ disabled = false }: { disabled?: boolean }) {
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
      className="h-10 w-full gap-2.5 rounded-lg border-border/80 bg-background font-medium shadow-2xs transition-all hover:border-border hover:bg-muted/70"
      disabled={disabled || isRedirecting}
      onClick={handleGoogleSignIn}
    >
      {isRedirecting ? (
        <Spinner />
      ) : (
        <svg
          aria-hidden="true"
          viewBox="0 0 24 24"
          className="size-4 shrink-0"
        >
          <path
            fill="#4285F4"
            d="M23.745 12.27c0-.7-.06-1.4-.19-2.07H12v4.51h6.6c-.29 1.52-1.14 2.82-2.4 3.68v3.05h3.88c2.27-2.09 3.665-5.17 3.665-9.17Z"
          />
          <path
            fill="#34A853"
            d="M12 24c3.3 0 6.08-1.09 8.105-2.96l-3.88-3.05c-1.1.74-2.51 1.18-4.225 1.18-3.25 0-6.01-2.19-6.995-5.14H1.06v3.14C3.125 21.24 7.31 24 12 24Z"
          />
          <path
            fill="#FBBC05"
            d="M5.005 14.03A7.22 7.22 0 0 1 4.62 12c0-.7.13-1.39.385-2.03V6.83H1.06A11.98 11.98 0 0 0 0 12c0 1.94.46 3.77 1.06 5.17l3.945-3.14Z"
          />
          <path
            fill="#EA4335"
            d="M12 4.83c1.8 0 3.41.62 4.68 1.83l3.51-3.51C18.075 1.19 15.3 0 12 0 7.31 0 3.125 2.76 1.06 6.83l3.945 3.14C5.99 7.02 8.75 4.83 12 4.83Z"
          />
        </svg>
      )}
      <span>{isRedirecting ? "Redirecting to Google..." : "Continue with Google"}</span>
    </Button>
  )
}
