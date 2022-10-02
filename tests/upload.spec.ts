/*

it("uploads a file and deletes it", () => {
  cy.visit("/");
  cy.login();

  cy.get(".file-input").attachFile("kittyface.jpg");

  cy.get("#upload-result .message-body").should("contain", "Upload complete!");

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

  cy.get('.navbar a[href="/files"]').click();
  cy.get('.table tbody tr:first-child [test-data-id="filename"]').should(
    "contain",
    "kittyface.jpg"
  );

  // Verify that cleanup doesn't incorrectly remove the file.
  cy.request("POST", "/api/cleanup");

  cy.get('.table tbody tr:first-child [pico-purpose="edit"]').click();

  cy.location("pathname").should("match", new RegExp("/files/.+/edit"));

  cy.get('[pico-purpose="delete"]').click();

  cy.location("pathname").should(
    "match",
    new RegExp("/files/.+/confirm-delete")
  );

  cy.get("#delete-btn").click();

  cy.location("pathname").should("eq", "/files");
});

it("uploads a file and deletes its note", () => {
  cy.visit("/");
  cy.login();

  cy.get("#note").type("For Pico, with Love and Squalor");

  cy.get(".file-input").attachFile("kittyface.jpg");

  cy.get("#upload-result .message-body").should("contain", "Upload complete!");

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

  cy.get('.navbar a[href="/files"]').click();
  cy.get('.table tbody tr:first-child [test-data-id="filename"]').should(
    "contain",
    "kittyface.jpg"
  );

  // Verify that cleanup doesn't incorrectly remove the file.
  cy.request("POST", "/api/cleanup");

  cy.get('.table tbody tr:first-child [pico-purpose="edit"]').click();

  cy.location("pathname").should("match", new RegExp("/files/.+/edit"));

  cy.get("#note").clear();
  cy.get("form .button.is-primary").click();

  cy.location("pathname").should("eq", "/files");

  cy.get('.table tbody tr:first-child [test-data-id="note"]').should(
    "not.exist"
  );
});

it("uploads a file and edits its note", () => {
  cy.visit("/");
  cy.login();

  cy.get("#note").type("For Pico, with Love and Squalor");

  cy.get(".file-input").attachFile("kittyface.jpg");

  cy.get("#upload-result .message-body").should("contain", "Upload complete!");

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

  // Verify that cleanup doesn't incorrectly remove the file.
  cy.request("POST", "/api/cleanup");

  cy.get('.navbar a[href="/files"]').click();
  cy.get('.table tbody tr:first-child [test-data-id="filename"]').should(
    "contain",
    "kittyface.jpg"
  );

  cy.get('.table tbody tr:first-child [pico-purpose="edit"]').click();

  cy.location("pathname").should("match", new RegExp("/files/.+/edit"));

  cy.get("#note").clear();
  cy.get("#note").type("My favorite kitten!");
  cy.get("form .button.is-primary").click();

  cy.location("pathname").should("eq", "/files");

  cy.get('.table tbody tr:first-child [test-data-id="note"]').should(
    "contain",
    "My favorite kitten!"
  );
});

it("uploads a file and changes its expiration time", () => {
  cy.visit("/");
  cy.login();

  cy.get(".file-input").attachFile("kittyface.jpg");

  cy.get("#upload-result .message-body").should("contain", "Upload complete!");

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

  // Verify that cleanup doesn't incorrectly remove the file.
  cy.request("POST", "/api/cleanup");

  cy.get('.navbar a[href="/files"]').click();
  cy.get('.table tbody tr:first-child [test-data-id="filename"]').should(
    "contain",
    "kittyface.jpg"
  );

  cy.get('.table tbody tr:first-child [pico-purpose="edit"]').click();

  cy.location("pathname").should("match", new RegExp("/files/.+/edit"));

  cy.get("#expiration").shadow().find("#expiration").clear();
  cy.get("#expiration").shadow().find("#expiration").type(
    "2029-09-03",
    // Cypress seems to incorrectly think that the input field is disabled.
    { force: true }
  );
  cy.get("form .button.is-primary").click();

  cy.location("pathname").should("eq", "/files");

  cy.get('.table tbody tr:first-child [test-data-id="expiration"]').should(
    "contain",
    "2029-09-03"
  );
});
*/

