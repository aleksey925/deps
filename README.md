# deps

TUI package manager for Python. Browse installed packages, check for updates, and install
new ones from PyPI — all from a single interactive terminal interface.

- [Installation](#installation)
- [How to Use](#how-to-use)

## Installation

Download the latest release from [releases](https://github.com/aleksey925/deps/releases) and install it manually
or you can run the following commands to install the latest version to `~/.local/bin`:

```bash
VERSION=$(curl -sL -o /dev/null -w '%{url_effective}' https://github.com/aleksey925/deps/releases/latest | sed 's/.*\/v//')
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -#L "https://github.com/aleksey925/deps/releases/download/v${VERSION}/deps_${VERSION}_${OS}_${ARCH}.tar.gz" | tar xz -C ~/.local/bin deps
```

Also, you can build it from source:

```bash
git clone https://github.com/aleksey925/deps.git
cd deps
make install  # copies to ~/.local/bin
```

Make sure `~/.local/bin` is in your PATH:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

## How to Use

```bash
cd your-project
deps
```

`deps` auto-detects the Python interpreter (checks `.venv/` first, then `PATH`) and package manager (`uv` or `pip`).

**Flags:**

| Flag              | Description                                    |
| ----------------- | ---------------------------------------------- |
| `-p`, `--python`  | Path to Python interpreter                     |
| `-m`, `--manager` | Package manager: `pip` or `uv` (auto-detected) |
| `-v`, `--version` | Print version and exit                         |

**Keyboard Shortcuts:**

| Key            | Action                               |
| -------------- | ------------------------------------ |
| `↑/↓` or `k/j` | Navigate packages                    |
| `/`            | Search installed packages            |
| `Tab`          | Switch between local and PyPI search |
| `Enter`        | Update selected / confirm            |
| `→` or `l`     | Browse available versions            |
| `i`            | Show package info                    |
| `Space`        | Select package                       |
| `a`            | Select all outdated                  |
| `A`            | Select all                           |
| `s`            | Toggle sort (name / outdated first)  |
| `Ctrl+R`       | Reload package list / PyPI index     |
| `Esc`          | Back / clear selection               |
| `q`            | Quit                                 |
