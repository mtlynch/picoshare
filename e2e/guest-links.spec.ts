import { test, expect } from "@playwright/test";
import { wipeDB } from "./helpers/db.js";
import { login } from "./helpers/login.js";

test.beforeEach(async ({ page }) => {
  await wipeDB(page);
});

test("creates a guest link and uploads a file as a guest", async ({ page }) => {
  await login(page);

  await page.locator("nav .navbar-item[href='/guest-links']").click();

  await page.locator(".content .button.is-primary").click();

  await expect(page).toHaveURL("/guest-links/new");
  await page.locator("#label").fill("For e2e testing");
  await page.locator("#max-file-size").fill("50");
  await page.locator("#file-upload-limit").fill("1");
  await page.locator("#create-guest-link-form input[type='submit']").click();

  await expect(page).toHaveURL("/guest-links");
  const guestLinkElement = page.locator(
    '.table tbody tr:first-child td[test-data-id="guest-link-label"] a',
    {
      hasText: "For e2e testing",
    }
  );
  expect(guestLinkElement).toBeVisible();

  // Save the route to the guest link URL so that we can return to it later.
  const guestLinkRouteValue = await guestLinkElement.getAttribute("href");
  expect(guestLinkRouteValue).not.toBeNull();
  const guestLinkRoute = String(guestLinkRouteValue);

  await page.locator(".navbar-end .navbar-item.is-hoverable").hover();
  await page.locator("#navbar-log-out").click();

  await expect(page).toHaveURL("/");
  await page.goto(guestLinkRoute);
  await page.locator(".file-input").setInputFiles([
    {
      name: "guest-link-upload.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("uploaded by a guest user"),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );

  await expect(page.locator("#upload-result upload-links")).toHaveAttribute(
    "filename",
    "guest-link-upload.txt"
  );
  await expect(
    page.locator("#upload-result upload-links #verbose-link-box #link")
  ).toBeVisible();
  await expect(
    page.locator("#upload-result upload-links #short-link-box #link")
  ).toBeVisible();

  await page.locator("#upload-another-btn").click();

  await expect(page.locator("h1")).toContainText("Guest Link Inactive");
  await expect(page.locator(".file-input")).toHaveCount(0);
});
