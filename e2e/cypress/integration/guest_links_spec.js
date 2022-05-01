it("creates a guest link and uploads a file as a guest", () => {
  cy.visit("/");
  cy.login();

  cy.location("pathname").should("eq", "/");
  cy.get('nav .navbar-item[href="/guest-links"]').click();

  cy.get(".content .button.is-primary").click();

  cy.location("pathname").should("eq", "/guest-links/new");

  cy.get("#label").type("For e2e testing");
  cy.get("#max-file-size").type("50");
  cy.get("#file-upload-limit").type("1");
  cy.get("#create-guest-link-form").submit();

  cy.location("pathname").should("eq", "/guest-links");
  cy.get('.table td[test-data-id="guest-link-label"] a').click();

  // Using force here because I can't figure out how to get Cypress to expose
  // the dropdown menu.
  cy.get("#navbar-log-out").click({ force: true });
  // Hack to get back to the guest upload page without extracting the path
  // within Cypress JS code.
  cy.go("back");

  cy.location("pathname").should("match", /^\/g\//);

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

  cy.get("#upload-another-btn").click();

  cy.get("h1").should("contain", "Guest Link Inactive");
  cy.get(".file-input").should("not.exist");
});
