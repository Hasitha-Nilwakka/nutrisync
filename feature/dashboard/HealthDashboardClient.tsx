// HealthDashboardClient is the interactive client container for /dashboard.
// Fetches GET /api/health/dashboard via Next.js /api/health/* rewrite and coordinates
// WellnessScorePanel, AiInsightsPanel, WeightActivitySection, GdprExportButton, and navigation.
"use client"

import { useCallback, useEffect, useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import axios from "axios"
import { Loader2, RefreshCw, UserCog } from "lucide-react"
import LogoutButton from "@/components/ui/LogoutButton"
import { Button } from "@/components/ui/button"
import { AiInsightsPanel } from "@/feature/dashboard/AiInsightsPanel"
import { GdprExportButton } from "@/feature/dashboard/GdprExportButton"
import { WeightActivitySection } from "@/feature/dashboard/WeightActivitySection"
import { WellnessScorePanel } from "@/feature/dashboard/WellnessScorePanel"
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
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-6 px-4 py-8 sm:px-6 lg:px-8">
      {/* Top Navigation & Actions */}
      <header className="flex flex-wrap items-center justify-between gap-4 border-b pb-5">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">NutriSync Health Dashboard</h1>
          <p className="text-sm text-muted-foreground">Signed in as {userEmail}</p>
        </div>

        <nav className="flex flex-wrap items-center gap-2" aria-label="Dashboard actions">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={isRefreshing || isLoading}
            onClick={() => void fetchDashboard(true)}
          >
            {isRefreshing ? <Loader2 className="size-4 animate-spin" /> : <RefreshCw className="size-4" />}
            <span>Refresh</span>
          </Button>
          <GdprExportButton />
          <Link
            href="/profile"
            className="inline-flex h-7 items-center justify-center gap-1.5 rounded-[min(var(--radius-md),12px)] border border-border bg-background px-2.5 text-[0.8rem] font-medium whitespace-nowrap transition-all hover:bg-muted hover:text-foreground"
          >
            <UserCog className="size-4" />
            <span>Profile</span>
          </Link>
          <LogoutButton />
        </nav>
      </header>

      {isLoading ? (
        <div className="flex min-h-64 flex-col items-center justify-center gap-3 rounded-xl border bg-card p-8">
          <Loader2 className="size-6 animate-spin text-primary" />
          <p className="text-sm text-muted-foreground">Loading your wellness analytics and AI insights...</p>
        </div>
      ) : errorMessage || !dashboard ? (
        <div className="flex flex-col items-center justify-center gap-4 rounded-xl border bg-card p-8 text-center">
          <p className="text-sm text-destructive">{errorMessage ?? "Dashboard data is unavailable."}</p>
          <Button type="button" variant="outline" size="sm" onClick={() => void fetchDashboard(false)}>
            Try Again
          </Button>
        </div>
      ) : (
        <div className="flex flex-col gap-6">
          <WellnessScorePanel
            healthProfile={dashboard.healthProfile}
            latestWellness={dashboard.latestWellness}
          />

          <AiInsightsPanel
            latestInsights={dashboard.latestInsights}
            insightHistory={dashboard.insightHistory}
            aiServiceAvailable={dashboard.aiServiceAvailable}
            isCachedFallback={dashboard.isCachedFallback}
            allowAiProcessing={dashboard.healthProfile?.allowAiProcessing ?? true}
            onInsightsRegenerated={handleInsightsRegenerated}
          />

          <WeightActivitySection
            weightHistory={dashboard.weightHistory}
            activityHistory={dashboard.activityHistory}
            preferredWeightUnit={dashboard.healthProfile?.preferredWeightUnit ?? "kg"}
            onEntryLogged={() => fetchDashboard(true)}
          />
        </div>
      )}
    </div>
  )
}
