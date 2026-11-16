// FeaturesSection presents the three core NutriSync pillars: Deterministic Wellness Engine,
// 3-Layer AI Nutrition Guardrails, and Privacy/2FA/GDPR Data Ownership.
import { BarChart3, BrainCircuit, ShieldAlert } from "lucide-react"

const FEATURES = [
  {
    id: "wellness-engine",
    icon: BarChart3,
    badge: "Vectorized Analytics",
    title: "Deterministic Wellness Score",
    description:
      "Every check-in computes a transparent 0–100 score weighted across BMI (30%), Activity (30%), Progress (20%), and Habits (20%), paired with 7-day and 30-day trend slopes.",
    accentClass: "text-[#0B74B8] dark:text-[#38BDF8] bg-[#0B74B8]/10",
  },
  {
    id: "ai-guardrails",
    icon: BrainCircuit,
    badge: "3-Layer Pipeline",
    title: "Clinically Guardrailed AI Insights",
    description:
      "SciSpacy extracts dietary restrictions and contraindicated terms before prompting, compacted trend statistics guide generation, and Layer 3 validates every recommendation with a deterministic clinical fallback.",
    accentClass: "text-[#18A999] dark:text-[#22B8B0] bg-[#18A999]/10",
  },
  {
    id: "privacy",
    icon: ShieldAlert,
    badge: "Privacy & Security",
    title: "Two-Tier PII Stripping & GDPR",
    description:
      "Direct identifiers are stripped before leaving our Go API, free-text notes are scrubbed again in Python, accounts support TOTP 2FA, and you can export your full dataset anytime.",
    accentClass: "text-[#43B047] dark:text-[#5AC155] bg-[#43B047]/10",
  },
] as const

export function FeaturesSection() {
  return (
    <section
      id="features"
      aria-labelledby="features-heading"
      className="mx-auto w-full max-w-7xl px-4 py-14 sm:px-6 sm:py-18 lg:px-8"
    >
      <div className="mx-auto max-w-2xl text-center">
        <p className="text-xs font-semibold uppercase tracking-widest text-primary">
          Core Architecture
        </p>
        <h2
          id="features-heading"
          className="mt-2 text-2xl font-bold tracking-tight text-foreground sm:text-3xl"
        >
          Precision Telemetry Meets Guardrailed AI
        </h2>
        <p className="mt-3 text-sm leading-relaxed text-muted-foreground sm:text-base">
          Designed so your numbers drive every recommendation—with strict privacy
          boundaries and zero black-box scoring.
        </p>
      </div>

      <div className="mt-10 grid grid-cols-1 gap-6 md:grid-cols-3">
        {FEATURES.map((feature) => {
          const Icon = feature.icon
          return (
            <article
              key={feature.id}
              id={feature.id === "privacy" ? "privacy" : undefined}
              className="flex flex-col justify-between rounded-2xl border border-border/80 bg-card p-6 shadow-xs transition-shadow hover:shadow-md"
            >
              <div>
                <div className="flex items-center justify-between gap-3">
                  <span
                    className={`inline-flex size-10 items-center justify-center rounded-xl ${feature.accentClass}`}
                  >
                    <Icon className="size-5" aria-hidden="true" />
                  </span>
                  <span className="rounded-full bg-secondary px-2.5 py-0.5 text-xs font-medium text-secondary-foreground">
                    {feature.badge}
                  </span>
                </div>
                <h3 className="mt-5 text-lg font-semibold text-foreground">
                  {feature.title}
                </h3>
                <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                  {feature.description}
                </p>
              </div>
            </article>
          )
        })}
      </div>
    </section>
  )
}
