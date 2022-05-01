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
