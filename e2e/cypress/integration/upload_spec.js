it("uploads a file", () => {
  cy.visit("/");
  cy.get('.navbar-item .button[data-test-id="log-in"]').click();

  cy.location("pathname").should("eq", "/login");
  cy.get('form input[type="password"]').type("dummypass");
  cy.get("form").submit();

  cy.location("pathname").should("eq", "/");

  cy.get(".file-input").attachFile("kittyface.jpg");

  cy.get("#upload-result .message-body").should("contain", "Upload complete!");

  cy.get("#upload-result upload-links")
    .should("have.attr", "filename")
    .and("equal", "kittyface.jpg");
  cy.get("#upload-result upload-links")
    .shadow()
    .find("#verbose-link")
    .should("be.visible");
  cy.get("#upload-result upload-links")
    .shadow()
    .find("#short-link")
    .should("be.visible");

  cy.get('.navbar a[href="/files"]').click();
  cy.get('.table td[test-data-id="filename"]').should(
    "contain",
    "kittyface.jpg"
  );
});
