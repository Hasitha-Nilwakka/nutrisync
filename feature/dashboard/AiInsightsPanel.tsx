// AiInsightsPanel renders the 3-Layer AI Insights (healthStatusAnalysis, progressEvaluation,
// prioritized recommendations, weekly/monthly summaries), service/fallback status, and regeneration trigger.
"use client"

import { useState } from "react"
import axios from "axios"
import {
  AlertTriangle,
  BrainCircuit,
  CalendarDays,
  CheckCircle2,
  Loader2,
  RefreshCw,
  ShieldAlert,
  Sparkles,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { showAPIResponseToast } from "@/components/ui/showToast"
import {
  type AIInsightRecordPayload,
  type AIRecommendationItem,
  parseRecommendationsJson,
  type RegenerateAIInsightsPayload,
} from "@/lib/schemas/dashboardSchema"

interface AiInsightsPanelProps {
  latestInsights: AIInsightRecordPayload | null
  insightHistory: AIInsightRecordPayload[]
  aiServiceAvailable: boolean
  isCachedFallback: boolean
  allowAiProcessing: boolean
  onInsightsRegenerated: (payload: RegenerateAIInsightsPayload) => void
}

const PRIORITY_RANK: Record<string, number> = {
  high: 0,
  medium: 1,
  low: 2,
}

function getPriorityBadgeClasses(priority: string): string {
  const normalized = priority.toLowerCase()
  if (normalized === "high") {
    return "border-rose-500/30 bg-rose-500/10 text-rose-700 dark:text-rose-300"
  }
  if (normalized === "medium") {
    return "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300"
  }
  return "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
}

function sortRecommendationsByPriority(items: AIRecommendationItem[]): AIRecommendationItem[] {
  return [...items].sort((a, b) => {
    const rankA = PRIORITY_RANK[a.priority.toLowerCase()] ?? 3
    const rankB = PRIORITY_RANK[b.priority.toLowerCase()] ?? 3
    return rankA - rankB
  })
}

export function AiInsightsPanel({
  latestInsights,
  insightHistory,
  aiServiceAvailable,
  isCachedFallback,
  allowAiProcessing,
  onInsightsRegenerated,
}: AiInsightsPanelProps) {
  const [isRegenerating, setIsRegenerating] = useState(false)
  const [selectedInsightId, setSelectedInsightId] = useState<string | null>(null)

  const orderedHistory = [...insightHistory].sort((a, b) => b.version - a.version)
  const activeInsight =
    (selectedInsightId ? orderedHistory.find((item) => item.id === selectedInsightId) : null) ??
    latestInsights ??
    orderedHistory[0] ??
    null

  const handleRegenerate = async () => {
    setIsRegenerating(true)
    try {
      const response = await axios.post<RegenerateAIInsightsPayload>(
        "/api/health/insights/regenerate",
        {},
        { withCredentials: true }
      )
      setSelectedInsightId(null)
      onInsightsRegenerated(response.data)

      if (response.data.isCachedFallback || !response.data.aiServiceAvailable) {
        showAPIResponseToast({
          type: "info",
          title: "Showing Cached Insights",
          message: "The AI analytics service is currently unavailable; displaying your latest cached insights.",
        })
      } else {
        showAPIResponseToast({
          type: "success",
          title: "Insights Regenerated",
          message: "Fresh 3-Layer AI health insights have been generated.",
        })
      }
    } catch {
      showAPIResponseToast({
        type: "error",
        title: "Regeneration Failed",
        message: "Could not regenerate AI insights right now. Please try again.",
      })
    } finally {
      setIsRegenerating(false)
    }
  }

  const recommendations = activeInsight
    ? sortRecommendationsByPriority(parseRecommendationsJson(activeInsight.recommendationsJson))
    : []

  return (
    <section className="flex flex-col gap-5 rounded-2xl border border-border/80 bg-card p-6 shadow-sm">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="space-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <BrainCircuit className="size-5 text-primary" />
            <h2 className="text-lg font-semibold tracking-tight">AI Health Insights</h2>
            {isCachedFallback ? (
              <span className="inline-flex items-center gap-1 rounded-full border border-amber-500/30 bg-amber-500/10 px-2.5 py-0.5 text-xs font-medium text-amber-700 dark:text-amber-300">
                <AlertTriangle className="size-3" />
                Cached Fallback
              </span>
            ) : aiServiceAvailable ? (
              <span className="inline-flex items-center gap-1 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2.5 py-0.5 text-xs font-medium text-emerald-700 dark:text-emerald-300">
                <CheckCircle2 className="size-3" />
                Live AI Service
              </span>
            ) : (
              <span className="inline-flex items-center gap-1 rounded-full border border-rose-500/30 bg-rose-500/10 px-2.5 py-0.5 text-xs font-medium text-rose-700 dark:text-rose-300">
                <ShieldAlert className="size-3" />
                AI Service Offline
              </span>
            )}
          </div>
          {activeInsight && (
            <p className="text-xs text-muted-foreground">
              Version v{activeInsight.version} · Goal: {activeInsight.primaryGoalSnapshot.replace(/_/g, " ")} · Strategy:{" "}
              {activeInsight.promptStrategy} · {new Date(activeInsight.createdAt).toLocaleString()}
            </p>
          )}
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {orderedHistory.length > 1 && (
            <select
              aria-label="Select insight version"
              value={activeInsight?.id ?? ""}
              onChange={(event) => setSelectedInsightId(event.target.value || null)}
              className="h-8 rounded-lg border border-input bg-background px-2.5 text-xs outline-none focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/40"
            >
              {orderedHistory.map((item) => (
                <option key={item.id} value={item.id}>
                  v{item.version} ({new Date(item.createdAt).toLocaleDateString()})
                </option>
              ))}
            </select>
          )}

          <Button
            type="button"
            size="sm"
            disabled={isRegenerating || !allowAiProcessing}
            onClick={handleRegenerate}
          >
            {isRegenerating ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <RefreshCw className="size-4" />
            )}
            <span>{isRegenerating ? "Regenerating..." : "Regenerate Insights"}</span>
          </Button>
        </div>
      </div>

      {!allowAiProcessing && (
        <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 p-3.5 text-xs text-amber-800 dark:text-amber-200">
          AI processing is currently disabled in your Health Profile privacy preferences. Enable{" "}
          <strong>Allow AI Processing</strong> to generate personalized insights.
        </div>
      )}

      {!activeInsight ? (
        <div className="rounded-lg border border-dashed p-6 text-center text-sm text-muted-foreground">
          No AI insights are available yet. Click <strong>Regenerate Insights</strong> to run the 3-Layer AI analysis.
        </div>
      ) : (
        <div className="space-y-5">
          {/* Status Analysis & Progress Evaluation */}
          <div className="grid gap-4 md:grid-cols-2">
            <div className="rounded-lg border bg-muted/20 p-4">
              <h3 className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
                Health Status Analysis
              </h3>
              <p className="mt-2 text-sm leading-relaxed">{activeInsight.healthStatusAnalysis}</p>
            </div>
            <div className="rounded-lg border bg-muted/20 p-4">
              <h3 className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
                Progress Evaluation
              </h3>
              <p className="mt-2 text-sm leading-relaxed">{activeInsight.progressEvaluation}</p>
            </div>
          </div>

          {/* Prioritized Recommendations */}
          <div className="space-y-3">
            <div className="flex items-center gap-2 text-sm font-medium">
              <Sparkles className="size-4 text-primary" />
              <span>Prioritized Recommendations</span>
            </div>

            {recommendations.length === 0 ? (
              <p className="text-xs text-muted-foreground">No specific recommendations were returned for this version.</p>
            ) : (
              <div className="grid gap-3 md:grid-cols-2">
                {recommendations.map((rec, index) => (
                  <div key={`${rec.title}-${index}`} className="flex flex-col justify-between gap-2 rounded-lg border p-4">
                    <div className="space-y-1.5">
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <span className="text-sm font-semibold">{rec.title}</span>
                        <div className="flex items-center gap-1.5">
                          <span className="rounded-full border px-2 py-0.5 text-[11px] capitalize text-muted-foreground">
                            {rec.category}
                          </span>
                          <span
                            className={`rounded-full border px-2 py-0.5 text-[11px] font-medium capitalize ${getPriorityBadgeClasses(rec.priority)}`}
                          >
                            {rec.priority}
                          </span>
                        </div>
                      </div>
                      {rec.action && <p className="text-xs leading-relaxed text-foreground/90">{rec.action}</p>}
                    </div>

                    {rec.expectedImpact && (
                      <p className="border-t pt-2 text-[11px] text-muted-foreground">
                        <strong>Expected Impact:</strong> {rec.expectedImpact}
                      </p>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Weekly & Monthly Summaries */}
          <div className="grid gap-4 md:grid-cols-2">
            <div className="rounded-lg border p-4">
              <div className="flex items-center gap-1.5 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
                <CalendarDays className="size-3.5" />
                <span>Weekly Summary</span>
              </div>
              <p className="mt-2 text-xs leading-relaxed text-foreground/90">{activeInsight.weeklySummary}</p>
            </div>
            <div className="rounded-lg border p-4">
              <div className="flex items-center gap-1.5 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
                <CalendarDays className="size-3.5" />
                <span>Monthly Summary</span>
              </div>
              <p className="mt-2 text-xs leading-relaxed text-foreground/90">{activeInsight.monthlySummary}</p>
            </div>
          </div>
        </div>
      )}
    </section>
  )
}
