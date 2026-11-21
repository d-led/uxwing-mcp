# Development

Everything a maintainer needs: the scripts, the layout, the testing approach and the CI setup.
Users want the [README](../README.md).

## Prerequisites

Go 1.27 or newer. The linters are optional; `scripts/lint.sh --install` fetches them.

## Scripts

Every task is a script, so a red build can be reproduced locally with one command:

| Script | Task |
| --- | --- |
| `scripts/build.sh` | Builds `bin/uxwing-mcp` with the git version stamped in. |
| `scripts/test.sh` | Runs the tests with the race detector and writes `build/coverage.out`. `--json` also writes `build/test-results.json` for the CI test reporter. |
| `scripts/lint.sh` | gofmt, `go vet`, staticcheck and golangci-lint, each one run if installed. `--install` installs them. |
| `scripts/crap.sh` | CRAP gate: every function must score at most 10. Needs `go install github.com/padiazg/go-crap@latest`. |
| `scripts/ci.sh` | Everything above, in the order CI runs it. |

## Layout

```
main.go                     subcommands: serve (default, stdio), install, uninstall
internal/install            writes the server into VS Code / Claude Desktop configs
internal/uxwing             the site: value types, pure parsers, HTTP client, file store
internal/mcpserver          the MCP tools, their handlers and result formatting
internal/uxwing/testdata    captured page shapes the parser tests run against
```

Pieces worth knowing:

- `uxwing.Slug` and `uxwing.Format` parse at the boundary, so an illegal value cannot travel far.
- The parsers in `internal/uxwing/page.go` take a `*goquery.Document` and return domain values;
  they never touch the network. Tests feed them `testdata` fixtures.
- `uxwing.Client` talks to the site through the `Getter` interface, which is why the client tests
  run against an `httptest` server built from the same fixtures.
- `uxwing.Store` is stateless: `Save(dir, asset, data)` creates the directory and refuses file
  names that could escape it.
- `mcpserver` depends on the small `Service` and `Saver` interfaces, so the tool tests drive the
  real MCP protocol over in-memory transports with fakes underneath.

## UXWing quirks baked into the parsers

- Icon cards are `a.post-thumbnail[href]` on search pages and `div.post-thumbnail a[href]` in the
  "Related Icons" block; the same selector list reads both. Related icons are the variations of
  the idea (black, outline, …).
- Download buttons are `a.downloadicon[href][download]`; the real file lives under
  `/wp-content/themes/uxwing/download/<category>/<file>`.
- UXWing serves **SVG and PNG only** (`.jpg`, `.webp`, `.ico`, `.eps`, `.pdf` all 404), and the
  `download.php?file=` endpoint is gone.
- The PNG button is frequently **commented out** while the `.png` file is still served, so
  `withOtherFormats` derives the missing format from the sibling file name.
- The heading carries a marketing suffix ("Robot Icon SVG Vector"), which `pageTitle` strips.

## Testing

- Behaviour, not implementation: the tool tests call the MCP protocol and check what a user sees
  (structured output, preview images, written files), the client tests check downloaded bytes.
- Fixtures are real page shapes; `{{BASE}}` is the only placeholder, rewritten to the live site in
  parser tests and to the `httptest` server in client tests.
- `scripts/crap.sh` keeps cyclomatic complexity times uncovered lines at 10 or below per function.
  Splitting a function is preferred over a clever guard.

## CI

`.github/workflows/ci.yml`:

- `lint` — gofmt, `go vet`, golangci-lint (built with the repo toolchain: a prebuilt binary can
  lag behind the export data format).
- `test` — `scripts/test.sh --json`, coverage in the job summary, results uploaded as the
  `test-results` artifact.
- `crap` — the CRAP gate.
- `latest-binaries` / `latest-macos-universal` — on pushes to `main` only, build the four binaries
  (`linux-amd64`, `linux-arm64`, `windows-amd64.exe`, `darwin-universal` with `CGO_ENABLED=0`) and
  upload each one as an artifact.

`.github/workflows/test-report.yml` turns the `test-results` artifact into a check run. It runs on
`workflow_run` so pull requests from forks get a report too.

The README links each binary through `nightly.link`, which serves the artifact of the newest `ci`
run on `main` without a login:

```
https://nightly.link/d-led/uxwing-mcp/workflows/ci/main/<artifact-name>.zip
```

Artifacts are named `uxwing-mcp-<goos>-<goarch>` (plus `uxwing-mcp-darwin-universal`) and never
carry the `.exe` suffix, because the artifact name is the last path segment of the link. Keep the
names stable: a rename breaks the README URL until nightly.link picks up the next run.

`.github/workflows/goreleaser.yml` builds versioned releases from `v*.*.*` tags with
`.goreleaser.yaml`. The binary's version comes from `-X main.version`, set by `scripts/build.sh` and
GoReleaser.

## Adding a tool

1. Add the capability to `uxwing.Client` (with a test against the fake site).
2. Add the method to the `mcpserver.Service` interface.
3. Add args and output value types plus a handler in `internal/mcpserver/tools.go`, and register
   the tool in `NewServer`.
4. Test it through the protocol in `server_test.go`; run `scripts/ci.sh`.
