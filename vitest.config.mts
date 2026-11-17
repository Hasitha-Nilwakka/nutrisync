import { defineConfig } from "vitest/config";
import { resolve } from "node:path";

export default defineConfig({
  resolve: {
    alias: {
      "server-only": resolve(__dirname, "tests/mocks/empty.ts"),
    },
  },

  test: {
    environment: "node",
    setupFiles: ["/tests/vitest.setup.ts"],
    include: ["tests/**/*.test.ts"],
    exclude: ["e2e/**", "node_modules/**"],
  },
});