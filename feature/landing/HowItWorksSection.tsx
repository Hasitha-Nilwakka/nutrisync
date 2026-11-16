// HowItWorksSection outlines the 3-step NutriSync user journey from onboarding
// and biometric logging to guardrailed AI nutrition insights.
const STEPS = [
  {
    step: "01",
    title: "Create Your Health Profile",
    description:
      "Confirm GDPR consent, set your primary wellness goal, and record baseline height, weight, and dietary restrictions.",
  },
  {
    step: "02",
    title: "Log Daily Weight & Activity",
    description:
      "Capture check-ins in metric or imperial units. NutriSync normalizes every entry and updates your 7-day and 30-day trend slopes.",
  },
  {
    step: "03",
    title: "Sync Guardrailed AI Insights",
    description:
      "Receive actionable nutrition and habit guidance filtered against your medical restrictions and anchored to your Wellness Score.",
  },
] as const

export function HowItWorksSection() {
  return (
    <section
      id="how-it-works"
      aria-labelledby="how-it-works-heading"
      className="border-t border-border/60 bg-muted/40"
    >
      <div className="mx-auto w-full max-w-7xl px-4 py-14 sm:px-6 sm:py-18 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-xs font-semibold uppercase tracking-widest text-primary">
            Workflow
          </p>
          <h2
            id="how-it-works-heading"
            className="mt-2 text-2xl font-bold tracking-tight text-foreground sm:text-3xl"
          >
            From Baseline Biometrics to Daily Clarity
          </h2>
        </div>

        <ol className="mt-10 grid grid-cols-1 gap-6 md:grid-cols-3">
          {STEPS.map((item) => (
            <li
              key={item.step}
              className="relative rounded-2xl border border-border/80 bg-card p-6 shadow-xs"
            >
              <span className="inline-flex rounded-lg bg-linear-to-r from-[#0B74B8] to-[#18A999] px-2.5 py-1 font-mono text-xs font-bold text-white">
                STEP {item.step}
              </span>
              <h3 className="mt-4 text-lg font-semibold text-foreground">
                {item.title}
              </h3>
              <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                {item.description}
              </p>
            </li>
          ))}
        </ol>
      </div>
    </section>
  )
}
