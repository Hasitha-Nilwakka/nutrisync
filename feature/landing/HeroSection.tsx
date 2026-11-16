// HeroSection renders the responsive above-the-fold NutriSync showcase and authentication split
// on /, balancing the brand narrative and wellness pillar preview with AuthTabs across breakpoints.
import { Activity, LockKeyhole, ShieldCheck, Sparkles } from "lucide-react"
import AuthTabs from "@/feature/auth/AuthTabs"
import { NutriSyncBrand } from "@/feature/landing/NutriSyncBrand"

const WELLNESS_PILLARS = [
  { label: "BMI Balance", weight: "30%", value: 84, barClass: "bg-chart-1" },
  { label: "Activity", weight: "30%", value: 78, barClass: "bg-chart-2" },
  { label: "Progress", weight: "20%", value: 90, barClass: "bg-chart-3" },
  { label: "Habits", weight: "20%", value: 82, barClass: "bg-chart-4" },
] as const

export function HeroSection() {
  return (
    <section
      aria-label="NutriSync overview and authentication"
      className="relative overflow-hidden border-b border-border/60"
    >
      {/* Theme-aware atmospheric radial gradients matching the logo's Blue -> Teal -> Green palette */}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 -z-10"
      >
        <div className="absolute -top-28 -left-24 h-80 w-80 rounded-full bg-[#0B74B8]/14 blur-3xl dark:bg-[#0B74B8]/22" />
        <div className="absolute top-1/3 left-1/2 h-96 w-96 -translate-x-1/2 rounded-full bg-[#18A999]/12 blur-3xl dark:bg-[#18A999]/18" />
        <div className="absolute -right-20 bottom-0 h-80 w-80 rounded-full bg-[#43B047]/12 blur-3xl dark:bg-[#43B047]/18" />
      </div>

      <div className="mx-auto grid w-full max-w-7xl grid-cols-1 items-center gap-10 px-4 py-10 sm:px-6 sm:py-14 lg:grid-cols-12 lg:gap-12 lg:px-8 lg:py-16">
        {/* Left Column: Brand Identity, Headline, Trust Pills & Wellness Formula Preview */}
        <div className="flex flex-col gap-6 lg:col-span-7">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <NutriSyncBrand showTagline size="lg" />
            <span className="inline-flex items-center gap-1.5 rounded-full border border-primary/25 bg-secondary px-3 py-1 text-xs font-semibold text-secondary-foreground">
              <ShieldCheck className="size-3.5 text-primary" aria-hidden="true" />
              3-Layer AI Guardrails
            </span>
          </div>

          <div className="space-y-4">
            <h1 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl lg:text-5xl lg:leading-[1.12]">
              Numbers Don&apos;t Lie.{" "}
              <span className="bg-linear-to-r from-[#0B74B8] via-[#18A999] to-[#43B047] bg-clip-text text-transparent dark:from-[#38BDF8] dark:via-[#22B8B0] dark:to-[#5AC155]">
                Personalized AI Nutrition
              </span>{" "}
              Built on Real Biometrics.
            </h1>

            <p className="max-w-2xl text-base leading-relaxed text-muted-foreground sm:text-lg">
              Track weight, activity, and daily habits with deterministic{" "}
              <span className="font-semibold text-foreground">0–100 Wellness Scoring</span>{" "}
              and clinically guardrailed AI insights—engineered with two-tier PII
              anonymization so your health data stays yours.
            </p>
          </div>

          {/* 3 Verifiable Trust & Telemetry Highlights */}
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <div className="rounded-xl border border-border/80 bg-card/85 p-3.5 shadow-xs backdrop-blur-xs">
              <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
                <Activity className="size-4 text-primary" aria-hidden="true" />
                <span>Wellness Engine</span>
              </div>
              <p className="mt-1 text-lg font-bold text-foreground">0–100 Score</p>
              <p className="text-xs text-muted-foreground">7d &amp; 30d trend slopes</p>
            </div>

            <div className="rounded-xl border border-border/80 bg-card/85 p-3.5 shadow-xs backdrop-blur-xs">
              <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
                <LockKeyhole className="size-4 text-[#18A999]" aria-hidden="true" />
                <span>Privacy Boundary</span>
              </div>
              <p className="mt-1 text-lg font-bold text-foreground">2-Tier PII Strip</p>
              <p className="text-xs text-muted-foreground">Zero identifiers sent to LLM</p>
            </div>

            <div className="rounded-xl border border-border/80 bg-card/85 p-3.5 shadow-xs backdrop-blur-xs">
              <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
                <Sparkles className="size-4 text-[#43B047]" aria-hidden="true" />
                <span>Data Ownership</span>
              </div>
              <p className="mt-1 text-lg font-bold text-foreground">GDPR &amp; 2FA</p>
              <p className="text-xs text-muted-foreground">One-click JSON export</p>
            </div>
          </div>

          {/* Compact Wellness Formula Preview (hidden on very small phones to keep AuthTabs near top) */}
          <div className="hidden rounded-2xl border border-border/80 bg-card/90 p-5 shadow-xs backdrop-blur-xs sm:block">
            <div className="mb-3 flex items-center justify-between gap-2">
              <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Deterministic Wellness Formula
              </span>
              <span className="rounded-full bg-accent px-2.5 py-0.5 text-xs font-semibold text-accent-foreground">
                Sample Score: 83.2 / 100
              </span>
            </div>
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
              {WELLNESS_PILLARS.map((pillar) => (
                <div key={pillar.label} className="space-y-1.5">
                  <div className="flex items-baseline justify-between text-xs">
                    <span className="font-medium text-foreground">{pillar.label}</span>
                    <span className="font-mono text-muted-foreground">{pillar.weight}</span>
                  </div>
                  <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                    <div
                      className={`h-full rounded-full ${pillar.barClass}`}
                      style={{ width: `${pillar.value}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Right Column: Elevated Authentication Card */}
        <div
          id="auth-panel"
          className="flex w-full justify-center lg:col-span-5 lg:justify-end"
        >
          <AuthTabs />
        </div>
      </div>
    </section>
  )
}
