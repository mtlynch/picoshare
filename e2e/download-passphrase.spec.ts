import { test, expect } from "./fixtures";
import { login } from "./helpers/login";

test("requires a passphrase for a protected file download", async ({
  page,
  browser,
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

  await page.getByRole("menuitem", { name: "Files" }).click();
  await page
    .getByRole("row")
    .filter({ hasText: "protected-download.txt" })
    .getByRole("button", { name: "Information" })
    .click();
  await expect(page).toHaveURL(/\/files\/.+\/info$/);
  await expect(
    page
      .locator("section")
      .filter({
        has: page.getByRole("heading", { name: "Download passphrase" }),
      })
      .locator(".value"),
  ).toHaveText("correct horse battery staple");

  // Try to download the file as an unauthenticated visitor.
  {
    const visitorContext = await browser.newContext({ baseURL });
    const visitorPage = await visitorContext.newPage();

    await visitorPage.goto("/");
    await visitorPage.goto(downloadURL);
    await expect(visitorPage).toHaveURL(/\/-[A-Za-z0-9]+\/unlock$/);

    await expect(
      visitorPage.getByRole("heading", { name: "Protected Download" }),
    ).toBeVisible();
    await visitorPage.getByLabel("Passphrase").fill("wrong passphrase");
    await visitorPage.getByRole("button", { name: "Download" }).click();
    await expect(visitorPage.getByText("Incorrect passphrase.")).toBeVisible();

    await visitorPage
      .getByLabel("Passphrase")
      .fill("correct horse battery staple");
    await visitorPage.getByRole("button", { name: "Download" }).click();
    await expect(visitorPage.locator("body")).toHaveText(
      "This download is protected.",
    );
    await visitorContext.close();
  }

  await page.goto(downloadURL);
  await expect(page.locator("body")).toHaveText("This download is protected.");
});

test("adds and removes a download passphrase from the edit page", async ({
  page,
  browser,
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

  // Try to download the file as an unauthenticated visitor.
  {
    const visitorContext = await browser.newContext({ baseURL });
    const visitorPage = await visitorContext.newPage();

    await visitorPage.goto(downloadURL);
    await expect(visitorPage).toHaveURL(/\/-[A-Za-z0-9]+\/unlock$/);
    await visitorPage.getByLabel("Passphrase").fill("open sesame");
    await visitorPage.getByRole("button", { name: "Download" }).click();
    await expect(visitorPage.locator("body")).toHaveText(
      "My passphrase changes after upload.",
    );
    await visitorContext.close();
  }

  // Remove the passphrase from the protected file.
  await page
    .getByRole("row")
    .filter({ hasText: "passphrase-edit.txt" })
    .getByRole("button", { name: "Edit" })
    .click();
  await expect(page).toHaveURL(/\/files\/.+\/edit$/);

  await expect(requirePassphrase).toBeChecked();
  await expect(page.locator("#download-passphrase")).toBeVisible();
  await expect(page.locator("#download-passphrase")).toHaveValue("open sesame");

  await requirePassphrase.uncheck();
  await expect(page.locator("#download-passphrase")).toBeHidden();
  await page.getByRole("button", { name: "Save" }).click();
  await expect(page).toHaveURL("/files");

  // Try to download the file as an unauthenticated visitor.
  {
    const visitorContext = await browser.newContext({ baseURL });
    const visitorPage = await visitorContext.newPage();

    await visitorPage.goto(downloadURL);
    await expect(visitorPage).toHaveURL(downloadURL);
    await expect(visitorPage.locator("body")).toHaveText(
      "My passphrase changes after upload.",
    );
    await visitorContext.close();
  }
});
