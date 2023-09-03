import { test, expect } from "@playwright/test";
import { login } from "./helpers/login.js";

test("upload a file with no attributes, and verify the file info page is correct", async ({
  page,
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
      .filter({ has: page.getByRole("heading", { name: "Filename" }) })
      .locator(".value")
  ).toHaveText("simple-upload.txt");
  await expect(
    page
      .locator("section")
      .filter({ has: page.getByRole("heading", { name: "Size" }) })
      .locator(".value")
  ).toHaveText("24 B");
  await expect(
    page
      .locator("section")
      .filter({ has: page.getByRole("heading", { name: "Expires" }) })
      .locator(".value")
  ).toHaveText(/ \(30 days\)$/);
  await expect(
    page
      .locator("section")
      .filter({ has: page.getByRole("heading", { name: "Note" }) })
      .locator(".value")
  ).toHaveText("None");
  await expect(
    page
      .locator("section")
      .filter({ has: page.getByRole("heading", { name: "Uploaded by" }) })
      .locator(".value")
  ).toHaveText("You");
  await expect(
    page
      .locator("section")
      .filter({ has: page.getByRole("heading", { name: "Upload time" }) })
      .locator(".value")
  ).toHaveText(
    /^(January|February|March|April|May|June|July|August|September|October|November|December) (\d{1,2}), (20[0-9]{2}) at ([1-9]|1[0-2]):([0-5][0-9]) (AM|PM)/
  );
});

test("upload a file with a note and custom expiration, and verify the file info page is correct", async ({
  page,
}) => {
  await login(page);

  await page.locator("#expiration-select").selectOption({ label: "Custom" });
  await page.locator("#expiration-picker #expiration").fill("2029-09-03");

  await page.locator("#note").fill("Please note that this upload has a note");

  await page.locator(".file-input").setInputFiles([
    {
      name: "upload-with-more-metadata.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("I'm an upload with custom metadata"),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );

  await page.getByRole("menuitem", { name: "Files" }).click();

  await expect(page).toHaveURL(/\/files$/);
  await page
    .getByRole("row")
    .filter({ hasText: "upload-with-more-metadata.txt" })
    .getByRole("button", { name: "Information" })
    .click();

  await expect(page).toHaveURL(/\/files\/.+\/info$/);
  await expect(
    page
      .locator("section")
      .filter({ has: page.getByRole("heading", { name: "Filename" }) })
      .locator(".value")
  ).toHaveText("upload-with-more-metadata.txt");
  await expect(
    page
      .locator("section")
      .filter({ has: page.getByRole("heading", { name: "Size" }) })
      .locator(".value")
  ).toHaveText("34 B");
  await expect(
    page
      .locator("section")
      .filter({ has: page.getByRole("heading", { name: "Expires" }) })
      .locator(".value")
  ).toHaveText(/2029-09-03 \(\d+ days\)/);
  await expect(
    page
      .locator("section")
      .filter({ has: page.getByRole("heading", { name: "Note" }) })
      .locator(".value")
  ).toHaveText("Please note that this upload has a note");
  await expect(
    page
      .locator("section")
      .filter({ has: page.getByRole("heading", { name: "Uploaded by" }) })
      .locator(".value")
  ).toHaveText("You");
  await expect(
    page
      .locator("section")
      .filter({ has: page.getByRole("heading", { name: "Upload time" }) })
      .locator(".value")
  ).toHaveText(
    /^(January|February|March|April|May|June|July|August|September|October|November|December) (\d{1,2}), (20[0-9]{2}) at ([1-9]|1[0-2]):([0-5][0-9]) (AM|PM)/
  );
});
