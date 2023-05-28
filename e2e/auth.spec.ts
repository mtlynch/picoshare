import { test, expect } from "@playwright/test";

test("logs in and logs out", async ({ page }) => {
  await page.goto("/");

  await page.getByTestId("log-in").click();

  await expect(page).toHaveURL("/login");
  await page.locator("form input[type='password']").fill("dummypass");
  await page.locator("form input[type='submit']").click();

  await expect(page).toHaveURL("/");
  await page.getByTestId("system-dropdown").hover();
  await page.locator("#navbar-log-out").click();

  await expect(
    page.locator(".navbar-item .button[data-testid='log-in']")
  ).toBeVisible();
});
