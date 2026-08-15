import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/e2e",
  testMatch: "prod-smoke.spec.ts",
  use: {
    baseURL: "https://splitmate-phi.vercel.app",
    trace: "on-first-retry",
  },
  workers: 1,
  timeout: 120000,
});