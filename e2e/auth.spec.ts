import { test, expect } from "@playwright/test";

test("logs in and logs out", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("menuitem", { name: "Log In" }).click();

  await expect(page).toHaveURL("/login");
  await page.locator("form input[type='password']").fill("dummypass");
  await page.locator("form input[type='submit']").click();

  await expect(page).toHaveURL("/");

  await page.getByRole("menuitem", { name: "System" }).hover();
  await page.getByRole("menuitem", { name: "Log Out" }).click();

  await expect(page.getByRole("menuitem", { name: "Log In" })).toBeVisible();
});
