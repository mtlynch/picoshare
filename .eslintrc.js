module.exports = {
  env: {
    node: true,
    browser: true,
    es2022: true,
  },
  extends: ["plugin:cypress/recommended", "eslint:recommended"],
  rules: {
    // This will produce an error for console.log or console.warn in production
    // and a warning in development console.error will not produce an error or
    // warning https://eslint.org/docs/rules/no-console#options
    "no-console": [
      process.env.NODE_ENV === "production" ? "error" : "warn",
      { allow: ["error"] },
    ],
  },
  ignorePatterns: ["playwright-report/*", "static/third-party/**/*.js"],
};
