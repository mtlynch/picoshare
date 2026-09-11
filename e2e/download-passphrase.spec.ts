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

  await expect(
    visitor.getByRole("heading", { name: "Download passphrase" }),
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
