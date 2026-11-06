# Go

## Defining symbols

- Define variables and structs as close as possible to when they're first used.
  - Exception: Declare all package-level variables and structs at the top of the file immediately after imports.
- Name helper functions with an explicit verb, e.g. `mustCreateUser` instead of `mustNewUser`.

## Formatting

- Run `gofmt -s -w` on new or modified `.go` files before committing.

## Error handling

- In `fmt.Errorf` wrappers, describe the failed operation with `failed to <verb>` instead of gerund phrases like `parsing <thing>`.

## Interfaces

- Define interfaces in the client package that consumes them, not in the package that implements them.
  - This follows the Go convention that clients should define interfaces based on what they need.
  - Implementing packages should return concrete types.

## Parse, don't validate

- Represent data constrained by business rules with a type that prevents invalid states rather than repeatedly validating primitives.
  - Give the type an unexported primitive field, construct it through a validating constructor, and expose an accessor for the validated value.
  - Apply this especially to user-supplied data.

For example:

```go
type HashKey struct {
	value string
}

func NewHashKey(raw string) (HashKey, error) {
	if utf8.RuneCountInString(raw) != 5 {
		return HashKey{}, fmt.Errorf("hash key has incorrect length")
	}
	return HashKey{value: raw}, nil
}

func (h HashKey) String() string {
	return h.value
}
```

## HTTP handlers

- Always parse user-supplied HTTP data into a typed request value with a dedicated parse function. This preserves the data's provenance.
- Convert every user-supplied request value into a named, validated type before passing it through the application.

For example:

```go
type entryUpdateRequest struct {
	EntryID app.EntryID
	Note    app.Note
}

func parseEntryUpdateRequest(r *http.Request) (entryUpdateRequest, error) {
	id, err := parseEntryIDFromPathValue(r, "id")
	if err != nil {
		return entryUpdateRequest{}, err
	}

	if err := r.ParseForm(); err != nil {
		return entryUpdateRequest{}, err
	}

	note, err := app.NewNote(r.FormValue("note"))
	if err != nil {
		return entryUpdateRequest{}, err
	}

	return entryUpdateRequest{
		EntryID: id,
		Note:    note,
	}, nil
}
```

## Go modules

- Do not vendor Go modules locally (`go mod vendor`).
- Use Nix `vendorHash` in `flake.nix` for reproducible builds instead.
- When dependencies change, update `vendorHash` by running `nix build` with a fake hash and using the correct hash from the error message.

## Go testing

- When writing new test cases, avoid having `t.Run` have special-case behavior for particular inputs.
  - Instead, use general purpose logic that doesn't assume particular inputs.
- Design test table fields to describe the scenario at a high level (e.g., `priorUploadsForUser int`) rather than exposing raw data structures the reader must mentally reconstruct (e.g., `recordedEntries []recordedEntry`). A reader should understand the test behavior from the table alone, before reading the test body.
- An `explanation` should state its condition and expected outcome.
  - Good: `"files with empty names should return an error"`
  - Bad: `"empty name"`
  - Good: `"files with special characters in the name should succeed"`
  - Bad: `"special characters"`
- Never use `time.Now` in tests. Use a hardcoded fixed time.
- When tests need time to progress, reassign one shared `now` variable in chronological order rather than using inline `now.Add(...)` values. Do not move test time backwards.
- Inject `time.Now` as a `func() time.Time` dependency so production code is testable.
  - If a type already injects a clock for one purpose, use that same clock for all time-dependent logic in that type.
- Go tests should be in a separate `_test` package so they don't test non-exported interfaces.
- Test HTTP handlers by sending requests to the relevant routes.
  - Minimize test coupling by avoiding tests that call HTTP handler functions directly.
  - The HTTP route and method a test exercises should be visible in the test body or testcase struct. Never hide the route or method in a helper function.
- When a Go test performs multiple HTTP requests in one flow, keep shared fixture setup at the top of the test and isolate each request in its own lexical scope.
  - Share state between request blocks explicitly through variables such as cookies or IDs.
  - Example pattern:

    ```go
    server := newTestServer(t)

    var sessionCookie *http.Cookie
    {
      req := httptest.NewRequest(http.MethodPost, "/login", http.NoBody)
      rr := httptest.NewRecorder()

      server.Router().ServeHTTP(rr, req)

      cookies := rr.Result().Cookies()
      if got, want := len(cookies), 1; got != want {
        t.Fatalf("cookie count=%d, want=%d", got, want)
      }
      sessionCookie = cookies[0]
    }

    {
      req := httptest.NewRequest(http.MethodGet, "/account", nil)
      req.AddCookie(sessionCookie)
      rr := httptest.NewRecorder()

      server.Router().ServeHTTP(rr, req)

      if got, want := rr.Code, http.StatusOK; got != want {
        t.Fatalf("status=%d, want=%d", got, want)
      }
    }
    ```

