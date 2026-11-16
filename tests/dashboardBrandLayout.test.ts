import fs from "node:fs"
import path from "node:path"
import { describe, expect, it } from "vitest"

const repoRoot = path.resolve(__dirname, "..")

describe("Dashboard brand alignment and section separation", () => {
  it("renders NutriSyncBrand in a sticky header and separates the 3 detail sections", () => {
    const dashboardSource = fs.readFileSync(
      path.join(repoRoot, "feature/dashboard/HealthDashboardClient.tsx"),
      "utf8"
    )

    expect(dashboardSource).toContain("<NutriSyncBrand")
    expect(dashboardSource).toContain('id="overview"')
    expect(dashboardSource).toContain('id="insights"')
    expect(dashboardSource).toContain('id="tracking"')
    expect(dashboardSource).toContain("01 · Wellness Overview")
    expect(dashboardSource).toContain("02 · AI Health Insights")
    expect(dashboardSource).toContain("03 · Weight &amp; Activity Log")
    expect(dashboardSource).toContain("<ThemeToggle")
  })
})
