import { FullConfig } from "@playwright/test";

async function globalSetup(config: FullConfig) {
  const { baseURL } = config.projects[0].use;

  // Wait for the server to accept requests before enabling per-session DBs.
  const maxAttempts = 30;
  const retryDelayMs = 500;
  let lastError: unknown = null;

  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      const response = await fetch(baseURL + "/api/debug/db/per-session", {
        method: "POST",
      });
      if (response.ok) {
        return;
      }
      lastError = new Error(
        `unexpected response: ${response.status} ${response.statusText}`
      );
    } catch (error) {
      lastError = error;
    }

    await new Promise((resolve) => setTimeout(resolve, retryDelayMs));
  }

  throw lastError;
}

export default globalSetup;
