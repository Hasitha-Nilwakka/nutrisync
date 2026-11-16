// LandingFooter renders the NutriSync brand lockup, section links, and clinical disclaimer.
import { NutriSyncBrand } from "@/feature/landing/NutriSyncBrand"

export function LandingFooter() {
  return (
    <footer className="mt-auto border-t border-border/70 bg-card/60">
      <div className="mx-auto flex w-full max-w-7xl flex-col gap-6 px-4 py-8 sm:px-6 lg:flex-row lg:items-center lg:justify-between lg:px-8">
        <NutriSyncBrand showTagline size="sm" />

        <div className="flex flex-wrap items-center gap-5 text-xs font-medium text-muted-foreground">
          <a href="#features" className="transition-colors hover:text-foreground">
            Features
          </a>
          <a href="#how-it-works" className="transition-colors hover:text-foreground">
            How It Works
          </a>
          <a href="#privacy" className="transition-colors hover:text-foreground">
            Privacy &amp; GDPR
          </a>
          <a href="#auth-panel" className="transition-colors hover:text-foreground">
            Log In
          </a>
        </div>

        <p className="max-w-md text-xs leading-relaxed text-muted-foreground lg:text-right">
          NutriSync provides personalized wellness analytics and educational
          nutrition insights. It is not a substitute for professional medical
          advice, diagnosis, or treatment.
        </p>
      </div>
    </footer>
  )
}
