import { test, expect } from "@playwright/test";
import { wipeDB } from "./helpers/db.js";
import { login } from "./helpers/login.js";

test.beforeEach(async ({ page }) => {
  await wipeDB(page);
});

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
  await request.post("/api/cleanup");

  await page.locator(".navbar a[href='/files']").click();
  await expect(
    page.locator(
      ".table tr[test-data-filename='simple-upload.txt'] [test-data-id='filename']"
    )
  ).toHaveText("simple-upload.txt");
  await expect(
    page.locator(
      ".table tr[test-data-filename='simple-upload.txt'] [test-data-id='note']"
    )
  ).toHaveCount(0);
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

  await page.locator(".navbar a[href='/files']").click();
  await expect(
    page.locator(
      ".table tr[test-data-filename='custom-expiration-upload.txt'] [test-data-id='filename']"
    )
  ).toHaveText("custom-expiration-upload.txt");
  await expect(
    page.locator(
      ".table tr[test-data-filename='custom-expiration-upload.txt'] [test-data-id='note']"
    )
  ).toHaveCount(0);

  await expect(
    page.locator(
      ".table tr[test-data-filename='custom-expiration-upload.txt'] [test-data-id='expiration']"
    )
  ).toHaveText(/^2029-09-03/);
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

  await page.locator(".navbar a[href='/files']").click();
  await expect(
    page.locator(
      ".table tr[test-data-filename='upload-with-note.txt'] [test-data-id='filename']"
    )
  ).toHaveText("upload-with-note.txt");
  await expect(
    page.locator(
      ".table tr[test-data-filename='upload-with-note.txt'] [test-data-id='note']"
    )
  ).toHaveText("For Pico, with Love and Squalor");
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

  await page.locator(".navbar a[href='/files']").click();
  await page
    .locator(
      ".table tr[test-data-filename='upload-for-deletion.txt'] [pico-purpose='edit']"
    )
    .click();

  await expect(page).toHaveURL(/\/files\/.+\/edit$/);
  await page.locator("[pico-purpose='delete']").click();

  await expect(page).toHaveURL(/\/files\/.+\/confirm-delete$/);
  await page.locator("#delete-btn").click();

  await expect(page).toHaveURL("/files");
  await expect(
    page.locator(".table tr[test-data-filename='upload-for-deletion.txt']")
  ).toHaveCount(0);
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

  await page.locator(".navbar a[href='/files']").click();
  await page
    .locator(
      ".table tr[test-data-filename='upload-with-temporary-note.txt'] [pico-purpose='edit']"
    )
    .click();

  await expect(page).toHaveURL(/\/files\/.+\/edit$/);
  await page.locator("#note").fill("");
  await page.locator("form .button.is-primary").click();

  await expect(page).toHaveURL("/files");
  await expect(
    page.locator(
      ".table tr[test-data-filename='upload-with-temporary-note.txt'] [test-data-id='note']"
    )
  ).toHaveCount(0);
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

  await page.locator(".navbar a[href='/files']").click();
  await page
    .locator(
      ".table tr[test-data-filename='upload-with-note-i-will-edit.txt'] [pico-purpose='edit']"
    )
    .click();

  await expect(page).toHaveURL(/\/files\/.+\/edit$/);
  await page.locator("#note").fill("I have a different note now");
  await page.locator("form .button.is-primary").click();

  await expect(page).toHaveURL("/files");
  await expect(
    page.locator(
      ".table tr[test-data-filename='upload-with-note-i-will-edit.txt'] [test-data-id='note']"
    )
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

  await page.locator(".navbar a[href='/files']").click();
  await page
    .locator(
      ".table tr[test-data-filename='upload-with-temporary-note.txt'] [pico-purpose='edit']"
    )
    .click();

  await expect(page).toHaveURL(/\/files\/.+\/edit$/);

  await page.locator("#expiration-picker #expiration").fill("2029-09-04");
  // Move focus to note field just to so the expiration date saves.
  await page.locator("#note").click();
  await page.locator("form .button.is-primary").click();

  await expect(page).toHaveURL("/files");
  await expect(
    page.locator(
      ".table tr[test-data-filename='upload-with-temporary-note.txt'] [test-data-id='expiration']"
    )
  ).toHaveText(/^2029-09-04/);
});
