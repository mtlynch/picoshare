import { test, expect } from "@playwright/test";

test("loads favicon", async ({ page }) => {
  {
    const response = await page.goto("/favicon.ico");
    await expect(response?.status()).toBe(200);
  }

  {
    const response = await page.goto("/android-chrome-192x192.png");
    await expect(response?.status()).toBe(200);
  }

  {
    const response = await page.goto("/apple-touch-icon.png");
    await expect(response?.status()).toBe(200);
  }

  {
    const response = await page.goto("/site.webmanifest");
    await expect(response?.status()).toBe(200);
  }
});
