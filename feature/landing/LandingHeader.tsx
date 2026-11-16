// LandingHeader renders the sticky top navigation bar on / with the NutriSync brand mark,
// section anchor links, and the Light/Dark mode ThemeToggle.
import Link from "next/link"
import { ThemeToggle } from "@/components/theme/ThemeToggle"
import { Button } from "@/components/ui/button"
import { NutriSyncBrand } from "@/feature/landing/NutriSyncBrand"

export function LandingHeader() {
  return (
    <header className="sticky top-0 z-40 w-full border-b border-border/70 bg-background/85 backdrop-blur-md">
      <div className="mx-auto flex h-16 w-full max-w-7xl items-center justify-between gap-4 px-4 sm:px-6 lg:px-8">
        <Link href="/" className="rounded-md focus-visible:outline-2 focus-visible:outline-ring">
          <NutriSyncBrand size="sm" />
        </Link>

        <nav
          aria-label="Landing page sections"
          className="hidden items-center gap-6 text-sm font-medium text-muted-foreground md:flex"
        >
          <a
            href="#features"
            className="transition-colors hover:text-foreground"
          >
            Features
          </a>
          <a
            href="#how-it-works"
            className="transition-colors hover:text-foreground"
          >
            How It Works
          </a>
          <a
            href="#privacy"
            className="transition-colors hover:text-foreground"
          >
            Privacy &amp; Safety
          </a>
        </nav>

        <div className="flex items-center gap-2.5">
          <Button
            variant="ghost"
            size="sm"
            render={<a href="#auth-panel" />}
            className="hidden sm:inline-flex"
          >
            Log In
          </Button>
          <ThemeToggle />
        </div>
      </div>
    </header>
  )
}
