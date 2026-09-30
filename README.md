# shell-history-lint

Your shell history accumulates years of commands, and some of them are
things you'd rather not have sitting around in plaintext: a `curl` with a
bearer token pasted straight into the URL, a password typed after `mysql -p`,
a `curl | bash` install script, an `rm -rf` you got lucky with. That file
gets backed up, synced to a dotfiles repo, attached to a support ticket, or
just read by whatever else has access to your home directory.

`shell-history-lint` scans a history file line by line and reports anything
that looks risky, with the line number so you can go fix it (or think twice
before running it again).

## Usage

```
go build -o shell-history-lint .

./shell-history-lint ~/.zsh_history
12: [error] credential-in-url: URL contains an embedded username and password
47: [warning] inline-secret: credential-looking value written directly in a command
203: [error] pipe-to-shell: downloading and piping straight into a shell
```

It also reads from stdin, so it works on live history without a file:

```
history | ./shell-history-lint
```

Exit code is `0` when nothing was flagged, `1` when there's at least one
finding, `2` on a real error (bad file path, read failure).

## Supported formats

- Plain history: one command per line.
- Bash extended history (`HISTTIMEFORMAT`): a `#<unix-timestamp>` comment
  line immediately followed by the command it timestamps.
- Zsh extended history (`EXTENDED_HISTORY`): `: <start>:<duration>;<command>`.

By default the linter looks at each line and figures out which of these it
is. That's almost always right, but the two timestamped formats can be
ambiguous - a plain bash command that starts with `: 123:456;` (`:` is a
real no-op builtin) reads as a zsh timestamp, and a literal `#1700000000`
command in a zsh history reads as a bash one. Pass `--format bash` or
`--format zsh` to skip the guessing and parse strictly as one or the other:

```
./shell-history-lint --format zsh ~/.zsh_history
```

## JSON output

`--output json` prints one JSON object per finding, one per line, for
editors and CI to consume:

```
./shell-history-lint --output json ~/.zsh_history
{"line":12,"rule":"credential-in-url","severity":"error","message":"URL contains an embedded username and password","timestamp":1700000000}
```

`timestamp` is the unix time from the history entry and is left out when the
file doesn't carry one. The command text is never included, since the
matching lines are the ones most likely to contain a secret. Output is JSON
Lines rather than an array so findings still appear as they are found.
The exit codes are the same as in text mode.

## Config file

By default every rule below runs at its listed severity. To disable specific
rules or change their severity, pass `--config` with a JSON file:

```json
{
  "disable": ["chmod-world-writable"],
  "severity": {
    "inline-secret": "error"
  }
}
```

```
./shell-history-lint --config ~/.histlintrc.json ~/.zsh_history
```

`disable` is a list of rule ids to skip entirely. `severity` maps a rule id
to `"error"` or `"warning"`, overriding its default. Referencing a rule id
that doesn't exist, or a severity other than those two values, is an error -
that's almost always a typo in the config, not intentional.

## Why streaming matters here

History files are append-only and can grow for years. The linter never
reads the whole input into memory - it pulls one line at a time off a
buffered reader, checks it against the rule set, and writes any findings
immediately. Memory use stays flat whether the file is 200 lines or 2GB.

## Current rules

| id | severity | catches |
|---|---|---|
| `dangerous-rm` | error | `rm -rf` (or similar) aimed at `/`, `~`, `$HOME`, or `*` |
| `pipe-to-shell` | error | `curl`/`wget` piped into `sh`/`bash`/`zsh` |
| `credential-in-url` | error | a URL with `user:password@` embedded in it |
| `inline-secret` | warning | `PASSWORD=`, `TOKEN=`, `API_KEY=`, etc. set to a literal value |
| `chmod-world-writable` | warning | `chmod 777` / `chmod a+rwx` |
| `aws-access-key` | error | an AWS access key ID |
| `github-token` | error | a GitHub personal access token |
| `slack-token` | error | a Slack bot token |
| `google-api-key` | error | a Google API key |
| `stripe-live-key` | error | a live Stripe secret key |
| `anthropic-api-key` | error | an Anthropic API key |

## License

MIT, see [LICENSE](LICENSE).
