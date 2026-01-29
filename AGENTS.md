# API design

## Minimize exported surface area

- Don't export methods just for testing - test through public APIs instead.
- Only export what external packages actually need to use.

## Avoid platform coupling

- Don't pass platform-specific types (e.g., AWS Lambda events) to business logic.
- Create simple structs with only the data needed, making code portable.

## Encapsulate related operations

- Group related operations (e.g., verification + processing) in a single method.
- This simplifies APIs and prevents steps from being accidentally skipped.

## Design for testing

- Consider allowing bypass mechanisms for tests (e.g., empty secret = skip verification).
- Test private methods indirectly through public APIs.
- Structure code so unit tests don't need complex setup (e.g., generating valid signatures).

## Keep interfaces simple

- Group related parameters into structs rather than multiple arguments.
- Return single error types that can represent multiple failure modes.
- One method should have one clear responsibility from the caller's perspective.

# Assistant guidelines

## Documenting lessons learned

After successfully completing a task where the user had to provide corrections or guidance, consider adding the lessons to `AGENTS.md`. This helps build institutional knowledge and prevents repeating mistakes.

### When to add new guidelines

- The user corrected a misunderstanding about the codebase.
- You learned a new pattern or best practice specific to this project.
- The user revealed a preference or requirement not previously documented.

### How to add guidelines

1. Identify the key principle or pattern learned.
2. Determine which section of `AGENTS.md` fits best.
3. Add a concise, actionable guideline.
4. Keep entries brief but clear for future LLM conversations.

### Example

If you learned that methods shouldn't be exported just for testing, add to `AGENTS.md`:

"Don't export methods just for testing - test through public APIs instead."

# Playwright (MCP)

- Start PicoShare before driving the UI:
  - `PS_VERSION="$(git describe --tags)" dev-scripts/build-backend dev`
  - `PS_SHARED_SECRET=somepassword ./bin/picoshare-dev -db data/store.db`
- Use the MCP Playwright server tools to drive the UI (for example, `mcp__playwright__browser_navigate` -> `mcp__playwright__browser_snapshot`).

# Go

- Define interfaces in the client package that consumes them, not in the package that implements them.
  - This follows the Go convention that clients should define interfaces based on what they need.
  - Implementing packages should return concrete types.

# Go modules

- Do not vendor Go modules locally (`go mod vendor`).
- Use Nix `vendorHash` in `flake.nix` for reproducible builds instead.
- When dependencies change, update `vendorHash` by running `nix build` with a fake hash and using the correct hash from the error message.

# npm modules

- When dependencies change, update `npmDepsHash` by running `nix build` with a fake hash and using the correct hash from the error message.
- Depend on exact versions of npm packages rather than package minimums.
  - Imprecise versions create discrepancies between npm and flake.nix, especially for playwright.

# dev-scripts

- Use scripts in the dev-scripts to build, run, and test code where possible.
- Do not depend on Nix in the `dev-scripts`.
  - Nix build targets should leverage scripts from `dev-scripts` rather than reimplementing the same logic in Nix.

# Style conventions

- End comments with trailing punctuation.
- Break comment lines at 80 characters.
- Attempt to break bash lines at 80 characters.
- Never keep dead code for the sake of backwards compatibility.
  - If there are no calls to a function or uses of a type/variable outside of test code, it is dead code and should be deleted.

## Markdown

- Do not add line breaks to fit any column width except in code snippets.

### Headings

- Use sentence casing and not title casing.
- Do not add trailing periods.

## JavaScript

- Do not use `alert()` or `confirm()`.
  - Use `window.dialogManager.alert()` and `window.dialogManager.confirm()` instead.
  - For htmx, use `hx-confirm` which hooks into the custom dialog automatically.
- Do not embed Go template conditionals inside JavaScript. Instead, render a data attribute or hidden input in HTML and read it from JS.

# Testing

- After every code change, run `nix flake check` before presenting the solution to the user.
- After every code change, run `dev-scripts/git-hooks/pre-commit` before presenting the solution to the user.
- To run unit tests, run `./dev-scripts/run-go-tests`.
- When writing tests to verify a bugfix, follow TDD conventions: write the test with the failing test first, verify the test fails, fix the bug, and verify that the test passes.
- When writing new test cases, avoid having t.Run have special-case behavior for particular inputs. Instead, use general purpose logic that doesn't assume particular inputs.
- Never use `time.Now` in tests. Use a hardcoded fixed time.
- Go tests should be in a separate `_test` package so they don't test non-exported interfaces.
- Test HTTP handlers by sending requests to the relevant routes. Minimize test coupling by avoiding tests that call HTTP handler functions directly.

## if got, want

Use the `if got, want` pattern when writing or editing unit tests. See this snippet as an example:

```go
func TestParseTwitterHandle(t *testing.T) {
  for _, tt := range []struct {
    explanation    string
    input          string
    handleExpected social.TwitterHandle
    errExpected    error
  }{
    {
      "regular handle on its own is valid",
      "jerry",
      social.TwitterHandle("jerry"),
      nil,
    },
    {
      "regular handle in URL is valid",
      "https://twitter.com/jerry",
      social.TwitterHandle("jerry"),
      nil,
    },
    {
      "handle with exactly 15 characters is valid",
      "https://twitter.com/" + strings.Repeat("A", 15),
      social.TwitterHandle(strings.Repeat("A", 15)),
      nil,
    },
    {
      "handle with more than 15 characters is invalid",
      "https://twitter.com/" + strings.Repeat("A", 16),
      social.TwitterHandle(""),
      social.ErrInvalidTwitterHandle,
    },
  } {
    t.Run(fmt.Sprintf("%s [%s]", tt.explanation, tt.input), func(t *testing.T) {
      handle, err := social.ParseTwitterHandle(tt.input)
      if got, want := err, tt.errExpected; got != want {
        t.Fatalf("err=%v, want=%v", got, want)
      }
      if got, want := handle, tt.handleExpected; got != want {
        t.Errorf("handle=%v, want=%v", got, want)
      }
    })
  }
}
```
