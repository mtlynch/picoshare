import { test, expect } from "@playwright/test";
import { login } from "./helpers/login";
import { readDbTokenCookie } from "./helpers/db";

const labelColumn = 0;
const expiresColumn = 4;

test("creates a guest link and uploads a file as a guest", async ({
  page,
  browser,
}) => {
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

  {
    const guestContext = await browser.newContext();

    // Share database across users
    const dbCookie = readDbTokenCookie(await page.context().cookies());
    if (dbCookie) {
      await guestContext.addCookies([dbCookie]);
    }

    const guestPage = await guestContext.newPage();

    await guestPage.goto(guestLinkRoute);
    await guestPage.locator(".file-input").setInputFiles([
      {
        name: "guest-link-upload.txt",
        mimeType: "text/plain",
        buffer: Buffer.from("uploaded by a guest user"),
      },
    ]);
    await expect(guestPage.locator("#upload-result .message-body")).toHaveText(
      "Upload complete!"
    );

    await expect(
      guestPage.locator("#upload-result upload-links")
    ).toHaveAttribute("filename", "guest-link-upload.txt");
    await expect(
      guestPage.locator("#upload-result upload-links #verbose-link-box #link")
    ).toBeVisible();
    await expect(
      guestPage.locator("#upload-result upload-links #short-link-box #link")
    ).toBeVisible();

    await guestPage.getByRole("button", { name: "Upload Another" }).click();

    await expect(guestPage.locator("h1")).toContainText("Guest Link Inactive");
    await expect(guestPage.locator(".file-input")).toHaveCount(0);
  }
  await page.getByRole("menuitem", { name: "Files" }).click();
  await expect(
    page
      .getByRole("row")
      .filter({ hasText: "guest-link-upload.txt" })
      .getByRole("cell")
      .nth(expiresColumn)
  ).toHaveText("Never");
});

test("files uploaded through guest link remain accessible after guest link is deleted", async ({
  page,
  browser,
}) => {
  await login(page);

  await page.getByRole("menuitem", { name: "Guest Links" }).click();

  await page.getByRole("button", { name: "Create new" }).click();

  await expect(page).toHaveURL("/guest-links/new");
  await page.locator("#label").fill("I'll be deleted soon");
  await page.locator("#file-upload-limit").fill("1");
  await page.getByRole("button", { name: "Create" }).click();

  await expect(page).toHaveURL("/guest-links");
  const guestLinkRow = await page
    .getByRole("row")
    .filter({ hasText: "I'll be deleted soon" });
  await expect(guestLinkRow).toBeVisible();

  // Save the route to the guest link URL so that we can return to it later.
  const guestLinkRouteValue = await guestLinkRow
    .getByRole("cell")
    .nth(labelColumn)
    .getByRole("link")
    .getAttribute("href");
  expect(guestLinkRouteValue).not.toBeNull();
  const guestLinkRoute = String(guestLinkRouteValue);

  {
    const guestContext = await browser.newContext();

    // Share database across users.
    const dbCookie = readDbTokenCookie(await page.context().cookies());
    if (dbCookie) {
      await guestContext.addCookies([dbCookie]);
    }

    const guestPage = await guestContext.newPage();

    await guestPage.goto(guestLinkRoute);
    await guestPage.locator(".file-input").setInputFiles([
      {
        name: "guest-upload2.txt",
        mimeType: "text/plain",
        buffer: Buffer.from("uploaded by a guest user"),
      },
    ]);
    await expect(guestPage.locator("#upload-result .message-body")).toHaveText(
      "Upload complete!"
    );
  }

  await guestLinkRow.getByRole("button", { name: "Delete" }).click();

  await page.getByRole("menuitem", { name: "Files" }).click();

  const filenameColumn = 0;
  await page
    .getByRole("row")
    .filter({ hasText: "guest-upload2.txt" })
    .getByRole("cell")
    .nth(filenameColumn)
    .getByRole("link")
    .click();

  await expect(page.locator("body")).toHaveText("uploaded by a guest user");
});

test("invalid options on guest link generate error message", async ({
  page,
}) => {
  await login(page);

  await page.getByRole("menuitem", { name: "Guest Links" }).click();

  await page.getByRole("button", { name: "Create new" }).click();

  await expect(page).toHaveURL("/guest-links/new");
  await page.locator("#label").fill("A".repeat(5000));
  await page.getByRole("button", { name: "Create" }).click();

  // We should still be on the same page.
  await expect(page).toHaveURL("/guest-links/new");

  // There should be an error message
  await expect(page.getByText("Invalid request: label too long")).toBeVisible();
});

test("disables and enables a guest link, affecting access", async ({
  page,
  browser,
}) => {
  await login(page);

  await page.getByRole("menuitem", { name: "Guest Links" }).click();

  await page.getByRole("button", { name: "Create new" }).click();

  await expect(page).toHaveURL("/guest-links/new");
  await page.locator("#label").fill("test guest link enable/disable");
  await page.locator("#file-upload-limit").fill("1");
  await page.getByRole("button", { name: "Create" }).click();

  await expect(page).toHaveURL("/guest-links");
  const guestLinkRow = await page
    .getByRole("row")
    .filter({ hasText: "test guest link enable/disable" });
  await expect(guestLinkRow).toBeVisible();

  // Save the route to the guest link URL so that we can return to it later.
  const guestLinkRouteValue = await guestLinkRow
    .getByRole("cell")
    .nth(labelColumn)
    .getByRole("link")
    .getAttribute("href");
  expect(guestLinkRouteValue).not.toBeNull();
  const guestLinkRoute = String(guestLinkRouteValue);

  // Disable the guest link.
  await guestLinkRow.getByRole("button", { name: "Disable" }).click();

  await expect(
    guestLinkRow.getByRole("button", { name: "Copy" })
  ).not.toBeVisible();

  // Try to access the guest link as a guest user.
  {
    const guestContext = await browser.newContext();

    // Share database across users.
    const dbCookie1 = readDbTokenCookie(await page.context().cookies());
    if (dbCookie1) {
      await guestContext.addCookies([dbCookie1]);
    }

    const guestPage = await guestContext.newPage();

    await guestPage.goto(guestLinkRoute);
    await expect(guestPage.locator("h1")).toContainText("Guest Link Inactive");
    await expect(guestPage.locator(".file-input")).toHaveCount(0);
  }

  // Enable the guest link.
  await guestLinkRow.getByRole("button", { name: "Enable" }).click();

  await expect(
    guestLinkRow.getByRole("button", { name: "Copy" })
  ).toBeVisible();

  // Try to access the guest link as a guest user.
  {
    const guestContext = await browser.newContext();

    // Share database across users.
    const dbCookie2 = readDbTokenCookie(await page.context().cookies());
    if (dbCookie2) {
      await guestContext.addCookies([dbCookie2]);
    }

    const guestPage = await guestContext.newPage();

    await guestPage.goto(guestLinkRoute);
    await expect(guestPage.locator("h1")).toContainText("Upload");
    await expect(guestPage.locator(".file-input")).toBeVisible();
  }
});
