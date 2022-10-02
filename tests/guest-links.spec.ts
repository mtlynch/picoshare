/*



  cy.get("#upload-result upload-links")
    .should("have.attr", "filename")
    .and("equal", "kittyface.jpg");
  cy.get("#upload-result upload-links")
    .shadow()
    .find("#verbose-link-box")
    .shadow()
    .find("#link")
    .should("be.visible");
  cy.get("#upload-result upload-links")
    .shadow()
    .find("#short-link-box")
    .shadow()
    .find("#link")
    .should("be.visible");

  cy.get("#upload-another-btn").click();

  cy.get("h1").should("contain", "Guest Link Inactive");
  cy.get(".file-input").should("not.exist");
});
*/

import { test, expect } from "@playwright/test";

test("creates a guest link and uploads a file as a guest", async ({ page }) => {
  await page.goto("/");

  await page.locator("data-test-id=log-in").click();

  await expect(page).toHaveURL("/login");
  await page.locator("form input[type='password']").fill("dummypass");
  await page.locator("form input[type='submit']").click();

  await expect(page).toHaveURL("/");
  await page.locator("nav .navbar-item[href='/guest-links']").click();

  await page.locator(".content .button.is-primary").click();

  await expect(page).toHaveURL("/guest-links/new");
  await page.locator("#label").fill("For e2e testing");
  await page.locator("#max-file-size").fill("50");
  await page.locator("#file-upload-limit").fill("1");
  await page.locator("#create-guest-link-form input[type='submit']").click();

  await expect(page).toHaveURL("/guest-links");
  await page
    .locator('.table td[test-data-id="guest-link-label"] a', {
      hasText: "For e2e testing",
    })
    .click();

  await page.locator(".navbar-end .navbar-item.is-hoverable").hover();
  await page.locator("#navbar-log-out").click();

  // Hack to get back to the guest upload page without extracting the path
  // within Cypress JS code.
  page.goBack();
  await expect(page).toHaveURL(/\/g\/.+/);
  await page
    .locator(".file-input")
    .setInputFiles(["./tests/testdata/kittyface.jpg"]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );
});
