import { test, expect } from "@playwright/test";
import { login } from "./helpers/login.js";

test("upload a file and verify it has no download history", async ({
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
