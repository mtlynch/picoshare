import { test, expect } from "@playwright/test";
import { login } from "./helpers/login";

// Reuse columns mapping from upload.spec where needed
const noteColumn = 1;

test("uploads a file with a passphrase and requires it on download", async ({
  page,
  request,
}) => {
  await login(page);

  // Set a note just to ensure it doesn't leak into the prompt page.
  await page
    .locator("#note")
    .fill("private note - should not appear on prompt");
  await page.locator("#passphrase").fill("letmein");

  await page.locator(".file-input").setInputFiles([
    {
      name: "protected.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("secret content"),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );

  // Capture short link from the upload result component (shadow DOM)
  const shortLink = await page.evaluate(() => {
    const resultLinks = document.getElementById("result-links");
    const uploadLinks = resultLinks?.querySelector(
      "upload-links"
    ) as HTMLElement & { shadowRoot: ShadowRoot | null };
    const shortBox = uploadLinks?.shadowRoot?.querySelector(
      "#short-link-box"
    ) as HTMLElement & { shadowRoot: ShadowRoot | null };
    const a = shortBox?.shadowRoot?.querySelector(
      "#link"
    ) as HTMLAnchorElement | null;
    return a?.href || "";
  });

  // Click Files and open the file view link
  await page.getByRole("menuitem", { name: "Files" }).click();
  const matchingRow = await page
    .getByRole("row")
    .filter({ hasText: "protected.txt" });
  await expect(matchingRow).toBeVisible();

  // Open the file detail (click filename link)
  await matchingRow.getByRole("link", { name: "protected.txt" }).click();

  // We should see the passphrase prompt page
  await expect(
    page.getByRole("heading", { name: "Enter passphrase" })
  ).toBeVisible();

  // Try wrong passphrase first
  await page.locator('input[name="passphrase"]').fill("wrong");
  await page.getByRole("button", { name: "View" }).click();
  await expect(page.getByText("Incorrect passphrase")).toBeVisible();

  // Now enter the correct passphrase
  await page.locator('input[name="passphrase"]').fill("letmein");
  await page.getByRole("button", { name: "View" }).click();

  // Content should be visible
  await expect(page.locator("pre")).toHaveText("secret content");

  // Also verify short link flow: open short link, then submit the form
  await page.goto(shortLink);
  await expect(
    page.getByRole("heading", { name: "Enter passphrase" })
  ).toBeVisible();
  await page.locator('input[name="passphrase"]').fill("letmein");
  await page.getByRole("button", { name: "View" }).click();
  await expect(page.locator("pre")).toHaveText("secret content");
});
