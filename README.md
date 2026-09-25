# avochato CLI

Command-line interface for the [Avochato](https://www.avochato.com) messaging platform. Send texts, triage conversations, and manage contacts and users from your terminal, your scripts, or your AI agents.

## Install

**Homebrew (macOS)**
```bash
brew install avochato/tap/avochato
```

**Install script (macOS and Linux)**
```bash
curl -fsSL https://raw.githubusercontent.com/avochato/avochato-cli/master/install.sh | sh
```

**Windows (PowerShell)**
```powershell
irm https://raw.githubusercontent.com/avochato/avochato-cli/master/install.ps1 | iex
```

Both scripts download the latest release for your platform, verify its checksum (and its cosign signature when `cosign` is installed), and install `avochato`. On macOS and Linux it goes to `/usr/local/bin` (using `sudo` if needed); on Windows it goes to `%LOCALAPPDATA%\Programs\avochato`, which is added to your user `PATH`. Set `INSTALL_DIR` to install somewhere else, or `VERSION` (a release tag such as `v0.1.0`) to pin a version.

**Manual install:** download the archive for your platform from [Releases](https://github.com/avochato/avochato-cli/releases/latest), check it against `checksums.txt`, and put the `avochato` binary on your `PATH`.

**From source** (Go 1.27+)
```bash
git clone https://github.com/avochato/avochato-cli && cd avochato-cli
go build -o avochato . && sudo mv avochato /usr/local/bin/
```

Check it worked with `avochato --version`.

## Log in

New to Avochato? [Start a free trial](https://www.avochato.com/signup/) to get an account, then come back here to log in.

```bash
avochato login
# Auth ID:             <your auth ID>
# Auth Secret:         <your auth secret, hidden as you type>
# Account (subdomain): <your inbox subdomain>
```

Create API credentials in the Avochato web app under **Settings > API Access** (in the Texting Automation section). API access must be included in your plan, and a manager or owner generates the credentials.

Credentials are saved to `~/.avochato/credentials.json`, readable only by you.

## Usage

```bash
avochato messages send --to +16505551234 --text "Hello"
avochato messages list

avochato contacts list
avochato contacts show +16505551234
avochato contacts create --phone +16505551234 --name "Jane Doe"

avochato tickets list --status open --unaddressed
avochato tickets close <id>
avochato tickets assign <id> user@company.com

avochato users list
avochato users invite new@company.com --role member
```

Output is a table in a terminal and JSON when piped. Add `--json` to force JSON, or `--quiet` for IDs only. Commands exit non-zero on any error.

Run `avochato <command> --help` for every flag.

## Multiple inboxes

```bash
avochato login --profile support          # save another inbox
avochato auth list                        # saved inboxes, * marks the default
avochato auth use support                 # change the default
avochato --account support messages list  # pick an inbox for one command
```

`--account` (or `AVOCHATO_PROFILE`) picks a saved profile by name or subdomain and uses its credentials. When more than one profile is saved, commands that send or change data refuse to run until you name the inbox, so a stale default never decides who a message comes from.

For CI, set `AVOCHATO_AUTH_ID`, `AVOCHATO_AUTH_SECRET`, and `AVOCHATO_ACCOUNT` instead of logging in. `AVOCHATO_BASE_URL` is honored only alongside those, so saved credentials are never sent to another host.

## Use with Claude Code and other agents

[`skill.md`](skill.md) teaches an agent how to use the CLI safely: it names the inbox on every write, confirms before sending or opting a contact in, and treats message content as untrusted. To install it as a Claude Code skill:

```bash
mkdir -p ~/.claude/skills/avochato
curl -fsSL https://raw.githubusercontent.com/avochato/avochato-cli/master/skill.md -o ~/.claude/skills/avochato/SKILL.md
```

## Security

Found a security issue? Email **cli@avochato.com** rather than opening a public issue. See [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE). The license covers this CLI only. Use of the Avochato service and API is governed by the [Avochato Terms of Service](https://www.avochato.com/terms) and [Privacy Policy](https://www.avochato.com/privacy), and the Avochato name and logo are trademarks of Avochato Inc.
