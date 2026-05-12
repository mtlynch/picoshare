import { expect, Page } from "@playwright/test";

export async function login(page: Page): Promise<void> {
  await page.goto("/");
  await page.getByRole("menuitem", { name: "Log In" }).click();

  await expect(page).toHaveURL("/login");
  await page.locator("form input[type='password']").fill("dummypass");
  await page.locator("form input[type='submit']").click();
  await expect(page).toHaveURL("/");
}
