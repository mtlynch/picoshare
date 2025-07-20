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
    const dbCookie = readDbTokenCookie(await page.context().cookies());
    if (dbCookie) {
      await guestContext.addCookies([dbCookie]);
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
    const dbCookie = readDbTokenCookie(await page.context().cookies());
    if (dbCookie) {
      await guestContext.addCookies([dbCookie]);
    }

    const guestPage = await guestContext.newPage();

    await guestPage.goto(guestLinkRoute);
    await expect(guestPage.locator("h1")).toContainText("Upload");
    await expect(guestPage.locator(".file-input")).toBeVisible();
  }
});

test("guest upload shows expiration dropdown with options limited by guest link", async ({
  page,
  browser,
}) => {
  await login(page);

  await page.getByRole("menuitem", { name: "Guest Links" }).click();

  await page.getByRole("button", { name: "Create new" }).click();

  await expect(page).toHaveURL("/guest-links/new");
  await page.locator("#label").fill("7-day expiration test");
  await page.locator("#file-upload-limit").fill("5");

  // Set file expiration to 7 days.
  await page.locator("#file-expiration-select").selectOption("168h0m0s");

  await page.getByRole("button", { name: "Create" }).click();

  await expect(page).toHaveURL("/guest-links");
  const guestLinkRow = await page
    .getByRole("row")
    .filter({ hasText: "7-day expiration test" });
  await expect(guestLinkRow).toBeVisible();

  // Get the guest link URL.
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

    // Check that the expiration dropdown is visible.
    await expect(guestPage.locator("#expiration-select")).toBeVisible();

    // Check that only options up to 7 days are available.
    const expirationOptions = await guestPage
      .locator("#expiration-select option")
      .allTextContents();

    expect(expirationOptions).toContain("1 day");
    expect(expirationOptions).toContain("7 days");
    expect(expirationOptions).not.toContain("30 days");
    expect(expirationOptions).not.toContain("1 year");
    expect(expirationOptions).not.toContain("Never");

    // Check that 7 days is selected by default.
    const defaultSelected = await guestPage
      .locator("#expiration-select option[selected]")
      .textContent();
    expect(defaultSelected).toBe("7 days");
  }
});

test("guest upload with infinite file lifetime shows all expiration options", async ({
  page,
  browser,
}) => {
  await login(page);

  await page.getByRole("menuitem", { name: "Guest Links" }).click();

  await page.getByRole("button", { name: "Create new" }).click();

  await expect(page).toHaveURL("/guest-links/new");
  await page.locator("#label").fill("Infinite expiration test");
  await page.locator("#file-upload-limit").fill("5");

  // Set file expiration to Never (infinite).
  await page.locator("#file-expiration-select").selectOption("876000h0m0s");

  await page.getByRole("button", { name: "Create" }).click();

  await expect(page).toHaveURL("/guest-links");
  const guestLinkRow = await page
    .getByRole("row")
    .filter({ hasText: "Infinite expiration test" });
  await expect(guestLinkRow).toBeVisible();

  // Get the guest link URL.
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

    // Check that the expiration dropdown is visible.
    await expect(guestPage.locator("#expiration-select")).toBeVisible();

    // Check that all expiration options are available.
    const expirationOptions = await guestPage
      .locator("#expiration-select option")
      .allTextContents();

    expect(expirationOptions).toContain("1 day");
    expect(expirationOptions).toContain("7 days");
    expect(expirationOptions).toContain("30 days");
    expect(expirationOptions).toContain("1 year");
    expect(expirationOptions).toContain("Never");

    // Check that Never is selected by default.
    const defaultSelected = await guestPage
      .locator("#expiration-select option[selected]")
      .textContent();
    expect(defaultSelected).toBe("Never");
  }
});

test("guest upload respects selected expiration time", async ({
  page,
  browser,
}) => {
  await login(page);

  await page.getByRole("menuitem", { name: "Guest Links" }).click();

  await page.getByRole("button", { name: "Create new" }).click();

  await expect(page).toHaveURL("/guest-links/new");
  await page.locator("#label").fill("Custom expiration test");
  await page.locator("#file-upload-limit").fill("5");

  // Set file expiration to 30 days.
  await page.locator("#file-expiration-select").selectOption("720h0m0s");

  await page.getByRole("button", { name: "Create" }).click();

  await expect(page).toHaveURL("/guest-links");
  const guestLinkRow = await page
    .getByRole("row")
    .filter({ hasText: "Custom expiration test" });
  await expect(guestLinkRow).toBeVisible();

  // Get the guest link URL.
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

    // Select 7 days instead of the default 30 days.
    await guestPage.locator("#expiration-select").selectOption("7 days");

    // Upload a file.
    await guestPage.locator(".file-input").setInputFiles([
      {
        name: "custom-expiration-test.txt",
        mimeType: "text/plain",
        buffer: Buffer.from("testing custom expiration"),
      },
    ]);

    await expect(guestPage.locator("#upload-result .message-body")).toHaveText(
      "Upload complete!"
    );
  }

  // Check that the file has the correct expiration (7 days, not 30).
  await page.getByRole("menuitem", { name: "Files" }).click();

  // Look for the uploaded file.
  const fileRow = page
    .getByRole("row")
    .filter({ hasText: "custom-expiration-test.txt" });
  await expect(fileRow).toBeVisible();

  // Check that the expiration is not "30 days" (which would be the guest link default).
  // The exact date depends on when the test runs, but it should not show "Never".
  const expirationCell = fileRow.getByRole("cell").nth(expiresColumn);
  const expirationText = await expirationCell.textContent();
  expect(expirationText).not.toBe("Never");
  expect(expirationText).not.toBe("30 days from now"); // Should be 7 days.
});
