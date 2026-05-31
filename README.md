# notch 🔖

Tagging releases by hand gets old fast, and it's easy to fat-finger a version or forget the changelog. `notch` does it for you: it looks at your [Conventional Commits](https://www.conventionalcommits.org/) since the last tag, works out what the next [SemVer](https://semver.org/) should be, writes the changelog, and creates the tag. One static binary, the only thing it needs around is `git`.

If your repo is a Helm chart or an npm package, it'll bump the version inside `Chart.yaml` / `package.json` too, in the same commit.

## How it works

The bump is whatever your commits imply:

| Commit | Bump |
| --- | --- |
| `fix: …` | patch |
| `feat: …` | minor |
| `feat!: …` · `BREAKING CHANGE` | major |

NOTE: **Before version 1.0.0 a breaking change only bumps the minor - the project is considering unstable.**

## Install

```sh
go install github.com/yoyrandao/notch@latest
```

**// TODO: installation documentation**

## Quick start

```sh
notch init          # write .notch.yaml with defaults
notch bump          # version, changelog, tag, push
notch bump --dry-run  # preview, touch nothing
```

Not sure what it'll do? `--dry-run` tells you and changes nothing:

```
# example --dry-run output

v0.5.0
dry-run: would update CHANGELOG.md, commit "chore(release): v0.5.0",
tag v0.5.0, push HEAD and v0.5.0 to origin
```

## Commands

- **`bump`** — the main event: figure out the version, update the changelog, commit, tag, push.
- **`init`** — drop a default `.notch.yaml` next to you.
- **`version`** — print which `notch` you're running.

The flags you'll actually reach for on `bump`: `--pre rc` for prereleases (the counter ticks up each time), `--as 1.0.0` to force a specific version, `--no-push` to keep it local, plus `--remote`, `--repo`, `--changelog`. Global ones: `--dry-run`, `-v/--verbose`, `-c/--config`. Anything on the command line wins over the config file.

## Configuration

`notch` reads `.notch.yaml` from the repo root. Nothing in it is required — here's everything with its default:

```yaml
repository: .

tag:
  prefix: "v"      # tag = prefix + version
  push: true

changelog:
  path: CHANGELOG.md

# Unwrap wrapped merge-commit subjects; capture group 1 is parsed.
# commit:
#   subject_pattern: '^Merged PR \d+: (.+)$'
```

That last bit is handy if your platform mangles commit subjects. Azure DevOps, for instance, turns a squash into `Merged PR 123: feat: …`. Point `subject_pattern` at it and `notch` pulls the real message out (capture group 1) before parsing.

## Tool-specific projects

No setup here — `notch` just looks for a known layout in the repo root and patches its version field:

| Project | File | Field |
| --- | --- | --- |
| Helm | `Chart.yaml` | `version` |
| npm | `package.json` | `version` |

Whatever it patches rides along in the release commit. If the file's there but the field isn't, it leaves it alone and carries on like an ordinary repo.

## Development

```sh
make build   # bin/notch
make test    # go test -race ./...
make vet
```

Tests aren't just unit-level — they spin up throwaway git repos and run the real thing end to end, tool-specific patching included.
