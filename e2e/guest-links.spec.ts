import { test, expect } from "@playwright/test";
import { login } from "./helpers/login.js";

const labelColumn = 0;

test("creates a guest link and uploads a file as a guest", async ({ page }) => {
  await login(page);

  await page.getByRole("menuitem", { name: "Guest Links" }).click();

  await page.getByRole("button", { name: "Create new" }).click();

  await expect(page).toHaveURL("/guest-links/new");
  await page.locator("#label").fill("For e2e testing");
  await page.locator("#max-file-size").fill("50");
  await page.locator("#file-upload-limit").fill("1");
  await page.getByRole("button", { name: "Create" }).click();

  await expect(page).toHaveURL("/guest-links");
  const guestLinkRow = await page
    .getByRole("row")
    .filter({ hasText: "For e2e testing" });
  await expect(guestLinkRow).toBeVisible();

  // Save the route to the guest link URL so that we can return to it later.
  const guestLinkRouteValue = await guestLinkRow
    .getByRole("cell")
    .nth(labelColumn)
    .getByRole("link")
    .getAttribute("href");
  expect(guestLinkRouteValue).not.toBeNull();
  const guestLinkRoute = String(guestLinkRouteValue);

  await page.getByRole("menuitem", { name: "System" }).hover();
  await page.getByRole("menuitem", { name: "Log Out" }).click();

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

  await page.getByRole("button", { name: "Upload Another" }).click();

  await expect(page.locator("h1")).toContainText("Guest Link Inactive");
  await expect(page.locator(".file-input")).toHaveCount(0);
});
