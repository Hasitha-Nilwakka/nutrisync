"use client"

import { useId, useState } from "react"
import { TrendingDown, TrendingUp, Minus } from "lucide-react"
import {
  computeWeightTrendPoints,
  convertKgToUnitValue,
} from "@/lib/dashboardChartHelpers"
import { formatWeightForUnit, type WeightEntryPayload } from "@/lib/schemas/dashboardSchema"

interface WeightTrendChartProps {
  entries: WeightEntryPayload[]
  preferredWeightUnit: string
  targetWeightKg?: number | null
}

export function WeightTrendChart({
  entries,
  preferredWeightUnit,
  targetWeightKg = null,
}: WeightTrendChartProps) {
  const gradientId = useId()
  const [activeIndex, setActiveIndex] = useState<number | null>(null)
  const unitLabel = preferredWeightUnit.toLowerCase() === "lbs" ? "lbs" : "kg"
  const points = computeWeightTrendPoints(entries, preferredWeightUnit)

  if (points.length === 0) {
    return (
      <div className="rounded-xl border border-dashed border-border/80 bg-muted/20 p-5 text-center text-xs text-muted-foreground">
        Log your first weight entry to visualize your trend over time.
      </div>
    )
  }

  if (points.length === 1) {
    return (
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border/80 bg-muted/20 p-4">
        <div>
          <p className="text-xs font-medium text-muted-foreground">Weight Trend</p>
          <p className="text-lg font-semibold text-foreground">
            {formatWeightForUnit(points[0].weightKg, preferredWeightUnit)}
          </p>
        </div>
        <p className="text-xs text-muted-foreground">
          Add one more entry to plot your weight trend curve.
        </p>
      </div>
    )
  }

  const latestPoint = points[points.length - 1]
  const firstPoint = points[0]
  const deltaDisplay = Math.round((latestPoint.displayWeight - firstPoint.displayWeight) * 10) / 10
  const targetDisplay =
    typeof targetWeightKg === "number" && targetWeightKg > 0
      ? convertKgToUnitValue(targetWeightKg, preferredWeightUnit)
      : null

  const allValues = points.map((item) => item.displayWeight)
  if (targetDisplay !== null) {
    allValues.push(targetDisplay)
  }

  const rawMin = Math.min(...allValues)
  const rawMax = Math.max(...allValues)
  const paddingRange = Math.max(0.5, (rawMax - rawMin) * 0.15)
  const yMin = rawMin - paddingRange
  const yMax = rawMax + paddingRange
  const ySpan = yMax - yMin || 1

  const svgWidth = 560
  const svgHeight = 180
  const padLeft = 44
  const padRight = 16
  const padTop = 16
  const padBottom = 28
  const plotWidth = svgWidth - padLeft - padRight
  const plotHeight = svgHeight - padTop - padBottom

  const toX = (index: number) =>
    padLeft + (index / (points.length - 1)) * plotWidth
  const toY = (val: number) =>
    padTop + plotHeight - ((val - yMin) / ySpan) * plotHeight

  const linePoints = points.map((p, idx) => `${toX(idx).toFixed(1)},${toY(p.displayWeight).toFixed(1)}`).join(" ")
  const areaPath = [
    `M ${toX(0).toFixed(1)} ${(padTop + plotHeight).toFixed(1)}`,
    ...points.map((p, idx) => `L ${toX(idx).toFixed(1)} ${toY(p.displayWeight).toFixed(1)}`),
    `L ${toX(points.length - 1).toFixed(1)} ${(padTop + plotHeight).toFixed(1)}`,
    "Z",
  ].join(" ")

  const gridValues = [rawMin, (rawMin + rawMax) / 2, rawMax]
  const targetY = targetDisplay !== null ? toY(targetDisplay) : null
  const inspectedPoint = activeIndex !== null ? points[activeIndex] ?? null : null

  return (
    <div className="space-y-3 rounded-xl border border-border/80 bg-muted/15 p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-3">
          <div>
            <p className="text-xs font-medium text-muted-foreground">
              Weight Trend ({points.length} entries)
            </p>
            <div className="flex items-baseline gap-2">
              <span className="text-base font-semibold text-foreground">
                {latestPoint.displayWeight} {unitLabel}
              </span>
              <span
                className={`inline-flex items-center gap-0.5 text-xs font-medium ${
                  deltaDisplay < 0
                    ? "text-emerald-600 dark:text-emerald-400"
                    : deltaDisplay > 0
                      ? "text-amber-600 dark:text-amber-400"
                      : "text-muted-foreground"
                }`}
              >
                {deltaDisplay < 0 ? (
                  <TrendingDown className="size-3.5" />
                ) : deltaDisplay > 0 ? (
                  <TrendingUp className="size-3.5" />
                ) : (
                  <Minus className="size-3.5" />
                )}
                {deltaDisplay > 0 ? `+${deltaDisplay}` : deltaDisplay} {unitLabel}
              </span>
            </div>
          </div>
        </div>

        <div className="text-right text-xs text-muted-foreground">
          {inspectedPoint ? (
            <span className="font-medium text-foreground">
              {inspectedPoint.label}: {inspectedPoint.displayWeight} {unitLabel}
            </span>
          ) : (
            <span>
              Range: {rawMin.toFixed(1)}–{rawMax.toFixed(1)} {unitLabel}
              {targetDisplay !== null ? ` · Target: ${targetDisplay} ${unitLabel}` : ""}
            </span>
          )}
        </div>
      </div>

      <svg
        viewBox={`0 0 ${svgWidth} ${svgHeight}`}
        className="h-44 w-full overflow-visible"
        role="img"
        aria-label="Weight trend chart over time"
      >
        <defs>
          <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#0B74B8" stopOpacity="0.28" />
            <stop offset="100%" stopColor="#0B74B8" stopOpacity="0.02" />
          </linearGradient>
        </defs>

        {gridValues.map((val, idx) => {
          const y = toY(val)
          return (
            <g key={idx}>
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
                {val.toFixed(1)}
              </text>
            </g>
          )
        })}

        {targetY !== null && (
          <g>
            <line
              x1={padLeft}
              y1={targetY}
              x2={svgWidth - padRight}
              y2={targetY}
              stroke="#18A999"
              strokeWidth="1.5"
              strokeDasharray="5 4"
            />
            <text
              x={svgWidth - padRight}
              y={targetY - 4}
              textAnchor="end"
              className="fill-[#18A999] text-[10px] font-medium"
            >
              Target {targetDisplay} {unitLabel}
            </text>
          </g>
        )}

        <path d={areaPath} fill={`url(#${gradientId})`} />

        <polyline
          fill="none"
          stroke="#0B74B8"
          strokeWidth="2.5"
          strokeLinecap="round"
          strokeLinejoin="round"
          points={linePoints}
        />

        {points.map((point, idx) => {
          const cx = toX(idx)
          const cy = toY(point.displayWeight)
          const isActive = activeIndex === idx
          return (
            <g key={point.id}>
              <circle
                cx={cx}
                cy={cy}
                r={isActive ? 5.5 : 3.5}
                fill={isActive ? "#18A999" : "#0B74B8"}
                stroke="var(--color-card)"
                strokeWidth="2"
                tabIndex={0}
                role="button"
                aria-label={`${point.label}: ${point.displayWeight} ${unitLabel}`}
                onMouseEnter={() => setActiveIndex(idx)}
                onMouseLeave={() => setActiveIndex(null)}
                onFocus={() => setActiveIndex(idx)}
                onBlur={() => setActiveIndex(null)}
                className="cursor-pointer outline-none"
              />
            </g>
          )
        })}

        <text
          x={padLeft}
          y={svgHeight - 6}
          textAnchor="start"
          className="fill-muted-foreground text-[10px]"
        >
          {firstPoint.label}
        </text>
        <text
          x={svgWidth - padRight}
          y={svgHeight - 6}
          textAnchor="end"
          className="fill-muted-foreground text-[10px]"
        >
          {latestPoint.label}
        </text>
      </svg>
    </div>
  )
}
