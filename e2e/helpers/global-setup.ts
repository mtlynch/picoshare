import fetch from "isomorphic-fetch";

import { FullConfig } from "@playwright/test";

async function globalSetup(config: FullConfig) {
  const { baseURL } = config.projects[0].use;

  // Tell PicoShare to enable per-session databases so that tests results stay
  // independent.
  await fetch(baseURL + "/api/debug/db/per-session", { method: "POST" });
}

export default globalSetup;
