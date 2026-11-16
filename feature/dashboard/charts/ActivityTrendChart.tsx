"use client"

import { useState } from "react"
import { summarizeActivityBreakdown } from "@/lib/dashboardChartHelpers"
import {
  ACTIVITY_INTENSITY_LABELS,
  type ActivityEntryPayload,
  type ActivityIntensity,
} from "@/lib/schemas/dashboardSchema"

const INTENSITY_COLOR_HEX: Record<string, string> = {
  low: "#18A999",
  moderate: "#0B74B8",
  high: "#F59E0B",
  vigorous: "#EF4444",
}

interface ActivityTrendChartProps {
  entries: ActivityEntryPayload[]
}

export function ActivityTrendChart({ entries }: ActivityTrendChartProps) {
  const [activeIndex, setActiveIndex] = useState<number | null>(null)

  if (entries.length === 0) {
    return (
      <div className="rounded-xl border border-dashed border-border/80 bg-muted/20 p-5 text-center text-xs text-muted-foreground">
        Log an activity session to visualize your workout duration and intensity mix.
      </div>
    )
  }

  const ordered = [...entries]
    .sort((a, b) => new Date(a.recordedAt).getTime() - new Date(b.recordedAt).getTime())
    .slice(-14)

  const summary = summarizeActivityBreakdown(entries)
  const maxDuration = Math.max(30, ...ordered.map((item) => item.durationMinutes))
  const svgWidth = 560
  const svgHeight = 180
  const padLeft = 40
  const padRight = 14
  const padTop = 16
  const padBottom = 28
  const plotWidth = svgWidth - padLeft - padRight
  const plotHeight = svgHeight - padTop - padBottom
  const slotWidth = plotWidth / ordered.length
  const barWidth = Math.min(28, Math.max(10, slotWidth * 0.62))

  const inspected = activeIndex !== null ? ordered[activeIndex] ?? null : null
  const gridTicks = [0, Math.round(maxDuration / 2), maxDuration]

  return (
    <div className="space-y-3 rounded-xl border border-border/80 bg-muted/15 p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <p className="text-xs font-medium text-muted-foreground">
            Activity Duration &amp; Intensity (last {ordered.length} sessions)
          </p>
          <div className="flex flex-wrap items-baseline gap-3">
            <span className="text-base font-semibold text-foreground">
              {summary.totalMinutes} min total
            </span>
            <span className="text-xs text-muted-foreground">
              Avg {summary.averageMinutes} min / session
            </span>
          </div>
        </div>

        <div className="text-right text-xs text-muted-foreground">
          {inspected ? (
            <span className="font-medium text-foreground">
              {inspected.exerciseType} · {inspected.durationMinutes} min (
              {ACTIVITY_INTENSITY_LABELS[inspected.intensity as ActivityIntensity] ?? inspected.intensity})
            </span>
          ) : (
            <div className="flex flex-wrap items-center justify-end gap-2.5">
              {(["low", "moderate", "high", "vigorous"] as const).map((level) => (
                <span key={level} className="inline-flex items-center gap-1 text-[11px]">
                  <span
                    className="size-2 rounded-full"
                    style={{ backgroundColor: INTENSITY_COLOR_HEX[level] }}
                  />
                  {ACTIVITY_INTENSITY_LABELS[level]}
                </span>
              ))}
            </div>
          )}
        </div>
      </div>

      <svg
        viewBox={`0 0 ${svgWidth} ${svgHeight}`}
        className="h-44 w-full overflow-visible"
        role="img"
        aria-label="Activity duration and intensity bar chart"
      >
        {gridTicks.map((tick) => {
          const y = padTop + plotHeight - (tick / maxDuration) * plotHeight
          return (
            <g key={tick}>
              <line
                x1={padLeft}
                y1={y}
                x2={svgWidth - padRight}
                y2={y}
                stroke="currentColor"
                strokeDasharray="3 4"
                className="text-border"
              />
              <text
                x={padLeft - 6}
                y={y + 3}
                textAnchor="end"
                className="fill-muted-foreground text-[10px]"
              >
                {tick}m
              </text>
            </g>
          )
        })}

        {ordered.map((entry, idx) => {
          const barHeight = Math.max(4, (entry.durationMinutes / maxDuration) * plotHeight)
          const x = padLeft + idx * slotWidth + (slotWidth - barWidth) / 2
          const y = padTop + plotHeight - barHeight
          const fill = INTENSITY_COLOR_HEX[entry.intensity.toLowerCase()] ?? "#0B74B8"
          const isActive = activeIndex === idx
          const shortDate = new Date(entry.recordedAt).toLocaleDateString(undefined, {
            month: "numeric",
            day: "numeric",
          })

          return (
            <g key={entry.id}>
              <rect
                x={x}
                y={y}
                width={barWidth}
                height={barHeight}
                rx="4"
                fill={fill}
                opacity={activeIndex === null || isActive ? 0.92 : 0.45}
                tabIndex={0}
                role="button"
                aria-label={`${entry.exerciseType}: ${entry.durationMinutes} min (${entry.intensity}) on ${shortDate}`}
                onMouseEnter={() => setActiveIndex(idx)}
                onMouseLeave={() => setActiveIndex(null)}
                onFocus={() => setActiveIndex(idx)}
                onBlur={() => setActiveIndex(null)}
                className="cursor-pointer outline-none transition-opacity"
              />
              {(idx === 0 || idx === ordered.length - 1 || ordered.length <= 7) && (
                <text
                  x={x + barWidth / 2}
                  y={svgHeight - 6}
                  textAnchor="middle"
                  className="fill-muted-foreground text-[10px]"
                >
                  {shortDate}
                </text>
              )}
            </g>
          )
        })}
      </svg>
    </div>
  )
}