import { test, expect } from "@playwright/test";
import { login } from "./helpers/login.js";

test("uploads a file without specifying any parameters", async ({
  page,
  request,
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
  await expect(page.locator("#upload-result upload-links")).toHaveAttribute(
    "filename",
    "simple-upload.txt"
  );
  await expect(
    page.locator("#upload-result upload-links #verbose-link-box #link")
  ).toBeVisible();
  await expect(
    page.locator("#upload-result upload-links #short-link-box #link")
  ).toBeVisible();

  // Verify that cleanup doesn't incorrectly remove the file.
  await request.post("/api/cleanup");

  await page.locator(".navbar a[href='/files']").click();
  await expect(
    page.locator(
      ".table tr[test-data-filename='simple-upload.txt'] [test-data-id='filename']"
    )
  ).toHaveText("simple-upload.txt");
  await expect(
    page.locator(
      ".table tr[test-data-filename='simple-upload.txt'] [test-data-id='note']"
    )
  ).toHaveCount(0);
});

test("uploads a file with a custom expiration time", async ({
  page,
  request,
}) => {
  await login(page);

  await page.locator("#expiration-select").selectOption({ label: "Custom" });

  await page.locator("#expiration-picker #expiration").fill("2029-09-03");

  // Move focus to note field just to so the expiration date saves.
  page.locator("#note").fill("");

  await page.locator(".file-input").setInputFiles([
    {
      name: "custom-expiration-upload.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("I have a custom expiration time"),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );
  await expect(page.locator("#upload-result upload-links")).toHaveAttribute(
    "filename",
    "custom-expiration-upload.txt"
  );
  await expect(
    page.locator("#upload-result upload-links #verbose-link-box #link")
  ).toBeVisible();
  await expect(
    page.locator("#upload-result upload-links #short-link-box #link")
  ).toBeVisible();

  // Verify that cleanup doesn't incorrectly remove the file.
  await request.post("/api/cleanup");

  await page.locator(".navbar a[href='/files']").click();
  await expect(
    page.locator(
      ".table tr[test-data-filename='custom-expiration-upload.txt'] [test-data-id='filename']"
    )
  ).toHaveText("custom-expiration-upload.txt");
  await expect(
    page.locator(
      ".table tr[test-data-filename='custom-expiration-upload.txt'] [test-data-id='note']"
    )
  ).toHaveCount(0);

  await expect(
    page.locator(".table tbody tr:first-child [test-data-id='expiration']")
  ).toHaveText(/^2029-09-03/);
});

test("uploads a file with a note", async ({ page, request }) => {
  await login(page);

  await page.locator("#note").fill("For Pico, with Love and Squalor");

  await page.locator(".file-input").setInputFiles([
    {
      name: "upload-with-note.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("I'm an upload with a note"),
    },
  ]);
  await expect(page.locator("#upload-result .message-body")).toHaveText(
    "Upload complete!"
  );
  await expect(page.locator("#upload-result upload-links")).toHaveAttribute(
    "filename",
    "upload-with-note.txt"
  );
  await expect(
    page.locator("#upload-result upload-links #verbose-link-box #link")
  ).toBeVisible();
  await expect(
    page.locator("#upload-result upload-links #short-link-box #link")
  ).toBeVisible();

  // Verify that cleanup doesn't incorrectly remove the file.
  await request.post("/api/cleanup");

  await page.locator(".navbar a[href='/files']").click();
  await expect(
    page.locator(
      ".table tr[test-data-filename='upload-with-note.txt'] [test-data-id='filename']"
    )
  ).toHaveText("upload-with-note.txt");
  await expect(
    page.locator(
      ".table tr[test-data-filename='upload-with-note.txt'] [test-data-id='note']"
    )
  ).toHaveText("For Pico, with Love and Squalor");
});
