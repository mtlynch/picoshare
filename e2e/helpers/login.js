import { expect } from "@playwright/test";

export async function login(page) {
  await page.goto("/");
  await page.locator("data-test-id=log-in").click();

  await expect(page).toHaveURL("/login");
  await page.locator("form input[type='password']").fill("dummypass");
  await page.locator("form input[type='submit']").click();
  await expect(page).toHaveURL("/");
}
