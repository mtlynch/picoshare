## Playwright (MCP)

- Start PicoShare before driving the UI:
  - `PS_VERSION="$(git describe --tags)" dev-scripts/build-backend dev`
  - `PS_SHARED_SECRET=somepassword ./bin/picoshare-dev -db data/store.db`
- Use the MCP Playwright server tools to drive the UI (for example,
  `mcp__playwright__browser_navigate` -> `mcp__playwright__browser_snapshot`).

## Development

- Set `GOCACHE` to `/tmp/go-cache` to avoid permission issues.
- Set `GOPATH` to `/tmp/go-workspace` to avoid permission issues.
