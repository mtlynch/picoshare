import { test, expect } from "@playwright/test";
import { login } from "./helpers/login.js";

test("default file expiration is 30 days", async ({ page }) => {
  await login(page);

  await page.locator("data-test-id=system-dropdown").hover();
  await page.locator("a[href='/settings']").click();
  await expect(page).toHaveURL("/settings");

  await expect(page.locator("#default-expiration")).toHaveValue("30");
  await expect(page.locator("#time-unit")).toHaveValue("days");

  await page.locator("data-test-id=upload-btn").click();
  await expect(page).toHaveURL("/");

  await expect(page.locator("#expiration-select option[selected]")).toHaveText(
    "30 days"
  );
});

test("changes default file expiration to 5 days", async ({ page }) => {
  await login(page);

  await page.locator("data-test-id=system-dropdown").hover();
  await page.locator("a[href='/settings']").click();
  await expect(page).toHaveURL("/settings");

  await page.locator("#default-expiration").fill("5");
  await page.locator("#settings-form button[type='submit']").click();

  await page.locator("data-test-id=upload-btn").click();
  await expect(page).toHaveURL("/");

  await expect(page.locator("#expiration-select option[selected]")).toHaveText(
    "5 days"
  );
});

test("changes default file expiration to 1 year", async ({ page }) => {
  await login(page);

  await page.locator("data-test-id=system-dropdown").hover();
  await page.locator("a[href='/settings']").click();
  await expect(page).toHaveURL("/settings");

  await page.locator("#default-expiration").fill("1");
  await page.locator("#time-unit").selectOption("years");
  await page.locator("#settings-form button[type='submit']").click();

  await page.locator("data-test-id=upload-btn").click();
  await expect(page).toHaveURL("/");

  await expect(page.locator("#expiration-select option[selected]")).toHaveText(
    "1 year"
  );

  // Because 1 year is one of the built-in defaults, we shouldn't see any
  // additional items in the options list.
  const expirationOptions = await page
    .locator("#expiration-select option")
    .allInnerTexts();
  expect(expirationOptions[0]).toEqual("1 day");
  expect(expirationOptions[1]).toEqual("7 days");
  expect(expirationOptions[2]).toEqual("30 days");
  expect(expirationOptions[3]).toEqual("1 year");
  expect(expirationOptions[4]).toEqual("Never");
  expect(expirationOptions[5]).toEqual("Custom");
});

test("changes default file expiration to 10 years", async ({ page }) => {
  await login(page);

  await page.locator("data-test-id=system-dropdown").hover();
  await page.locator("a[href='/settings']").click();
  await expect(page).toHaveURL("/settings");

  // Change default expiration to 10 years.
  await page.locator("#default-expiration").fill("10");
  await page.locator("#time-unit").selectOption("years");
  await page.locator("#settings-form button[type='submit']").click();

  await page.locator("data-test-id=upload-btn").click();
  await expect(page).toHaveURL("/");

  await expect(page.locator("#expiration-select option[selected]")).toHaveText(
    "10 years"
  );
});