- Use `t.Fatalf` when a failed assertion leaves nothing useful to test because the remaining values would be invalid or undefined.
  - For example, stop after an unexpected error when subsequent assertions depend on the returned value.
  - An HTTP status mismatch is almost always a fatal error, as there's no reason to check the body when the status is not what the test expected.
  - Otherwise, use `t.Errorf` so independent assertions still run.
- Do not write tests that scrape information from `log.Print*` calls.
- Test domain-value validation in the package that defines the typed value, not only through HTTP handler tests.

### if got, want

Use the `if got, want` pattern when writing or editing unit tests. See this snippet as an example:

```go
func TestParseEntryID(t *testing.T) {
  for _, tt := range []struct {
    explanation string
    input       string
    idExpected  entries.ID
    errExpected error
  }{
    {
      explanation: "regular entry ID is valid",
      input:       "abc123",
      idExpected:  entries.ID("abc123"),
      errExpected: nil,
    },
    {
      explanation: "entry ID with a slash is invalid",
      input:       "abc/123",
      idExpected:  entries.ID(""),
      errExpected: entries.ErrInvalidID,
    },
  } {
    t.Run(fmt.Sprintf("%s [%s]", tt.explanation, tt.input), func(t *testing.T) {
      id, err := entries.ParseID(tt.input)
      if got, want := err, tt.errExpected; got != want {
        t.Fatalf("err=%v, want=%v", got, want)
      }
      if got, want := id, tt.idExpected; got != want {
        t.Errorf("id=%v, want=%v", got, want)
      }
    })
  }
}
```

### Table-driven tests

- The ordering of fields in `TestParseEntryID` is intentional. It goes:
  - Human-facing explanation of what the test is asserting.
  - Input(s) to the function under test.
  - Output(s) from the function under test.
- Use an `explanation string` field as the first field in table-driven test cases.
- When testing HTTP handlers, place the expected HTTP status first among the outputs.
- The test cases should all have keyed field names like `explanation: "..."` for easier readability.
  - The field names should always start on a new line.

# Callsite legibility

Function calls should be legible from the callsite.

Here's an illegible function call:

```go
foo(3, "foo", 9.2)
```

The call is illegible because it's impossible to infer from the callsite what the parameters mean.

Here is a more legible callsite:

```go
angle := 3
rocketName := "foo"
power := 9.2
foo(angle, rocketName, power)
```

Or change the callee to accept a struct:

```go
foo(LaunchParams{
	Angle:      3,
	RocketName: "foo",
	Power:      9.2,
})
```

Dave Cheney's functional options pattern is also a more legible alternative:

```go
func (s Store) ReadEntries(opts ...store.ReadEntriesOption) { ... }

...

s.ReadEntries(
    store.FilterByUploader(user),
    store.FilterByEntryID(id),
)
```

# API design

## Simplicity

- Keep shell scripts and Nix apps as simple as possible.
- Do not add more than one method for achieving the same outcome.
  - If a script or workflow is replaced with a better one, do not keep the old one for legacy purposes or backwards compatibility.
  - Upgrade all references to use the new path.
- Do not add extra flags to support hypothetical scenarios.
- Do not add experimental modes, compatibility shims, aliases, workarounds, or fallback paths.
  - For code we own, prefer a single path that we know works and fix that path when it fails.
- YAGNI.
  - Do not add flags or options until there is a specific need.

## Minimize exported surface area

- Don't export methods just for testing - test through public APIs instead.
- Only export what external packages actually need to use.

## Avoid platform coupling

- Don't pass platform-specific types (e.g., AWS Lambda events) to business logic.
- Create simple structs with only the data needed, making code portable.

## Encapsulate related operations

- Group related operations (e.g., verification and processing) in a single method.
- This simplifies APIs and prevents steps from being accidentally skipped.

## HTTP errors

- Include the failed operation and underlying error in user-facing HTTP error messages instead of generic messages like `Internal Server Error`.
  - No: `http.Error(w, "Internal Server Error", http.StatusInternalServerError)`
  - Yes: `http.Error(w, fmt.Sprintf("Failed to retrieve reactions: %v", err), http.StatusInternalServerError)`
- Keep the server-side log call alongside the corresponding `http.Error` call.
  - Log messages are internally-facing and should include more context when possible, such as the input data that the server caused a failure.
    - Log messages should not be capitalized unless they begin with a word that is always capitalized, such as a proper noun.
  - HTTP error messages are user-facing and should include friendlier, less technical language. They should include errors to help the user report failures to the developers, but they should avoid leaking too many implementation details.
    - HTTP error messages should be capitalized.
  - Do not repeat the request method or URL in handler error logs; `LoggingHandler` already logs them.

