it("uploads a file", () => {
  cy.visit("/");
  cy.get('.navbar-item .button[data-test-id="log-in"]').click();

  cy.location("pathname").should("eq", "/login");
  cy.get('form input[type="password"]').type("dummypass");
  cy.get("form").submit();

  cy.location("pathname").should("eq", "/");

  cy.get(".file-input").attachFile("kittyface.jpg");
  cy.get("#upload-result a").should("be.visible");
});
