# uxwing-mcp

Free [UXWing](https://uxwing.com) icons for your MCP client: search them, **see the previews**,
open them on the website, and download the SVG and PNG files — all from the chat.

## Downloads

Binaries from the newest `main` build, pure Go with cgo disabled:

| Platform | Download |
| --- | --- |
| Linux x86_64 | [`uxwing-mcp-linux-amd64`](https://nightly.link/d-led/uxwing-mcp/workflows/ci/main/uxwing-mcp-linux-amd64.zip) |
| Linux arm64 | [`uxwing-mcp-linux-arm64`](https://nightly.link/d-led/uxwing-mcp/workflows/ci/main/uxwing-mcp-linux-arm64.zip) |
| Windows x86_64 | [`uxwing-mcp-windows-amd64.exe`](https://nightly.link/d-led/uxwing-mcp/workflows/ci/main/uxwing-mcp-windows-amd64.zip) |
| macOS, Apple silicon and Intel | [`uxwing-mcp-darwin-universal`](https://nightly.link/d-led/uxwing-mcp/workflows/ci/main/uxwing-mcp-darwin-universal.zip) |

Each link is a zip holding the binary: unzip it, `chmod +x` it on Linux and macOS, then register it
with your client. Versioned releases are on the [releases page](https://github.com/d-led/uxwing-mcp/releases).

## Add it to your MCP client

The binary registers itself and merges with the servers already in the file:

```sh
uxwing-mcp install --client vscode      # VS Code user profile, all workspaces
uxwing-mcp install --client workspace   # this folder: .vscode/mcp.json
uxwing-mcp install --client claude      # Claude Desktop

uxwing-mcp uninstall --client vscode    # remove the entry again
```

`install` records the absolute path of the binary and the icon directory (the `UXWING_ICON_DIR`
value, else `icons`). `--name` picks another entry name, `--dir` another download directory.
Running it twice changes nothing, and a config that is not valid JSON is left untouched.

Prefer to edit the config by hand? Add an entry to the `servers` section (`mcpServers` for Claude
Desktop):

```json
{
  "servers": {
    "uxwing": {
      "type": "stdio",
      "command": "/absolute/path/to/uxwing-mcp",
      "env": { "UXWING_ICON_DIR": "/absolute/path/to/project/assets/icons" }
    }
  }
}
```

## What you can ask for

| Tool | What you get |
| --- | --- |
| `search_icons` | Icons matching your words, each with its uxwing.com page address and a 512×512 PNG preview shown right in the chat. |
| `get_icon` | One icon in detail: page address, the formats UXWing serves, a preview, and the variations of the same idea (black, outline, …). |
| `download_icon` | The SVG and/or PNG files written into your icon directory, with the saved paths and a preview of what was written. |

UXWing serves **SVG and PNG**. When the site hides its PNG button but still serves the PNG file,
the server finds the file anyway.

## Where icons are saved

`download_icon` writes into the directory the server was started with: `UXWING_ICON_DIR`, else
`icons` in the working directory. A call can also name its own `dir`.

## Licence

The code is [MPL-2.0](LICENSE). The icons belong to [UXWing](https://uxwing.com) and are free to
use under their [licence](https://uxwing.com/license/); this tool only downloads them for you.

## Building from source

```sh
go install github.com/d-led/uxwing-mcp@latest
```

Working on the server itself? [docs/development.md](docs/development.md) has the scripts, the
layout, the testing approach and the CI setup.