## Design for testing

- Consider allowing bypass mechanisms for tests (e.g., empty secret means skip verification).
- Test private methods indirectly through public APIs.
- Structure code so unit tests don't need complex setup (e.g., generating valid signatures).

## Keep interfaces simple

- Group related parameters into structs rather than multiple arguments.
- Return single error types that can represent multiple failure modes.
- One method should have one clear responsibility from the caller's perspective.

## Avoid redundant naming in interface methods

- Method names should not repeat the domain implied by the interface or type name.
- Read the method at its callsite (`s.field.Method()`) to check for redundancy.
- Prefer names like `PasswordResetter.Request(user)` over `PasswordResetter.RequestReset(user)`.

# Assistant guidelines

- Do not ask the user to run a command or test, or read an output file, that you can read yourself.
  - Aim as much as possible to absorb work from the user.

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
  - `dev-scripts/build-backend dev`
  - `PS_SHARED_SECRET=somepassword ./bin/picoshare-dev -db data/store.db`
- Use the MCP Playwright server tools to drive the UI (for example, `mcp__playwright__browser_navigate` followed by `mcp__playwright__browser_snapshot`).

# Nix flake

- Do not embed scripts in Nix files when the script is longer than 15 lines of code.
  - Keep scripts as standalone files and call them from the flake.
  - Exception: The development shell hook may remain embedded regardless of its length.
- Avoid string substitution to inject large shell scripts into the flake.
- When communicating executable locations from the flake to shell scripts, prefer updating `PATH` so scripts can execute binaries by name.

# npm modules

- When dependencies change, update `npmDepsHash` by running `nix build` with a fake hash and using the correct hash from the error message.
- Depend on exact versions of npm packages rather than package minimums.
  - Imprecise versions create discrepancies between npm and `flake.nix`, especially for Playwright.

# dev-scripts

- Prefer cached Nix test targets when the flake exposes an equivalent target.
- Do not depend on Nix in `dev-scripts`.
  - Nix build targets should leverage scripts from `dev-scripts` rather than reimplementing the same logic in Nix.

# Style conventions

- End complete sentences in comments with trailing punctuation. Do not add punctuation to headings or sentence fragments, including inline labels such as `# Replace` and `# Optional` or section headings such as `// Static files`.
- Do not add a period immediately after a URL, even at the end of a sentence.
- Break code comment lines at 80 characters.
  - Treat tabs as two spaces.
- Do not make assumptions about external clients or deployment environments in code comments unless they impose a relevant design constraint.
  - Incorrect: `// StoreEntry saves an upload that comes from the web interface when deployed on Fly.io.`
  - Correct: `// StoreEntry persists an entry in the SQLite database.`
  - Exception: `// RetryConnection works around a Fly.io issue where the first connection always fails.`
- Never keep dead code for the sake of backwards compatibility.
  - If there are no calls to a function or uses of a type or variable outside of test code, it is dead code and should be deleted.

## Markdown

- Do not hard wrap Markdown text outside code snippets.
- Generally, prefer bulleted lists to tables.

### Headings

- Use sentence casing and not title casing.
- Do not add trailing periods.

## JavaScript

- Do not use `alert()` or `confirm()`.
  - Use PicoShare's custom dialog UI instead.
- Do not embed Go template conditionals inside JavaScript. Instead, render a data attribute or hidden input in HTML and read it from JavaScript.

# Shell scripts

- Attempt to break Bash lines at 80 characters, but don't add complexity merely to fit within 80 characters.
  - For example, if a long URL forces the line beyond 80 characters, leave the URL intact rather than breaking it into artificial parts.
- Prefer passing information to shell scripts through command-line arguments rather than environment variables.
  - This applies to `dev-scripts/` interfaces. Application runtime configuration may use environment variables when that is the intended interface.
- When removing a command-line argument, delete it cleanly.
  - Do not add compatibility checks for the old interface.

# Testing

- After every code change, run `nix flake check` before presenting the solution to the user.
- After every code change, run `dev-scripts/git-hooks/pre-commit` before presenting the solution to the user.
- To run Go unit tests, run `./dev-scripts/run-go-tests`.
- Do not write to the filesystem in unit tests. End-to-end tests may write to the filesystem.
  - Design production interfaces around file readers or filesystem interfaces rather than the local filesystem.
- Avoid testing implementation-dependent call counts when you can assert a more meaningful signal that is part of the function contract.
- When writing tests to verify a bugfix, follow TDD conventions: write the test first, verify that it fails, fix the bug, and verify that the test passes.

## Tests must be correct by inspection

