// Unit tests verifying the NutriSync front-page SVG emblem, Light/Dark mode CSS tokens,
// ThemeProvider wiring in app/layout.tsx, and landing page module exports.
import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

const ROOT = resolve(__dirname, "..");

describe("NutriSync brand emblem & asset hygiene", () => {
  it("provides a transparent-background SVG emblem with logo gradient stops", () => {
    const svgPath = resolve(ROOT, "public/nutrisync-mark.svg");
    expect(existsSync(svgPath)).toBe(true);

    const svg = readFileSync(svgPath, "utf8");
    expect(svg).toContain("<svg");
    expect(svg).toContain('fill="none"');
    expect(svg).toContain("#0B74B8");
    expect(svg).toContain("#18A999");
    expect(svg).toContain("#43B047");
  });

  it("removes the legacy nutrisync-hero.jpg asset", () => {
    const legacyHeroPath = resolve(ROOT, "public/nutrisync-hero.jpg");
    expect(existsSync(legacyHeroPath)).toBe(false);
  });
});

describe("Light & Dark mode CSS tokens and layout wiring", () => {
  it("configures :root light and .dark tokens without conflicting prefers-color-scheme overrides", () => {
    const css = readFileSync(resolve(ROOT, "app/globals.css"), "utf8");
    expect(css).toContain("color-scheme: light;");
    expect(css).toContain("color-scheme: dark;");
    expect(css).toContain(".dark {");
    expect(css).not.toContain("@media (prefers-color-scheme: dark)");
  });

  it("wires ThemeProvider and suppressHydrationWarning in app/layout.tsx", () => {
    const layout = readFileSync(resolve(ROOT, "app/layout.tsx"), "utf8");
    expect(layout).toContain("ThemeProvider");
    expect(layout).toContain("suppressHydrationWarning");
    expect(layout).toContain('attribute="class"');
  });

  it("places ThemeToggle on the landing header only", () => {
    const landingHeader = readFileSync(
      resolve(ROOT, "feature/landing/LandingHeader.tsx"),
      "utf8"
    );
    const dashboardClient = readFileSync(
      resolve(ROOT, "feature/dashboard/HealthDashboardClient.tsx"),
      "utf8"
    );
    expect(landingHeader).toContain("ThemeToggle");
    expect(dashboardClient).not.toContain("ThemeToggle");
  });
});
