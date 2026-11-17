import { defineConfig, devices } from "@playwright/test"

/**
 * Playwright E2E Configuration for NutriSync Next.js App Router.
 * Configured with hermetic mock Go API server (port 8088),
 * Next.js dev server on port 3000, and Chromium browser project.
 */
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : 2,
  reporter: [["html", { open: "never" }], ["list"]],
  use: {
    baseURL: process.env.PLAYWRIGHT_TEST_BASE_URL || "http://127.0.0.1:3000",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: [
    {
      command: "node e2e/mockGoServer.mjs",
      url: "http://127.0.0.1:8088/api/auth/session",
      reuseExistingServer: !process.env.CI,
      timeout: 15_000,
    },
    {
      command: "__NEXT_AGENT_UPGRADE='' GO_API_URL=http://127.0.0.1:8088 pnpm dev",
      url: "http://127.0.0.1:3000",
      reuseExistingServer: true,
      timeout: 120_000,
    },
  ],
})
