import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/e2e",
  timeout: 60_000,
  fullyParallel: false,
  workers: 1,
  reporter: [["list"]],
  use: {
    baseURL: "http://localhost:3000",
    trace: "retain-on-failure",
  },
  webServer: [
    {
      command: "go run ./cmd/api",
      cwd: "../backend",
      port: 8080,
      reuseExistingServer: !process.env.CI,
      env: {
        ...process.env,
        PORT: "8080",
        APP_ENV: "development",
        DATABASE_URL:
          "postgres://splitmate:splitmate@localhost:5433/splitmate?sslmode=disable",
        JWT_SECRET: "dev-only-secret-change-me",
      },
    },
    {
      command: "bun run dev",
      cwd: ".",
      port: 3000,
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
    },
  ],
});
