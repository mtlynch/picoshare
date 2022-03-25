module.exports = {
  root: true,
  env: {
    node: true,
    es6: true,
  },
  extends: ["plugin:cypress/recommended", "eslint:recommended"],
  rules: {
    "no-console": "error",
  },
  parserOptions: {
    parser: "babel-eslint",
  },
};
