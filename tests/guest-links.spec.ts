import { test, expect } from "@playwright/test";

test("creates a guest link and uploads a file as a guest", async ({ page }) => {
  await page.goto("/");

  await page.locator("data-test-id=log-in").click();

  await expect(page).toHaveURL("/login");
  await page.locator("form input[type='password']").fill("dummypass");
  await page.locator("form input[type='submit']").click();

  await expect(page).toHaveURL("/");
  await page.locator("nav .navbar-item[href='/guest-links']").click();

  await page.locator(".content .button.is-primary").click();

  await expect(page).toHaveURL("/guest-links/new");
  await page.locator("#label").fill("For e2e testing");
  await page.locator("#max-file-size").fill("50");
  await page.locator("#file-upload-limit").fill("1");
  await page.locator("#create-guest-link-form input[type='submit']").click();

  await expect(page).toHaveURL("/guest-links");
  const guestLinkElement = page.locator(
    '.table td[test-data-id="guest-link-label"] a',
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

  await page.goto(guestLinkRoute);
  await page
    .locator(".file-input")
    .setInputFiles(["./tests/testdata/kittyface.jpg"]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );

  await expect(page.locator("#upload-result upload-links")).toHaveAttribute(
    "filename",
    "kittyface.jpg"
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
