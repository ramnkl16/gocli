# gocli

> One CLI for Jira, GitHub, Copilot PR review, and deployments.

`gocli` is a single static Go binary that gives developers a unified
"cockpit" for the work they normally do across many tabs and tools.

## What it does today (MVP)

| Area     | Commands                                                                                  |
| -------- | ----------------------------------------------------------------------------------------- |
| Auth     | `auth login`, `auth status`, `auth logout`                                                |
| Jira     | `jira list`, `jira view`, `jira open`, `jira transition`, `jira branch` (see **Jira: branch** below) |
| GitHub   | `gh prs`, `gh pr-view <n>`, `gh pr-checkout <n>`, `gh issues`                             |
| PR       | `pr create` (open a PR on GitHub), `pr review <n>` (AI review via GitHub Copilot / Models) |
| Deploy   | `deploy list`, `deploy run <pipeline>`, `deploy validate` — driven by `deploy.yml`        |

Tokens are kept in the OS keyring (Windows Credential Manager / macOS
Keychain / Secret Service on Linux). Env vars (`JIRA_API_TOKEN`,
`GITHUB_TOKEN`) are an automatic fallback for CI.

## Install

Requires Go 1.22+.

```bash
go install github.com/yourorg/gocli@latest
```

Or build from a clone:

```bash
git clone https://github.com/yourorg/gocli
cd gocli
go build -o gocli .
```

The optional `gh` CLI is used as a fallback for `gh pr-checkout` and
`gh copilot` shell-outs.

## First run

```bash
gocli auth login
gocli auth status
```

`auth login` collects:

- **Jira** — base URL (e.g. `https://acme.atlassian.net`), email, project
  key, API token (created at <https://id.atlassian.com/manage-profile/security/api-tokens>).
  The **Sprint** column uses your Jira site’s **Sprint** custom field. The
  field id is **discovered automatically** (GET `/rest/api/2/field`, first
  custom field named “Sprint”). Override in `~/.gocli/config.yaml` as
  `jira.sprint_field: customfield_XXXXX`, or `GOCLI_JIRA_SPRINT_FIELD`. Set
  `jira.sprint_field: auto` or leave it unset to keep discovery; use `none` in
  the env to skip requesting the field.
- **GitHub** — username, default repo (`owner/name`, optional), personal
  access token (scopes: `repo`, `read:org`).
- **AI provider** — `github-models` (default; uses your GitHub token
  against <https://models.github.ai>) or `copilot` (shells out to the
  `gh copilot` extension).

Config lives at `~/.gocli/config.yaml` (override with `GOCLI_CONFIG`).
Secrets never touch disk.

### Project-local environment with `.env`

For per-project setup (or for CI / containers where there is no OS
keyring), `gocli` automatically loads `.env` and `.env.local` from the
current directory at startup. Real OS environment variables always win
over `.env` values.

```bash
cp .env.example .env       # macOS / Linux
Copy-Item .env.example .env # PowerShell
# edit .env and fill in JIRA_API_TOKEN / GITHUB_TOKEN
gocli auth status
```

Both `.env` and `.env.local` are gitignored (only `.env.example` is
committed). Set `GOCLI_NO_DOTENV=1` in your shell to disable `.env`
loading entirely.

## Jira: `branch`

From a **git** working tree, `gocli jira branch <ISSUE-KEY> --type <bug|fea|chg>`:

1. Fetches the issue summary from Jira and builds a branch name:
   `{type}-{KEY}-{slug}` where `slug` is the first 20 characters of the
   title, lowercased, with spaces and punctuation turned into single
   hyphens (e.g. `bug-LPAD-26763-login-form-validation`).
2. Transitions the issue to **In Progress** (override with
   `--to-status "Name"` if your workflow uses a different status).
3. Runs `git checkout -b <branch>` in a repository. By default that is the
   **current directory**; use `--workdir <path>` (short: `-C`, same idea as
   `git -C`) to use another clone, e.g. `C:\work\my-app` on Windows or
   `~/src/my-app` on macOS/Linux.

`gocli` still talks to Jira the same way; only **git** runs in the chosen
folder.

Refuses a dirty working tree in that repo unless `--allow-dirty`.

Use `--dry-run` to print the branch name without touching git or Jira, or
`--no-transition` to only create the branch.

## Examples

```bash
gocli jira list                               # my open issues (includes SPRINT if Jira Software sprints are present)
gocli jira list --assignee bob                # shorthand; see `jira.assignee_aliases` in ~/.gocli/config.yaml
gocli jira list -u teammate@corp.com          # `-u` is short for `--assignee` (literal Jira identifier)
gocli jira list --status "In Review"

Optional assignee shortcuts in config (same file `auth login` writes):

```yaml
jira:
  assignee_aliases:
    bob: bob@corp.com
    jd: JIRA_ACCOUNT_ID_HERE
```
gocli jira view LPAD-26763
gocli jira transition LPAD-26763 "In Progress"

# create branch bug-LPAD-26763-…, move ticket to "In Progress", git checkout -b
gocli jira branch LPAD-26763 --type bug
gocli jira branch LPAD-26763 --type fea --to-status "In Progress"
gocli jira branch LPAD-26763 --type chg --dry-run   # show branch name only
gocli jira branch LPAD-26763 --type bug --no-transition   # branch only, no Jira
gocli jira branch LPAD-26763 --type fea --workdir C:\path\to\clone   # or: -C C:\path\to\clone

gocli gh prs                                  # PRs in current repo
gocli gh prs --mine --state open
gocli gh pr-view 482                          # title, checks, mergeability
gocli gh pr-checkout 482

gocli pr create --title "Fix login"           # or omit --title to use last commit subject
gocli pr create --base main --head my-feature
gocli pr create -R myorg/therepo --workdir C:\other\clone --draft
gocli pr review 482                           # AI review, prints markdown

gocli deploy list -f examples/deploy.yml
gocli deploy run staging --dry-run
gocli deploy run docker-release
```

## `deploy.yml`

A pipeline is an ordered list of steps. Each step has a `type` handled
by a pluggable provider:

- `script` — shell command (auto-picks `powershell` on Windows, `bash`
  elsewhere; override with `shell: pwsh|bash|sh|cmd`).
- `docker` — `docker build` / `push` / `run` (set `image`, `context`,
  `dockerfile`, `push`, `args`).
- `k8s` — wraps `kubectl` (`action: apply | delete | rollout`,
  `manifest`, `namespace`, `args`).

See [`examples/deploy.yml`](examples/deploy.yml) for the full schema.
Adding a new provider is one `deploy.Register("name", impl)` call.

## Why "gocli"?

Because flipping between Jira, GitHub, the dashboard, and `kubectl` is
the slowest part of shipping a small change. This collapses the loop:

```text
pick ticket  →  branch / PR  →  AI review  →  deploy  →  transition ticket
```

…all without leaving your terminal.
