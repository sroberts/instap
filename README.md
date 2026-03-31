# instap

A command-line tool and TUI for interacting with the Instapaper API, written in Go.

## Features
- **CLI Mode:** Save and list bookmarks directly from your terminal.
- **TUI Mode:** Browse your bookmarks in an interactive terminal interface.
- **Reader View:** Fetch article content and render it as Markdown in your terminal.
- **Management:** Archive, Delete, Star/Unstar, and Move bookmarks to folders.
- **Auth:** Secure xAuth flow to handle Instapaper credentials.

## Installation
```bash
go build -o instap main.go
```

## Setup
1.  **Configure API Credentials:**
    ```bash
    ./instap auth set-credentials --key YOUR_KEY --secret YOUR_SECRET
    ```
2.  **Log In:**
    ```bash
    ./instap auth login
    ```

## Usage
- **Launch TUI:** `./instap`
- **Save Bookmark:** `./instap save <url>`
- **List Recent:** `./instap list`

## TUI Shortcuts
- `enter`: Reader view (Markdown)
- `o` or `shift+enter`: Open in browser
- `a`: Archive
- `s`: Star/Unstar
- `m`: Move to folder
- `d`: Delete
- `r`: Refresh list
- `q`: Quit
- `j`/`k` or arrows: Navigate / Scroll reader view
