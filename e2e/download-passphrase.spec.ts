import { test, expect } from "./fixtures";
import { login } from "./helpers/login";

test("requires a passphrase for a protected file download", async ({
  page,
  baseURL,
}) => {
  await login(page);

  await page
    .locator("#download-passphrase")
    .fill("correct horse battery staple");
  await page.locator(".file-input").setInputFiles([
    {
      name: "protected-download.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("This download is protected."),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!",
  );

  const downloadURL = await page
    .locator("#result-links a")
    .first()
    .evaluate((link) => (link as HTMLAnchorElement).href);

  const browser = page.context().browser();
  if (browser === null) {
    throw new Error("browser is unavailable");
  }
  const visitorContext = await browser.newContext({ baseURL });
  const visitor = await visitorContext.newPage();
  await visitor.goto("/");
  await visitor.goto(downloadURL);
  await expect(visitor).toHaveURL(/\/-[A-Za-z0-9]+\/unlock$/);

  await expect(
    visitor.getByRole("heading", { name: "Protected Download" }),
  ).toBeVisible();
  await expect(
    visitor.getByText(
      "Enter this file's passphrase to complete your download:",
    ),
  ).toBeVisible();
  await visitor.getByLabel("Passphrase").fill("wrong passphrase");
  await visitor.getByRole("button", { name: "Download" }).click();
  await expect(visitor.getByText("Incorrect passphrase.")).toBeVisible();

  await visitor.getByLabel("Passphrase").fill("correct horse battery staple");
  await visitor.getByRole("button", { name: "Download" }).click();
  await expect(visitor.locator("body")).toHaveText(
    "This download is protected.",
  );

  await page.goto(downloadURL);
  await expect(page.locator("body")).toHaveText("This download is protected.");
  await visitorContext.close();
});

test("adds and removes a download passphrase from the edit page", async ({
  page,
  baseURL,
}) => {
  await login(page);

  await page.locator(".file-input").setInputFiles([
    {
      name: "passphrase-edit.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("My passphrase changes after upload."),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!",
  );

  const downloadURL = await page
    .locator("#result-links a")
    .first()
    .evaluate((link) => (link as HTMLAnchorElement).href);

  const browser = page.context().browser();
  if (browser === null) {
    throw new Error("browser is unavailable");
  }
  const visitorContext = await browser.newContext({ baseURL });
  const visitor = await visitorContext.newPage();

  // Add a passphrase to the unprotected file.
  await page.getByRole("menuitem", { name: "Files" }).click();
  await page
    .getByRole("row")
    .filter({ hasText: "passphrase-edit.txt" })
    .getByRole("button", { name: "Edit" })
    .click();
  await expect(page).toHaveURL(/\/files\/.+\/edit$/);

  const requirePassphrase = page.getByLabel(
    "Require users to enter a passphrase before downloading",
  );
  await expect(requirePassphrase).not.toBeChecked();
  await expect(page.locator("#download-passphrase")).toBeHidden();

  await requirePassphrase.check();
  await expect(page.locator("#download-passphrase")).toBeVisible();
  await page.locator("#download-passphrase").fill("open sesame");
  await page.getByRole("button", { name: "Save" }).click();
  await expect(page).toHaveURL("/files");

  await visitor.goto(downloadURL);
  await expect(visitor).toHaveURL(/\/-[A-Za-z0-9]+\/unlock$/);
  await visitor.getByLabel("Passphrase").fill("open sesame");
  await visitor.getByRole("button", { name: "Download" }).click();
  await expect(visitor.locator("body")).toHaveText(
    "My passphrase changes after upload.",
  );

  // Remove the passphrase from the protected file.
  await page
    .getByRole("row")
    .filter({ hasText: "passphrase-edit.txt" })
    .getByRole("button", { name: "Edit" })
    .click();
  await expect(page).toHaveURL(/\/files\/.+\/edit$/);

  await expect(requirePassphrase).toBeChecked();
  await expect(page.locator("#download-passphrase")).toBeVisible();

  await requirePassphrase.uncheck();
  await expect(page.locator("#download-passphrase")).toBeHidden();
  await page.getByRole("button", { name: "Save" }).click();
  await expect(page).toHaveURL("/files");

  await visitor.goto(downloadURL);
  await expect(visitor).toHaveURL(downloadURL);
  await expect(visitor.locator("body")).toHaveText(
    "My passphrase changes after upload.",
  );
  await visitorContext.close();
});
