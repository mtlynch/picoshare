module.exports = {
  root: true,
  env: {
    node: true,
    browser: true,
    es2022: true,
  },
  extends: ["plugin:cypress/recommended", "eslint:recommended"],
  rules: {
    "no-console": "error",
  },
};
