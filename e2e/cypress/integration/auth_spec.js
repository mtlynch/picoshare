it("logs in and logs out", () => {
  cy.visit("/");
  cy.get('.navbar-item[data-test-id="log-in"]').click();

  cy.location("pathname").should("eq", "/login");
  cy.get('form input[type="password"]').type("dummypass");
  cy.get("form").submit();

  cy.location("pathname").should("eq", "/");
  // Using force here because I can't figure out how to get Cypress to expose
  // the dropdown menu.
  cy.get("#navbar-log-out").click({ force: true });

  cy.location("pathname").should("eq", "/");
  cy.get('.navbar-item[data-test-id="log-in"]').should("be.visible");
});
