import { test, expect } from "@playwright/test";
import { login } from "./helpers/login.js";

const noteColumn = 1;
const expirationColumn = 4;

test("uploads a file without specifying any parameters", async ({
  page,
  request,
}) => {
  await login(page);

  await page.locator(".file-input").setInputFiles([
    {
      name: "simple-upload.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("I'm just a simple upload"),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );
  await expect(page.locator("#upload-result upload-links")).toHaveAttribute(
    "filename",
    "simple-upload.txt"
  );
  await expect(
    page.locator("#upload-result upload-links #verbose-link-box #link")
  ).toBeVisible();
  await expect(
    page.locator("#upload-result upload-links #short-link-box #link")
  ).toBeVisible();

  // Verify that cleanup doesn't incorrectly remove the file.
  await request.post("/api/debug/db/cleanup");

  await page.getByRole("menuitem", { name: "Files" }).click();
  const matchingRow = await page
    .getByRole("row")
    .filter({ hasText: "simple-upload.txt" });
  await expect(matchingRow).toBeVisible();
  await expect(matchingRow.getByRole("cell").nth(noteColumn)).toBeEmpty();
});

test("uploads a file with a custom expiration time", async ({ page }) => {
  await login(page);

  await page.locator("#expiration-select").selectOption({ label: "Custom" });

  await page.locator("#expiration-picker #expiration").fill("2029-09-03");
  // Move focus to note field just to so the expiration date saves.
  await page.locator("#note").click();

  await page.locator(".file-input").setInputFiles([
    {
      name: "custom-expiration-upload.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("I have a custom expiration time"),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );

  await page.getByRole("menuitem", { name: "Files" }).click();

  const matchingRow = await page
    .getByRole("row")
    .filter({ hasText: "custom-expiration-upload.txt" });
  await expect(matchingRow).toBeVisible();
  await expect(matchingRow.getByRole("cell").nth(noteColumn)).toBeEmpty();
  await expect(matchingRow.getByRole("cell").nth(expirationColumn)).toHaveText(
    /^2029-09-03/
  );
});

test("uploads a file with a note", async ({ page }) => {
  await login(page);

  await page.locator("#note").fill("For Pico, with Love and Squalor");

  await page.locator(".file-input").setInputFiles([
    {
      name: "upload-with-note.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("I'm an upload with a note"),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );

  await page.getByRole("menuitem", { name: "Files" }).click();

  const matchingRow = await page
    .getByRole("row")
    .filter({ hasText: "upload-with-note.txt" });
  await expect(matchingRow).toBeVisible();
  await expect(matchingRow.getByRole("cell").nth(noteColumn)).toHaveText(
    "For Pico, with Love and Squalor"
  );
});

test("uploads a file and deletes it", async ({ page }) => {
  await login(page);

  await page.locator(".file-input").setInputFiles([
    {
      name: "upload-for-deletion.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("I'm an upload that will soon be deleted"),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );

  await page.getByRole("menuitem", { name: "Files" }).click();

  await page
    .getByRole("row")
    .filter({ hasText: "upload-for-deletion.txt" })
    .getByRole("button")
    .filter({ has: page.locator(".fa-edit") })
    .click();

  await expect(page).toHaveURL(/\/files\/.+\/edit$/);
  await page.getByRole("link", { name: "Delete" }).click();

  await expect(page).toHaveURL(/\/files\/.+\/confirm-delete$/);
  await page.getByRole("button", { name: "Delete" }).click();

  await expect(page).toHaveURL("/files");
  await expect(
    await page.getByRole("row").filter({ hasText: "upload-for-deletion.txt" })
  ).toHaveCount(0);
});

// Prevent a regression of a bug affecting Firefox:
// https://github.com/mtlynch/picoshare/issues/405
test("uploads a file and then uploads another", async ({ page }) => {
  await login(page);

  // Set default to 30 days.
  await page.getByRole("menuitem", { name: "System" }).hover();
  await page.getByRole("menuitem", { name: "Settings" }).click();
  await expect(page).toHaveURL("/settings");

  await page.locator("#default-expiration").fill("30");
  await page.locator("#time-unit").selectOption("days");
  await page.locator("#settings-form button[type='submit']").click();

  await page.getByRole("menuitem", { name: "Upload" }).click();
  await expect(page).toHaveURL("/");

  await page.locator(".file-input").setInputFiles([
    {
      name: "upload-1.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("I'm the first upload"),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );

  await page.getByRole("button", { name: "Upload Another" }).click();

  await page.locator(".file-input").setInputFiles([
    {
      name: "upload-2.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("I'm the second upload"),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );

  await page.getByRole("menuitem", { name: "Files" }).click();

  await expect(page).toHaveURL("/files");
  await expect(
    page
      .getByRole("row")
      .filter({ hasText: "upload-1.txt" })
      .getByRole("cell")
      .nth(expirationColumn)
  ).toHaveText(/ \(30 days\)$/);
  await expect(
    page
      .getByRole("row")
      .filter({ hasText: "upload-2.txt" })
      .getByRole("cell")
      .nth(expirationColumn)
  ).toHaveText(/ \(30 days\)$/);
});

test("uploads a file and deletes its note", async ({ page }) => {
  await login(page);

  await page.locator("#note").fill("For Pico, with Love and Squalor");

  await page.locator(".file-input").setInputFiles([
    {
      name: "upload-with-temporary-note.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("I'm an upload with a temporary note"),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );

  await page.getByRole("menuitem", { name: "Files" }).click();

  const matchingRow = await page
    .getByRole("row")
    .filter({ hasText: "upload-with-temporary-note.txt" });
  await expect(matchingRow.getByRole("cell").nth(noteColumn)).toHaveText(
    "For Pico, with Love and Squalor"
  );
  await matchingRow
    .getByRole("button")
    .filter({ has: page.locator(".fa-edit") })
    .click();

  await expect(page).toHaveURL(/\/files\/.+\/edit$/);
  await page.locator("#note").fill("");
  await page.getByRole("button", { name: "Save" }).click();

  await expect(page).toHaveURL("/files");
  await expect(
    page
      .getByRole("row")
      .filter({ hasText: "upload-with-temporary-note.txt" })
      .getByRole("cell")
      .nth(noteColumn)
  ).toBeEmpty();
});

test("uploads a file and edits its note", async ({ page }) => {
  await login(page);

  await page.locator("#note").fill("For Pico, with Love and Squalor");

  await page.locator(".file-input").setInputFiles([
    {
      name: "upload-with-note-i-will-edit.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("I'm an upload with a temporary note"),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );

  await page.getByRole("menuitem", { name: "Files" }).click();
  await page
    .getByRole("row")
    .filter({ hasText: "upload-with-note-i-will-edit.txt" })
    .getByRole("button")
    .filter({ has: page.locator(".fa-edit") })
    .click();

  await expect(page).toHaveURL(/\/files\/.+\/edit$/);
  await page.locator("#note").fill("I have a different note now");
  await page.getByRole("button", { name: "Save" }).click();

  await expect(page).toHaveURL("/files");
  await expect(
    await page
      .getByRole("row")
      .filter({ hasText: "upload-with-note-i-will-edit.txt" })
      .getByRole("cell")
      .nth(noteColumn)
  ).toHaveText("I have a different note now");
});

test("uploads a file and changes its expiration time", async ({ page }) => {
  await login(page);

  await page.locator(".file-input").setInputFiles([
    {
      name: "file-with-new-expiration.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("I will have a new custom expiration time"),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );

  await page.getByRole("menuitem", { name: "Files" }).click();
  await page
    .getByRole("row")
    .filter({ hasText: "file-with-new-expiration.txt" })
    .getByRole("button")
    .filter({ has: page.locator(".fa-edit") })
    .click();

  await expect(page).toHaveURL(/\/files\/.+\/edit$/);

  await page.locator("#expiration-picker #expiration").fill("2029-09-04");
  // Move focus to note field just to so the expiration date saves.
  await page.locator("#note").click();
  await page.getByRole("button", { name: "Save" }).click();

  await expect(page).toHaveURL("/files");

  await expect(
    page
      .getByRole("row")
      .filter({ hasText: "file-with-new-expiration.txt" })
      .getByRole("cell")
      .nth(expirationColumn)
  ).toHaveText(/^2029-09-04/);
});
