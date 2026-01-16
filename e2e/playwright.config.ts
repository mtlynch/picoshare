import type { PlaywrightTestConfig } from "@playwright/test";
import { devices } from "@playwright/test";

const port = Number(process.env.PS_E2E_PORT ?? "6001");
const baseURL = process.env.PS_E2E_BASE_URL ?? `http://localhost:${port}`;

const workers = process.env.PS_E2E_WORKERS
  ? Number(process.env.PS_E2E_WORKERS)
  : 1;

const projectFilter = process.env.PS_E2E_PROJECTS
  ? new Set(
      process.env.PS_E2E_PROJECTS.split(",")
        .map((name) => name.trim())
        .filter(Boolean),
    )
  : null;

const chromiumArgs = [
  "--no-sandbox",
  "--disable-setuid-sandbox",
  "--disable-dev-shm-usage",
  "--disable-gpu",
  "--single-process",
  "--no-zygote",
  ...(process.env.PS_E2E_CHROMIUM_ARGS
    ? process.env.PS_E2E_CHROMIUM_ARGS.split(" ").filter(Boolean)
    : []),
];

const allProjects: PlaywrightTestConfig["projects"] = [
  {
    name: "chromium",
    use: {
      ...devices["Desktop Chrome"],
      launchOptions: {
        args: chromiumArgs,
      },
    },
  },
  {
    name: "firefox",
    use: { ...devices["Desktop Firefox"] },
  },
];

const projects = projectFilter
  ? allProjects.filter((project) =>
      project?.name ? projectFilter.has(project.name) : false,
    )
  : allProjects;

const retries = process.env.PS_E2E_RETRIES
  ? Number(process.env.PS_E2E_RETRIES)
  : process.env.CI
    ? 2
    : 0;

const trace = process.env.PS_E2E_TRACE ?? "on";
const video = process.env.PS_E2E_VIDEO ?? "on";

const config: PlaywrightTestConfig = {
  testDir: "./",
  timeout: 60 * 1000,
  expect: {
    timeout: 5 * 1000,
  },
  fullyParallel: process.env.PS_E2E_FULLY_PARALLEL !== "0",
  forbidOnly: !!process.env.CI,
  // Retry on CI only (can be overridden).
  retries,
  workers,
  reporter: "html",
  globalSetup: require.resolve("./helpers/global-setup"),
  use: {
    baseURL,
    actionTimeout: 0,
    trace: trace as "on" | "off" | "retain-on-failure",
    video: video as "on" | "off" | "retain-on-failure",
  },

  projects,

  outputDir: "results/",

  webServer: {
    command: `PS_SHARED_SECRET=dummypass PORT=${port} ../bin/picoshare-dev`,
    port,
  },
};

export default config;
