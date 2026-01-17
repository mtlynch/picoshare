import type { PlaywrightTestConfig } from "@playwright/test";
import { devices } from "@playwright/test";
import { fileURLToPath } from "node:url";

const config: PlaywrightTestConfig = {
  testDir: "./e2e",
  timeout: 15 * 1000,
  expect: {
    timeout: 5 * 1000,
  },
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: "html",
  globalSetup: fileURLToPath(
    new URL("./e2e/helpers/global-setup", import.meta.url)
  ),
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
  ],

  outputDir: "e2e-results/",

  webServer: {
    command: "PS_SHARED_SECRET=dummypass PORT=6001 TZ=UTC ./bin/picoshare-dev",
    port: 6001,
  },
};

export default config;
