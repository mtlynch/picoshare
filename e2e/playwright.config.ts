import os from "node:os";
import type { PlaywrightTestConfig } from "@playwright/test";
import { devices } from "@playwright/test";

// Sandboxed builders may allot fewer cores than the host reports, so trust the
// builder's declared capacity when it's available.
const declaredCores = Number(process.env.NIX_BUILD_CORES);
const availableCores =
  Number.isInteger(declaredCores) && declaredCores > 0
    ? declaredCores
    : os.availableParallelism();

const config: PlaywrightTestConfig = {
  testDir: "./",
  timeout: 60 * 1000,
  expect: {
    timeout: 5 * 1000,
  },
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  // Retry on CI only.
  retries: process.env.CI ? 2 : 0,
  // Concurrent browsers starve each other before the five-second assertion
  // timeout when workers exceed the available cores.
  workers: Math.min(availableCores, 4),
  reporter: "html",
  use: {
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
};

export default config;
