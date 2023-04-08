import type { PlaywrightTestConfig } from "@playwright/test";
import { devices } from "@playwright/test";

const config: PlaywrightTestConfig = {
  testDir: "./e2e",
  timeout: 30 * 1000,
  expect: {
    timeout: 5 * 1000,
  },
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: "html",
  use: {
    baseURL: "http://localhost:6001",
    actionTimeout: 0,
    trace: "on",
    video: "on",
  },

  projects: [
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"],
      },
    },
    {
      name: "firefox",
      use: { ...devices["Desktop Firefox"] },
    },
  ],

  outputDir: "e2e-results/",

  webServer: {
    command:
      "PS_SHARED_SECRET=dummypass PORT=6001 ./bin/picoshare-dev -db data/store-e2e.db",
    port: 6001,
  },
};

export default config;
