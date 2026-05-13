import { test, expect } from "@playwright/test";
import { login } from "./helpers/login";

test("sandboxed HTML downloads cannot log out the current user session", async ({
  page,
}) => {
  await login(page);

  await page.locator(".file-input").setInputFiles([
    {
      name: "logout-attack.html",
      mimeType: "text/html",
      buffer: Buffer.from(`<!doctype html>
<html lang="en">
  <body>
    <p id="attack-status">Attack did not start.</p>
    <script>
      document.getElementById("attack-status").textContent =
        "Attack started.";
      fetch("/api/auth", { method: "DELETE", credentials: "include" });
    </script>
  </body>
</html>`),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );

  const downloadURL = await page.locator("#result-links a").first().evaluate(
    (link) => (link as HTMLAnchorElement).href
  );

  const attackPage = await page.context().newPage();
  const response = await attackPage.goto(downloadURL);
  expect(response).not.toBeNull();
  expect(response?.headers()["content-security-policy"]).toBe("sandbox");

  await expect(attackPage.locator("#attack-status")).toHaveText(
    "Attack did not start."
  );
  await attackPage.waitForLoadState("networkidle");

  await page.getByRole("menuitem", { name: "Files" }).click();
  await expect(page).toHaveURL(/\/files$/);
  await expect(
    page.getByRole("row").filter({ hasText: "logout-attack.html" })
  ).toBeVisible();
});
