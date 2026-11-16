# CI and release

## Local validation

Run `nix flake check` to validate the flake without CI credentials. The flake exposes the Go, frontend, workflow, formatting, SQL, and browser end-to-end checks. The Go check runs `dev-scripts/run-go-tests --full` and retains `.coverage.out` and `.coverage.html` in its package output. The end-to-end check runs Playwright with Chromium and Firefox and retains its Playwright report in its package output.

`dev-scripts/build-all-flake-targets` runs `nix flake check` with `--max-jobs auto` by default. Any arguments replace those defaults and pass through to `nix flake check`. It then runs `dev-scripts/upload-to-nixci-cache`, which copies `checks.<system>.all` to the NixCI cache. Upload errors fail the runner; it does not silently skip an unsuccessful upload. The `all` check is a link farm of every package for the current system, so it builds the package closure rather than merely the checks listed under `checks`.

Local validation does not need secrets. NixCI automatically discovers the flake through its GitHub integration; the repository does not need a separate CI trigger workflow. Hosted runs need the repository's NixCI integration enabled and credentials provisioned outside this checkout. NixCI's Fly deploy requires `FLY_API_TOKEN`; release publishing requires `GH_TOKEN`, `DOCKERHUB_USERNAME`, and `DOCKERHUB_ACCESS_TOKEN`. NixCI cache access also requires externally provisioned cache authorization. Never put real credentials in the repository.

## Release and deployment

The release deploy in `nix-ci.nix` selects `v*` tags. NixCI publishing accepts only semantic-version tags (`vMAJOR.MINOR.PATCH`) whose checked-out commit matches the tag, and expects the GitHub release to exist already. It publishes these binary archives:

- `picoshare-vVERSION-linux-amd64.tar.gz`
- `picoshare-vVERSION-linux-arm64.tar.gz`
- `picoshare-vVERSION-linux-armv7.tar.gz`

It also publishes Docker Hub images tagged `vVERSION-amd64`, `vVERSION-arm64`, `vVERSION-armv7`, and a multi-platform `vVERSION` tag. After verifying the manifest's three platforms and uploading the GitHub release archives, it moves `latest` to that multi-platform release image. A failed upload does not advance `latest`.

NixCI deploys to Fly only on the default branch, after qualification, using the Fly token. The deploy builds and uploads an image tagged with the current commit; it does not use the Docker Hub `latest` tag.
