it("logs in and logs out", () => {
  cy.visit("/");
  cy.get('.navbar-item[data-test-id="navbar-log-in"]').click();

  cy.location("pathname").should("eq", "/login");
  cy.get('form input[type="password"]').type("dummypass");
  cy.get("form").submit();
});
