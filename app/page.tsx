// Root route (/) page composing the NutriSync landing header, hero + AuthTabs split,
// core capabilities Bento grid, 3-step workflow, and footer.
import { FeaturesSection } from "@/feature/landing/FeaturesSection"
import { HeroSection } from "@/feature/landing/HeroSection"
import { HowItWorksSection } from "@/feature/landing/HowItWorksSection"
import { LandingFooter } from "@/feature/landing/LandingFooter"
import { LandingHeader } from "@/feature/landing/LandingHeader"

export default function Home() {
  return (
    <div className="flex min-h-dvh flex-col bg-background text-foreground">
      <LandingHeader />
      <HeroSection />
      <FeaturesSection />
      <HowItWorksSection />
      <LandingFooter />
    </div>
  )
}
