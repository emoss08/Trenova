import { defineConfig, devices } from "@playwright/test";
import path from "node:path";
import { BASE_URL, CHROMIUM_PATH, E2E_DIR, STORAGE_STATE } from "./env";

/**
 * End-to-end tests for the Desk, run against a stack that is already up: the
 * API and worker, the web dev server, and the scripted mock model in
 * e2e/mockllm. Nothing here starts them; see e2e/README.md.
 *
 * E2E_BASE_URL        the web app (default http://localhost:5173)
 * E2E_EMAIL           who to sign in as (default the development seed's admin)
 * E2E_PASSWORD        their password (default the development seed's)
 * E2E_STORAGE_STATE   a saved, signed-in storage state to reuse instead of signing in
 * E2E_CHROMIUM_PATH   a Chromium binary to launch instead of Playwright's own download
 * E2E_COMMIT=1        let approvals go through rather than undoing them
 */
const launchOptions = { executablePath: CHROMIUM_PATH };

export default defineConfig({
  testDir: E2E_DIR,
  testMatch: /.*\.spec\.ts$/,
  outputDir: path.join(E2E_DIR, ".results"),
  globalSetup: path.join(E2E_DIR, "global-setup.ts"),
  // One stack, one mock and one database: the flows share them, so they run
  // one at a time.
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  forbidOnly: Boolean(process.env.CI),
  timeout: 120_000,
  expect: { timeout: 15_000 },
  reporter: [["list"], ["html", { outputFolder: path.join(E2E_DIR, ".report"), open: "never" }]],
  use: {
    baseURL: BASE_URL,
    storageState: STORAGE_STATE,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "off",
  },
  projects: [
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"],
        viewport: { width: 1440, height: 900 },
        launchOptions,
      },
    },
  ],
});
