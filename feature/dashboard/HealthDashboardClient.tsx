// HealthDashboardClient is the interactive client container for /dashboard.
// Fetches GET /api/health/dashboard via Next.js /api/health/* rewrite and coordinates
// WellnessScorePanel, AiInsightsPanel, WeightActivitySection, GdprExportButton, and navigation.
"use client"

import { useCallback, useEffect, useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import axios from "axios"
import { Loader2, RefreshCw, UserCog } from "lucide-react"
import { ThemeToggle } from "@/components/theme/ThemeToggle"
import LogoutButton from "@/components/ui/LogoutButton"
import { Button } from "@/components/ui/button"
import { AiInsightsPanel } from "@/feature/dashboard/AiInsightsPanel"
import { GdprExportButton } from "@/feature/dashboard/GdprExportButton"
import { UserHealthInfographicSection } from "@/feature/dashboard/UserHealthInfographicSection"
import { WeightActivitySection } from "@/feature/dashboard/WeightActivitySection"
import { WellnessScorePanel } from "@/feature/dashboard/WellnessScorePanel"
import { NutriSyncBrand } from "@/feature/landing/NutriSyncBrand"
import type {
  HealthDashboardPayload,
  RegenerateAIInsightsPayload,
} from "@/lib/schemas/dashboardSchema"

interface HealthDashboardClientProps {
  userEmail: string
}

export function HealthDashboardClient({ userEmail }: HealthDashboardClientProps) {
  const router = useRouter()
  const [dashboard, setDashboard] = useState<HealthDashboardPayload | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  const fetchDashboard = useCallback(
    async (silent = false) => {
      if (silent) {
        setIsRefreshing(true)
      } else {
        setIsLoading(true)
      }
      setErrorMessage(null)

      try {
        const response = await axios.get<HealthDashboardPayload>("/api/health/dashboard", {
          withCredentials: true,
        })
        setDashboard(response.data)
      } catch (error) {
        if (axios.isAxiosError(error) && error.response?.status === 401) {
          router.replace("/")
          router.refresh()
          return
        }
        const message =
          axios.isAxiosError(error) && typeof error.response?.data?.message === "string"
            ? error.response.data.message
            : "Unable to load your health dashboard right now."
        setErrorMessage(message)
      } finally {
        setIsLoading(false)
        setIsRefreshing(false)
      }
    },
    [router]
  )

  useEffect(() => {
    let cancelled = false

    axios
      .get<HealthDashboardPayload>("/api/health/dashboard", {
        withCredentials: true,
      })
      .then((response) => {
        if (!cancelled) {
          setDashboard(response.data)
          setErrorMessage(null)
        }
      })
      .catch((error: unknown) => {
        if (cancelled) {
          return
        }
        if (axios.isAxiosError(error) && error.response?.status === 401) {
          router.replace("/")
          router.refresh()
          return
        }
        const message =
          axios.isAxiosError(error) && typeof error.response?.data?.message === "string"
            ? error.response.data.message
            : "Unable to load your health dashboard right now."
        setErrorMessage(message)
      })
      .finally(() => {
        if (!cancelled) {
          setIsLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [router])

  const handleInsightsRegenerated = (regenerated: RegenerateAIInsightsPayload) => {
    setDashboard((prev) =>
      prev
        ? {
            ...prev,
            latestInsights: regenerated.latestInsights,
            insightHistory: regenerated.insightHistory,
            aiServiceAvailable: regenerated.aiServiceAvailable,
            isCachedFallback: regenerated.isCachedFallback,
          }
        : prev
    )
  }

  return (
    <div className="relative min-h-screen bg-background">
      <div
        aria-hidden="true"
        className="pointer-events-none fixed inset-0 -z-10 bg-[radial-gradient(ellipse_at_top,_rgba(11,116,184,0.10),_transparent_55%),radial-gradient(ellipse_at_bottom_right,_rgba(24,169,153,0.08),_transparent_50%)] dark:bg-[radial-gradient(ellipse_at_top,_rgba(47,168,230,0.14),_transparent_55%),radial-gradient(ellipse_at_bottom_right,_rgba(41,208,187,0.10),_transparent_50%)]"
      />

      {/* Sticky Branded Header matching LandingHeader */}
      <header className="sticky top-0 z-40 w-full border-b border-border/70 bg-background/85 backdrop-blur-md">
        <div className="mx-auto flex h-16 w-full max-w-6xl items-center justify-between gap-4 px-4 sm:px-6 lg:px-8">
          <Link
            href="/dashboard"
            className="rounded-md focus-visible:outline-2 focus-visible:outline-ring"
          >
            <NutriSyncBrand size="sm" />
          </Link>

          <nav
            aria-label="Dashboard sections"
            className="hidden items-center gap-6 text-sm font-medium text-muted-foreground md:flex"
          >
            <a href="#overview" className="transition-colors hover:text-foreground">
              Overview
            </a>
            <a href="#insights" className="transition-colors hover:text-foreground">
              AI Insights
            </a>
            <a href="#tracking" className="transition-colors hover:text-foreground">
              Tracking
            </a>
            <a href="#infographic" className="transition-colors hover:text-foreground">
              Infographic
            </a>
          </nav>

          <nav className="flex flex-wrap items-center gap-2" aria-label="Dashboard actions">
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={isRefreshing || isLoading}
              onClick={() => void fetchDashboard(true)}
            >
              {isRefreshing ? <Loader2 className="size-4 animate-spin" /> : <RefreshCw className="size-4" />}
              <span className="hidden sm:inline">Refresh</span>
            </Button>
            <GdprExportButton />
            <Link
              href="/profile"
              className="inline-flex h-7 items-center justify-center gap-1.5 rounded-[min(var(--radius-md),12px)] border border-border bg-background px-2.5 text-[0.8rem] font-medium whitespace-nowrap transition-all hover:bg-muted hover:text-foreground"
            >
              <UserCog className="size-4" />
              <span className="hidden sm:inline">Profile</span>
            </Link>
            <ThemeToggle />
            <LogoutButton />
          </nav>
        </div>
      </header>

      <main className="mx-auto flex w-full max-w-6xl flex-col gap-8 px-4 py-8 sm:px-6 lg:px-8">
        {/* Page Context Header */}
        <div className="flex flex-col gap-1 border-b border-border/60 pb-6">
          <h1 className="text-2xl font-bold tracking-tight text-foreground sm:text-3xl">
            Health Dashboard
          </h1>
          <p className="text-sm text-muted-foreground">
            Your biometric wellness score, guardrailed AI insights, and daily tracking · Signed in as{" "}
            <span className="font-medium text-foreground">{userEmail}</span>
          </p>
        </div>

        {isLoading ? (
          <div className="flex min-h-64 flex-col items-center justify-center gap-3 rounded-2xl border border-border/80 bg-card p-8 shadow-sm">
            <Loader2 className="size-6 animate-spin text-primary" />
            <p className="text-sm text-muted-foreground">Loading your wellness analytics and AI insights...</p>
          </div>
        ) : errorMessage || !dashboard ? (
          <div className="flex flex-col items-center justify-center gap-4 rounded-2xl border border-border/80 bg-card p-8 text-center shadow-sm">
            <p className="text-sm text-destructive">{errorMessage ?? "Dashboard data is unavailable."}</p>
            <Button type="button" variant="outline" size="sm" onClick={() => void fetchDashboard(false)}>
              Try Again
            </Button>
          </div>
        ) : (
          <div className="flex flex-col gap-10">
            {/* Section 01: Wellness Overview */}
            <section id="overview" aria-labelledby="section-overview-heading" className="scroll-mt-24 space-y-3">
              <div className="flex items-center justify-between">
                <p
                  id="section-overview-heading"
                  className="text-xs font-semibold tracking-wider text-muted-foreground uppercase"
                >
                  01 · Wellness Overview
                </p>
              </div>
              <WellnessScorePanel
                healthProfile={dashboard.healthProfile}
                latestWellness={dashboard.latestWellness}
              />
            </section>

            {/* Section 02: AI Health Insights */}
            <section
              id="insights"
              aria-labelledby="section-insights-heading"
              className="scroll-mt-24 space-y-3 border-t border-border/60 pt-8"
            >
              <div className="flex items-center justify-between">
                <p
                  id="section-insights-heading"
                  className="text-xs font-semibold tracking-wider text-muted-foreground uppercase"
                >
                  02 · AI Health Insights
                </p>
              </div>
              <AiInsightsPanel
                latestInsights={dashboard.latestInsights}
                insightHistory={dashboard.insightHistory}
                aiServiceAvailable={dashboard.aiServiceAvailable}
                isCachedFallback={dashboard.isCachedFallback}
                allowAiProcessing={dashboard.healthProfile?.allowAiProcessing ?? true}
                onInsightsRegenerated={handleInsightsRegenerated}
              />
            </section>

            {/* Section 03: Weight & Activity Log */}
            <section
              id="tracking"
              aria-labelledby="section-tracking-heading"
              className="scroll-mt-24 space-y-3 border-t border-border/60 pt-8"
            >
              <div className="flex items-center justify-between">
                <p
                  id="section-tracking-heading"
                  className="text-xs font-semibold tracking-wider text-muted-foreground uppercase"
                >
                  03 · Weight &amp; Activity Log
                </p>
              </div>
              <WeightActivitySection
                weightHistory={dashboard.weightHistory}
                activityHistory={dashboard.activityHistory}
                preferredWeightUnit={dashboard.healthProfile?.preferredWeightUnit ?? "kg"}
                targetWeightKg={dashboard.healthProfile?.targetWeightKg ?? null}
                onEntryLogged={() => fetchDashboard(true)}
              />
            </section>

            {/* Section 04: Health Infographic */}
            <section
              id="infographic"
              aria-labelledby="section-infographic-heading"
              className="scroll-mt-24 space-y-3 border-t border-border/60 pt-8"
            >
              <div className="flex items-center justify-between">
                <p
                  id="section-infographic-heading"
                  className="text-xs font-semibold tracking-wider text-muted-foreground uppercase"
                >
                  04 · Health Infographic
                </p>
              </div>
              <UserHealthInfographicSection
                healthProfile={dashboard.healthProfile}
                latestWellness={dashboard.latestWellness}
                weightHistory={dashboard.weightHistory}
                activityHistory={dashboard.activityHistory}
              />
            </section>
          </div>
        )}

        <footer className="border-t border-border/60 pt-6 pb-2 text-center text-xs text-muted-foreground">
          NutriSync wellness scores and AI insights are informational and do not replace professional medical advice.
        </footer>
      </main>
    </div>
  )
}
