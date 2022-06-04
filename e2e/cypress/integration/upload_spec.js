it("uploads a file without specifying any parameters", () => {
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
  cy.get('.table tbody tr:first-child [test-data-id="note"]').should(
    "not.exist"
  );
});

it("uploads a file with a custom expiration time", () => {
  cy.visit("/");
  cy.login();

  cy.get("#expiration-select").select("Custom");
  cy.get("#expiration-picker").shadow().find("#expiration").clear();
  cy.get("#expiration-picker").shadow().find("#expiration").type(
    "2029-09-03",
    // Cypress seems to incorrectly think that the input field is disabled.
    { force: true }
  );

  // We need a note to break focus from the expiration picker.
  cy.get("#note").type("Test note");

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
  cy.get('.table tbody tr:first-child [test-data-id="expiration"]').should(
    "contain",
    "2029-09-03"
  );
});

it("uploads a file with a note", () => {
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
  cy.get('.table tbody tr:first-child [test-data-id="note"]').should(
    "contain",
    "For Pico, with Love and Squalor"
  );
});

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
