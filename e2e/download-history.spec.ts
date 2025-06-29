import { test, expect } from "@playwright/test";
import { login } from "./helpers/login";

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
    .getByLabel("Information")
    .click();

  await expect(page).toHaveURL(/\/files\/.+\/info$/);
  await expect(page.getByText("(History)")).toHaveText("0 (History)");
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
    .getByLabel("Information")
    .click();

  await expect(page).toHaveURL(/\/files\/.+\/info$/);
  await expect(page.getByText("(History)")).toHaveText("1 (History)");
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
    return "Unknown";
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

test("filter downloads to unique IPs only", async ({ page }) => {
  await login(page);

  // Upload a file
  await page.locator(".file-input").setInputFiles([
    {
      name: "multi-download.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("File for testing unique IP filtering"),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );

  await page.locator("#result-links a").first().click();
  await expect(
    page.getByText("File for testing unique IP filtering")
  ).toBeVisible();
  // Force a second download.
  await page.reload();

  // Navigate to downloads history
  await page.goBack();
  await page.getByRole("menuitem", { name: "Files" }).click();
  await expect(page).toHaveURL(/\/files$/);
  await page
    .getByRole("row")
    .filter({ hasText: "multi-download.txt" })
    .getByLabel("Information")
    .click();

  await expect(page).toHaveURL(/\/files\/.+\/info$/);
  await page
    .locator("section")
    .filter({ has: page.getByRole("heading", { name: "Downloads" }) })
    .getByRole("link", { name: "History" })
    .click();

  await expect(page).toHaveURL(/\/files\/.+\/downloads$/);

  // Verify we have 2 total downloads.
  // Header row + 2 body rows = 3 total rows.
  await expect(page.getByRole("table").getByRole("row")).toHaveCount(3);

  // Verify the checkbox exists and is unchecked by default.
  const uniqueCheckbox = page.locator("#unique-ips-only");
  await expect(uniqueCheckbox).toBeVisible();
  await expect(uniqueCheckbox).not.toBeChecked();

  // Check the "Unique IPs only" checkbox.
  await uniqueCheckbox.check();

  // Verify the page reloads with the unique parameter.
  await expect(page).toHaveURL(/\/files\/.+\/downloads\?unique=true$/);

  // Verify the checkbox is now checked (state preserved).
  await expect(uniqueCheckbox).toBeChecked();

  // Verify we now have only 1 download (unique IP filtering).
  // Header row + 1 body row = 2 total rows.
  await expect(page.getByRole("table").getByRole("row")).toHaveCount(2);

  // Verify download number is correct (should show "1").
  await expect(
    page.getByRole("table").getByRole("row").nth(2).getByRole("cell").first()
  ).toHaveText("1");

  // Uncheck the checkbox.
  await uniqueCheckbox.uncheck();

  // Verify we're back to showing all downloads.
  await expect(page).toHaveURL(/\/files\/.+\/downloads$/);
  await expect(page.getByRole("table").getByRole("row")).toHaveCount(3);
  await expect(uniqueCheckbox).not.toBeChecked();
});
