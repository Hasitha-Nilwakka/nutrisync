// NutriSyncBrand renders the transparent-background SVG emblem alongside the themed
// NutriSync wordmark and optional "Personalized AI Nutrition" tagline.
import Image from "next/image"

interface NutriSyncBrandProps {
  showTagline?: boolean
  size?: "sm" | "lg"
}

export function NutriSyncBrand({
  showTagline = false,
  size = "sm",
}: NutriSyncBrandProps) {
  const iconSize = size === "lg" ? 56 : 36

  return (
    <div className="inline-flex items-center gap-3">
      <Image
        src="/nutrisync-mark.svg"
        alt="NutriSync circular leaf and neural node emblem"
        width={iconSize}
        height={iconSize}
        priority
        className="shrink-0"
      />
      <div className="flex flex-col">
        <span
          className={
            size === "lg"
              ? "text-2xl font-bold tracking-tight text-foreground sm:text-3xl"
              : "text-lg font-bold tracking-tight text-foreground"
          }
        >
          NutriSync
        </span>
        {showTagline && (
          <span className="text-xs font-medium tracking-wide text-muted-foreground sm:text-sm">
            Personalized AI Nutrition
          </span>
        )}
      </div>
    </div>
  )
}
