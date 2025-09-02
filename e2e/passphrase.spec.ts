import { test, expect } from "@playwright/test";
import { login } from "./helpers/login";

// Reuse columns mapping from upload.spec where needed
const noteColumn = 1;

test("uploads a file with a passphrase and requires it on download", async ({ page, request }) => {
  await login(page);

  // Set a note just to ensure it doesn't leak into the prompt page.
  await page.locator("#note").fill("private note - should not appear on prompt");
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

  // Click Files and open the file view link
  await page.getByRole("menuitem", { name: "Files" }).click();
  const matchingRow = await page.getByRole("row").filter({ hasText: "protected.txt" });
  await expect(matchingRow).toBeVisible();

  // Open the file detail (click filename link)
  await matchingRow.getByRole("link", { name: "protected.txt" }).click();

  // We should see the passphrase prompt page
  await expect(page.getByRole("heading", { name: "Enter passphrase" })).toBeVisible();
  await expect(page.getByText("private note - should not appear on prompt")).toHaveCount(0);

  // Try wrong passphrase first
  await page.locator('input[name="passphrase"]').fill("wrong");
  await page.getByRole("button", { name: "View" }).click();
  await expect(page.getByText("Incorrect passphrase")).toBeVisible();

  // Now enter the correct passphrase
  await page.locator('input[name="passphrase"]').fill("letmein");
  await page.getByRole("button", { name: "View" }).click();

  // Content should be visible
  await expect(page.locator("pre")).toHaveText("secret content");

  // Also verify the query param path works: open short link with ?passphrase=
  // First, go back to files and get the verbose link from the info page
  await page.goBack();
  // Re-seleccionar la fila tras navegar
  const rowAfterBack = await page
    .getByRole("row")
    .filter({ hasText: "protected.txt" });
  await rowAfterBack.getByRole("button", { name: "Information" }).click();
  // Copy short link from the UI component
  const shortLink = await page.evaluate(() => {
    const uploadLinks = document.querySelector(
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

  // Open link in same page with passphrase param
  await page.goto(shortLink + "?passphrase=letmein");
  await expect(page.locator("pre")).toHaveText("secret content");
});
