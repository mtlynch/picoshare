import { test, expect } from "@playwright/test";
import { login } from "./helpers/login.js";

const browserColumn = 3;

test("upload a file and verify it has no download history", async ({
  page,
}) => {
  await login(page);

  await page.locator(".file-input").setInputFiles([
    {
      name: "simple-upload.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("I expect zero downloads"),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );

  await page.getByRole("menuitem", { name: "Files" }).click();

  await expect(page).toHaveURL(/\/files$/);
  await page
    .getByRole("row")
    .filter({ hasText: "simple-upload.txt" })
    .getByRole("button", { name: "Information" })
    .click();

  await expect(page).toHaveURL(/\/files\/.+\/info$/);
  await page
    .locator("section")
    .filter({ has: page.getByRole("heading", { name: "Downloads" }) })
    .getByRole("link", { name: "History" })
    .click();

  await expect(page).toHaveURL(/\/files\/.+\/downloads$/);
  await expect(
    page.getByRole("heading", { name: "simple-upload.txt" })
  ).toBeVisible();

  await expect(page.getByText("No downloads yet.")).toBeVisible();
});

test("upload a file, download it, and verify it has a download history", async ({
  page,
  browserName,
}) => {
  await login(page);

  await page.locator(".file-input").setInputFiles([
    {
      name: "simple-upload.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("I expect one download"),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );

  await page.locator("#result-links a").first().click();

  await expect(page.getByText("I expect one download")).toBeVisible();

  await page.goBack();

  await page.getByRole("menuitem", { name: "Files" }).click();

  await expect(page).toHaveURL(/\/files$/);
  await page
    .getByRole("row")
    .filter({ hasText: "simple-upload.txt" })
    .getByRole("button", { name: "Information" })
    .click();

  await expect(page).toHaveURL(/\/files\/.+\/info$/);
  await expect(
    page
      .locator("section")
      .filter({ has: page.getByRole("heading", { name: "Downloads" }) })
      .locator(".value")
  ).toHaveText("1 (History)");
  await page
    .locator("section")
    .filter({ has: page.getByRole("heading", { name: "Downloads" }) })
    .getByRole("link", { name: "History" })
    .click();

  await expect(page).toHaveURL(/\/files\/.+\/downloads$/);
  await expect(
    page.getByRole("heading", { name: "simple-upload.txt" })
  ).toBeVisible();

  // We expect one header row + one body row.
  await expect(page.getByRole("table").getByRole("row")).toHaveCount(2);

  const browserNameExpected = (() => {
    if (browserName === "chromium") {
      return "Chrome";
    }
    if (browserName == "firefox") {
      return "Firefox";
    }
    return null;
  })();

  await expect(
    page
      .getByRole("table")
      .getByRole("row")
      .last()
      .getByRole("cell")
      .nth(browserColumn)
  ).toHaveText(browserNameExpected);
});