- Minimize complexity and abstraction in tests, as indirection makes it harder to verify correctness by simple inspection.
- Avoid loops except for standard table-driven tests in Go.
- Don't use helper functions that hide information needed to understand a test's correctness.
  - Helper functions in tests are acceptable as long as they don't hide critical values.
- If a function uses table-driven tests, use one table per function.

### Example: Bad test

```python
def setUp(self):
  database = MockDatabase()
  database.add_row({
      'username': 'joe123',
      'score': 150.0
    })
  self.account_manager = AccountManager(database)

def test_initial_score(self):
  initial_score = self.account_manager.get_score(username='joe123')
  self.assertEqual(150.0, initial_score)
```

This is a bad test because the reader can't determine from `test_initial_score` why the assertion is correct. They have to leave the test function to find the value `150.0`.

### Example: Good test with helper method

Helper methods in tests are acceptable as long as they don't hide critical values:

```python
def make_dummy_account(self, username, score):
  return Account(username=username,
                 name='Dummy User',         # <- OK: Buries values but they're
                 email='dummy@example.com', # <-     irrelevant to the test
                 score=score)

def test_increase_score(self):
  account_manager = AccountManager()
  account_manager.add_account(
    make_dummy_account(
      username='joe123',  # <- GOOD: Relevant values stay
      score=150.0))       # <-       in the test

  account_manager.adjust_score(username='joe123',
                               adjustment=25.0)

  self.assertEqual(175.0,
                   account_manager.get_score(username='joe123'))
```

## Playwright tests

### Test realistic user flows

- Playwright tests should begin with the user landing on the root route and navigating to other app pages by clicking page elements.
  - Tests should not simulate the user navigating the app by changing the URL manually.
  - Tests should assert that the URL changed when expected, but the app should cause the change, not the test.
- Run end-to-end tests with `nix build .#e2e-tests`.
  - Do not run Playwright directly unless the user explicitly asks for that workflow.

### Test user-visible behavior

- Test what end users see and interact with, not implementation details.
- Avoid assertions based on internal function names, CSS class names, or data structures.

### Test isolation

- Each test should be fully independent, with its own local storage, session storage, cookies, and data.
- Use `beforeEach` hooks for shared setup such as navigation or login rather than sharing state across tests.

### Use locators

- Prefer user-facing locators: role first, then text.
  - `page.getByRole('button', { name: 'Submit' })`
  - `page.getByText('Welcome')`
- Add `data-testid` attributes only as a last resort.
  - When neither role nor text identifies an element, give it a real accessible name or role instead of a test-only hook.
  - Don't add a production class or ID purely for the sake of testing.
- Avoid CSS selectors and XPath — the DOM changes frequently and they couple tests to implementation.
- Narrow locators to a specific section of the page using chaining and filtering:

  ```ts
  await page
    .getByRole("listitem")
    .filter({ hasText: "File 2" })
    .getByRole("button", { name: "Download" })
    .click();
  ```

### Use web-first assertions

- Use `expect(locator).toBeVisible()` and similar web-first assertions because they automatically wait for conditions to be met.
- Do not use `expect(await locator.isVisible()).toBe(true)` because `isVisible()` resolves immediately without waiting.

### Parallelism

- Playwright runs test files in parallel by default, so keep tests independent to take full advantage of this.
- For many independent tests within a single file, enable parallel mode explicitly:

  ```ts
  test.describe.configure({ mode: "parallel" });
  ```

# Documentation

- Use active voice in documentation.
  - Prefer clear `<subject> <verb> <object>` structure.
  - Especially avoid passive voice that obscures which component performs an action, such as "the token is passed."

# Git

## Pre-commit checks

- If Git pre-commit hooks fail, pause and fix the failures.
- Do not ignore pre-commit failures.
- Never bypass hooks with `--no-verify`.

## Commit message conventions

- Always write commit messages with single-quoted heredocs so Bash characters do not interpolate within the message.
- Make the first line describe the effect of the change, not the implementation details.
- Put the most important information first so reviewers can quickly understand the change.
- Use the body to explain motivation, user or client impact, and relevant background.
- Include searchable details such as exact error messages when they aid future debugging.
- Include relevant breaking changes, dependency rationale, and issue or commit references. Summarize linked bugs and references in the commit message; do not rely on links alone.
- When the diff is not self-evident, include validation instructions or known testing limitations.
- Use clear section headings for long messages.
- Leave out details that are obvious from the diff, such as file lists and mechanical implementation walkthroughs.
- Keep short-term review chatter, preview URLs, and ephemeral build artifacts out of commit messages.
- Do not bury critical maintenance constraints in commit messages.
  - Enforce them in code, tests, or documentation.

## Remotes

- Do not attempt to push, pull, or fetch from remotes unless the user instructs you to do so directly.
