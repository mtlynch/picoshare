import type { PlaywrightTestConfig } from "@playwright/test";
import { devices } from "@playwright/test";

const config: PlaywrightTestConfig = {
  testDir: "./",
  timeout: 60 * 1000,
  expect: {
    timeout: 5 * 1000,
  },
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: undefined,
  reporter: "html",
  globalSetup: require.resolve("./helpers/global-setup"),
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

  outputDir: "results/",

  webServer: {
    command: "PS_SHARED_SECRET=dummypass PORT=6001 ../bin/picoshare-dev",
    port: 6001,
  },
};

export default config;
