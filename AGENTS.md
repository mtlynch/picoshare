## Playwright (MCP)

- Enter the dev shell so Playwright sees the Nix-provided browsers.
- Start PicoShare before driving the UI:
  - `PS_VERSION="$(git describe --tags)" dev-scripts/build-backend dev`
  - `PS_SHARED_SECRET=somepassword ./bin/picoshare-dev -db data/store.db`
- Use the MCP Playwright server tools to drive the UI (for example,
  `mcp__playwright__browser_navigate` -> `mcp__playwright__browser_snapshot`).
- If the MCP server cannot find a browser, set
  `PLAYWRIGHT_BROWSERS_PATH` to the flake-provided path in
  `flake.nix` and retry.

## Development

- Set `GOCACHE` to `/tmp/go-cache` to avoid permission issues.
- Set `GOPATH` to `/tmp/go-workspace` to avoid permission issues.
