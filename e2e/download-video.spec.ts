import { test, expect } from "./fixtures";
import { login } from "./helpers/login";

test("plays an uploaded video", async ({ page }) => {
  await login(page);

  await page
    .getByLabel("Choose a file…")
    .setInputFiles("e2e/fixtures/tiny-video.mp4");
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!",
  );

  await page.getByRole("menuitem", { name: "Files" }).click();
  const fileRow = page.getByRole("row").filter({ hasText: "tiny-video.mp4" });
  await fileRow.getByRole("link", { name: "tiny-video.mp4" }).click();
  await expect(page).toHaveURL(/\/-[A-Za-z0-9]+$/);

  // Chrome's generated media page does not provide an accessible name for its
  // video element.
  const player = page.locator("video");
  await expect(player).toHaveJSProperty("videoWidth", 64);
  await expect(player).toHaveJSProperty("videoHeight", 64);
  await player.evaluate((element) => (element as HTMLVideoElement).play());
  await expect
    .poll(() =>
      player.evaluate((element) => (element as HTMLVideoElement).currentTime),
    )
    .toBeGreaterThan(0.25);
  await expect(player).toHaveJSProperty("error", null);
});
