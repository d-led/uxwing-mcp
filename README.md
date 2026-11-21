# uxwing-mcp

An [MCP](https://modelcontextprotocol.io) server for [UXWing](https://uxwing.com) icons: search
them, look at previews, open the icon on the website, and download the SVG and PNG files.

## Tools

| Tool | What it does |
| --- | --- |
| `search_icons` | Searches uxwing.com. Returns each icon's slug, title, page address and a 512×512 PNG preview, and attaches the preview images so the user can see them. |
| `get_icon` | Shows one icon: its page address, the formats UXWing serves, a PNG preview, and the variations UXWing offers for the same idea (for example the black or outline drawing). |
| `download_icon` | Writes the SVG and/or PNG files into a directory and reports the saved paths. Returns the downloaded PNG as a preview. |

Formats: UXWing serves **SVG** and **PNG** only. The page sometimes comments its PNG button out,
while the PNG file itself is still served; the server therefore derives the missing format from
the file the page does offer.

## Install

```sh
git clone https://github.com/d-led/uxwing-mcp
cd uxwing-mcp
scripts/build.sh          # produces bin/uxwing-mcp
```

or

```sh
go install github.com/d-led/uxwing-mcp@latest
```

## Use

The server speaks MCP on stdin/stdout. Point your client at the binary:

```json
{
  "servers": {
    "uxwing": {
      "type": "stdio",
      "command": "/absolute/path/to/bin/uxwing-mcp",
      "env": { "UXWING_ICON_DIR": "/absolute/path/to/project/assets/icons" }
    }
  }
}
```

`UXWING_ICON_DIR` is where downloads land unless a `download_icon` call names its own `dir`.
It defaults to `icons` in the working directory.

`uxwing-mcp --version` prints the build version.

## Development

Every task is a script:

| Script | Task |
| --- | --- |
| `scripts/build.sh` | Build `bin/uxwing-mcp` with the git version stamped in. |
| `scripts/test.sh` | Run the tests with the race detector and write `build/coverage.out`. `--json` writes `build/test-results.json` for the CI reporter. |
| `scripts/lint.sh` | gofmt, go vet, staticcheck and golangci-lint when installed. `--install` installs them. |
| `scripts/crap.sh` | CRAP gate: every function must score at most 10 (go-crap). |
| `scripts/ci.sh` | Everything CI runs, in the same order. |

CI (`.github/workflows/ci.yml`) runs lint, tests and the CRAP gate on every push and pull
request, publishes the test results as a check run (`.github/workflows/test-report.yml`), and
uploads fresh binaries for `main`. Tagging `v*.*.*` builds the release with GoReleaser.

## Layout

```
main.go                    wiring: client, tools, stdio transport
internal/uxwing            the site: parsing, HTTP, downloads, file store
internal/mcpserver         the MCP tools and their handlers
```

The parsers are pure functions over a parsed HTML document, so the tests pin the real page
shapes (captured into `internal/uxwing/testdata`) without touching the network.

## Licence

The code is MIT licensed. The icons belong to [UXWing](https://uxwing.com) and are free to use
under their [licence](https://uxwing.com/license/); this tool only downloads them for you.
