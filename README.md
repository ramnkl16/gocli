# gocli

> One CLI for Jira, GitHub, Copilot PR review, and deployments.

`gocli` is a single static Go binary that gives developers a unified
"cockpit" for the work they normally do across many tabs and tools.

---

### From source (Go)

Requires Go 1.22+.

## 🚀 Quick Start

### 1. Install

Published on [GitHub Releases](https://github.com/yourorg/gocli/releases). Asset names are stable (no version in the filename), so you can use **latest** URLs or the scripts below.

Replace `yourorg/gocli` with your real `owner/repo`, or set `GOCLI_GITHUB_REPO` when using the scripts.

```bash
export GOCLI_GITHUB_REPO='yourorg/gocli'
```

**Windows (PowerShell):**
```powershell
iex "& { $(irm https://raw.githubusercontent.com/yourorg/gocli/main/scripts/install.ps1) }"
```
**macOS / Linux:**
```bash
curl -sSfL https://raw.githubusercontent.com/yourorg/gocli/main/scripts/install.sh | bash
```

### 2. Authenticate
Run the login command and follow the prompts to connect your Jira and GitHub accounts:
```bash
gocli auth login
 
 -- Jira API token (created at <https://id.atlassian.com/manage-profile/security/api-tokens>).
 -- Gitub API tocken (created at https://github.com/settings/tokens) with repo scope (selcet the repo checkbox).
 -- GitHub default repo (Optional).
 -- AI provider (Optional).
 -- AI model (Optional).
 

gocli auth status
 -- status should be like this:

 SETTING             │ VALUE                              │ OK  
─────────────────────┼────────────────────────────────────┼─────
 Jira URL            │ https://sample.atlassian.net       │ yes
 Jira email          │ [EMAIL_ADDRESS]                    │ yes
 Jira token          │ JIRA_API_TOKEN                     │ yes
 GitHub user         │ GitHub_User                        │ yes
 GitHub default repo │                                    │ no
 GitHub token        │ GITHUB_TOKEN                       │ yes
 AI provider         │ github-models                      │ yes
 AI model            │ openai/gpt-4o-mini                 │ yes
 
```

---

### Build `dist/` with GoReleaser

From the repo root, with [GoReleaser](https://goreleaser.com/) installed and on your `PATH`:

```bash
goreleaser release --snapshot --clean
```

Writes archives and checksums under **`dist/`** (local snapshot; does not require a release tag).

Compile only (binaries under `dist/` without full release packaging):

```bash
goreleaser build --snapshot --clean
```

To **publish** a GitHub Release from your machine, set `GITHUB_TOKEN` and run `goreleaser release --clean` on a tagged commit, or push a tag like `v0.2.0` and let [`.github/workflows/release.yml`](.github/workflows/release.yml) run GoReleaser in CI.

The optional `gh` CLI is used as a fallback for `gh pr-checkout` and
`gh copilot` shell-outs.

## 🔄 The Developer Loop

This is the recommended workflow to move faster:

1.  **Pick Work**: `gocli jira list` (Find your open tickets)
2.  **Start Coding**: `gocli jira branch KAN-224 --type fea` (Creates git branch & moves ticket to "In Progress")
3.  **Create PR**: `gocli pr create` (Creates GitHub PR from your commits)
4.  **AI Review**: `gocli pr review 482` (Get instant AI feedback on your changes)
5.  **Deploy**: `gocli deploy run staging` (Run your defined deployment pipeline)

---

## 📖 Command Reference

| Area | Command | Description |
| :--- | :--- | :--- |
| **Jira** | `jira list` | List your active issues |
| | `jira view <KEY>` | View issue details & description |
| | `jira open <KEY>` | Open issue in browser |
| | `jira transition <KEY> <STATUS>` | Move ticket to Done, Review, etc. |
| **GitHub** | `gh prs` | List open PRs in current repo |
| | `gh pr-checkout <N>` | Checkout a PR locally |
| **AI** | `pr review <N>` | Get AI feedback on a PR |
| **Deploy** | `deploy run <NAME>` | Execute a pipeline from `deploy.yml` |

---

## ⚙️ Configuration

- **Config File**: `~/.gocli/config.yaml` (Base URL, Email, Project keys)
- **Secrets**: Stored securely in your OS Keyring (Windows Credential Manager / Keychain).
- **Project Overrides**: You can use a local `.env` file for project-specific `JIRA_API_TOKEN` or `GITHUB_TOKEN`.

---

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

## 💡 Examples

```bash
gocli jira list                               # my open issues
gocli jira list --assignee bob                # shorthand; see `jira.assignee_aliases` in config
gocli jira list -u teammate@corp.com          # filter by literal Jira identifier
gocli jira list --status "In Review"

gocli jira view LPAD-26763
gocli jira view LPAD-26763 --markdown         # or -m; copy output into Cursor Agent
gocli jira transition LPAD-26763 "In Progress"

# create branch bug-LPAD-26763-..., move ticket to "In Progress", git checkout -b
gocli jira branch LPAD-26763 --type bug
gocli jira branch LPAD-26763 --type fea --to-status "In Progress"
gocli jira branch LPAD-26763 --type chg --dry-run   # show branch name only
gocli jira branch LPAD-26763 --type bug --no-transition
gocli jira branch LPAD-26763 --type fea -C C:\path\to\clone

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
---

## 🛠️ Build from Source
Requires Go 1.22+.
```bash
git clone https://github.com/yourorg/gocli
cd gocli
go build -o gocli .
```
