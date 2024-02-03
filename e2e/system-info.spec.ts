import { test, expect } from "@playwright/test";
import { login } from "./helpers/login.js";

test("views system info", async ({ page }) => {
  await login(page);

  await page.getByRole("menuitem", { name: "System" }).hover();
  await page.getByRole("menuitem", { name: "Information" }).click();
  await expect(page).toHaveURL("/information");
});
