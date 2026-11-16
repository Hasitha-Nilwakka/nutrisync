// ThemeToggle provides an accessible light/dark mode switch on the landing page
// using next-themes and useSyncExternalStore to prevent SSR hydration mismatches.
"use client"

import * as React from "react"
import { Moon, Sun } from "lucide-react"
import { useTheme } from "next-themes"
import { Button } from "@/components/ui/button"

const emptySubscribe = () => () => {}

export function ThemeToggle() {
  const { resolvedTheme, setTheme } = useTheme()
  const mounted = React.useSyncExternalStore(
    emptySubscribe,
    () => true,
    () => false
  )

  const isDark = mounted && resolvedTheme === "dark"
  const nextTheme = isDark ? "light" : "dark"
  const label = isDark ? "Switch to light theme" : "Switch to dark theme"

  return (
    <Button
      type="button"
      variant="outline"
      size="icon"
      aria-label={label}
      title={label}
      onClick={() => setTheme(nextTheme)}
      className="relative rounded-full border-border/80 bg-card/80 backdrop-blur-xs transition-colors hover:bg-secondary"
    >
      <Sun
        className="hidden size-4 text-amber-400 dark:block"
        aria-hidden="true"
      />
      <Moon
        className="block size-4 text-primary dark:hidden"
        aria-hidden="true"
      />
      <span className="sr-only">{label}</span>
    </Button>
  )
}
