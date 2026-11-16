import { Suspense } from "react"
import { VerifyEmailStatus } from "@/feature/auth/register/VerifyEmailStatus"
import { NutriSyncBrand } from "@/feature/landing/NutriSyncBrand"

export default function VerifyEmailPage() {
  return (
    <main className="relative flex min-h-screen flex-col items-center justify-center overflow-hidden bg-background px-4 py-12">
      <div
        aria-hidden="true"
        className="pointer-events-none fixed inset-0 -z-10 bg-[radial-gradient(ellipse_at_top,_rgba(11,116,184,0.12),_transparent_55%),radial-gradient(ellipse_at_bottom_right,_rgba(24,169,153,0.10),_transparent_50%)] dark:bg-[radial-gradient(ellipse_at_top,_rgba(47,168,230,0.16),_transparent_55%),radial-gradient(ellipse_at_bottom_right,_rgba(41,208,187,0.12),_transparent_50%)]"
      />
      <div className="mb-8">
        <NutriSyncBrand />
      </div>
      <Suspense
        fallback={
          <div className="w-full max-w-md rounded-2xl border border-border/80 bg-card p-6 text-center text-sm text-muted-foreground shadow-sm">
            Verifying your email...
          </div>
        }
      >
        <VerifyEmailStatus />
      </Suspense>
    </main>
  )
}
