# gocli

> One CLI for Jira, GitHub, Copilot PR review, deployments, and Teams alerts.

`gocli` is a single static Go binary that gives developers a unified
"cockpit" for the work they normally do across many tabs and tools.

---

**If that one-liner is blocked** (policy, proxy, etc.): open
[Releases](https://github.com/ramnkl16/gocli/releases/latest), download
`gocli_Windows_amd64.zip` (most PCs) or `gocli_Windows_arm64.zip` (ARM),
extract `gocli.exe`, place it in a folder that is already on your `PATH`, or
add that folder under **Settings → System → About → Advanced system settings
→ Environment Variables → Path** (your user).

#### Linux

Use the same `install.sh` flow as macOS. The script supports **macOS** and
**Linux** only. From
[Releases](https://github.com/ramnkl16/gocli/releases/latest), you can instead
download `gocli_Linux_amd64.tar.gz` or `gocli_Linux_arm64.tar.gz`, extract the
`gocli` binary into e.g. `~/.local/bin`, and ensure that directory is on your
`PATH`.

### Step 2 — Create API tokens (before `auth login`)

Have these ready; you will paste them when `gocli auth login` asks.

#### Jira (Atlassian) API token

1. In a browser, sign in to your Atlassian account:
   [https://id.atlassian.com](https://id.atlassian.com).
2. Open **API tokens**:
   [https://id.atlassian.com/manage-profile/security/api-tokens](https://id.atlassian.com/manage-profile/security/api-tokens).
3. Click **Create API token**, choose a label (e.g. `gocli`), and confirm.
4. **Copy the token immediately** and store it somewhere safe; you cannot view
   it again later.
5. Also note for the next step:
   - **Jira base URL** — e.g. `https://yourcompany.atlassian.net` (same host
     you see in the browser on an issue).
   - **Email** — the Atlassian / Jira account email.
   - **Default project key** (optional) — e.g. `ENG` from tickets like `ENG-123`.

#### GitHub personal access token

1. Sign in to GitHub and open token settings:
   [https://github.com/settings/tokens](https://github.com/settings/tokens).
2. Click **Generate new token** and choose **Generate new token (classic)**.
3. Set a **Note** (e.g. `gocli`), **Expiration**, and enable at least:
   - **`repo`** — use private repos and PR workflows.
   - **`read:org`** — read org and team membership.
4. Click **Generate token**, then **copy** the token (classic tokens often
   start with `ghp_`).

Your organization may require a **fine-grained** token instead; create one under
GitHub → **Settings → Developer settings** with access to the repositories you
need and permissions consistent with `repo` + org read for those resources.

### Step 3 — Sign in and verify

```bash
gocli auth login
gocli auth status
```

Use the same commands in **PowerShell** on Windows after install.

`auth login` asks for:

- **Jira** — base URL, email, optional default project key, and the **Jira API
  token** from step 2. The **Sprint** column uses your Jira site’s **Sprint**
  custom field; the field id is **discovered automatically** (first custom
  field named “Sprint”). Override in `~/.gocli/config.yaml` (or
  `%USERPROFILE%\.gocli\config.yaml` on Windows) as `jira.sprint_field:
  customfield_XXXXX`, or `GOCLI_JIRA_SPRINT_FIELD`. Set `jira.sprint_field:
  auto` or leave unset for discovery; use `none` in the env to skip requesting
  the field.
- **GitHub** — username, optional default repo `owner/name`, and the **GitHub
  token** from step 2 (`repo`, `read:org` for classic tokens).
- **AI provider** — `github-models` (default; uses your GitHub token with
  [models.github.ai](https://models.github.ai)) or `copilot` (shells out to the
  `gh copilot` extension).

Config path: `~/.gocli/config.yaml` on macOS and Linux,
`%USERPROFILE%\.gocli\config.yaml` on Windows (override with `GOCLI_CONFIG`).
Secrets are stored in the keyring, not as plain text in the config file.

**Microsoft Teams** uses [Incoming Webhooks](https://learn.microsoft.com/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook)
(Workflow or legacy connector). Optional YAML (or use env in CI instead of committing URLs):

```yaml
teams:
  devops_webhook: "https://…"   # DevOps / handoff channel
  deploy_webhook: "https://…"   # deployment-complete channel
```

Environment overrides: `GOCLI_TEAMS_DEVOPS_WEBHOOK`, `GOCLI_TEAMS_DEPLOY_WEBHOOK`
(each wins over the matching config field).

The optional `gh` CLI is used as a fallback for `gh pr-checkout` and
`gh copilot` shell-outs.

### Project-local environment with `.env`

For per-project setup (or for CI / containers where there is no OS keyring),
`gocli` automatically loads `.env` and `.env.local` from the current directory
at startup. Real OS environment variables always win over `.env` values.

```bash
cp .env.example .env       # macOS / Linux
Copy-Item .env.example .env # PowerShell
# edit .env and fill in JIRA_API_TOKEN / GITHUB_TOKEN
gocli auth status
```

Both `.env` and `.env.local` are gitignored (only `.env.example` is committed).
Set `GOCLI_NO_DOTENV=1` in your shell to disable `.env` loading entirely.

## Install from source (Go)

Requires Go 1.22+.

```bash
go install github.com/ramnkl16/gocli@latest
```
## 🚀 Quick Start

### 1. Install

Published on [GitHub Releases](https://github.com/yourorg/gocli/releases). Asset names are stable (no version in the filename), so you can use **latest** URLs or the scripts below.

Replace `yourorg/gocli` with your real `owner/repo`, or set `GOCLI_GITHUB_REPO` when using the scripts.

```bash
git clone https://github.com/ramnkl16/gocli
cd gocli
go build -o gocli .
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

# Teams (configure webhooks in config or GOCLI_TEAMS_* env — see Microsoft Teams below)
gocli teams notify dev-complete
gocli teams notify dev-complete -m "ENG-123 merged; ready for release prep."
gocli teams notify deployment -m "Production deploy finished."
gocli jira transition ENG-123 Done --notify-devops   # transition + DevOps channel post
```

## Microsoft Teams

Notify a **DevOps** channel when development is handed off, and a **deployment**
channel when a pipeline finishes (or send messages manually).

### 1. Create webhooks in Teams

In the target channel, add an **Incoming Webhook** (or **Workflow** POST URL from
Power Automate). You will get a long `https://…` URL per channel.

### 2. Configure gocli

Either add URLs to `~/.gocli/config.yaml` / `%USERPROFILE%\.gocli\config.yaml`:

```yaml
teams:
  devops_webhook: "https://outlook.office.com/webhook/…"   # handoff / DevOps
  deploy_webhook: "https://outlook.office.com/webhook/…"   # deploy completed
```

…or set environment variables (they override the YAML values — good for CI and
for keeping URLs out of files):

| Variable | Use |
| -------- | --- |
| `GOCLI_TEAMS_DEVOPS_WEBHOOK` | `teams notify dev-complete`, `jira transition … --notify-devops` |
| `GOCLI_TEAMS_DEPLOY_WEBHOOK` | `teams notify deployment`, auto-notify after successful `deploy run` |

You can also put those env vars in a project **`.env`** (see above); OS env wins.

### 3. Commands

| Action | Command |
| ------ | ------- |
| Post a generic “dev complete” message to the DevOps webhook | `gocli teams notify dev-complete` |
| Custom message | `gocli teams notify dev-complete -m "…"` |
| Post to the deployment channel | `gocli teams notify deployment` / `-m "…"` |
| After Jira transition, also notify DevOps | `gocli jira transition KEY "Status" --notify-devops` |
| After **`deploy run`** succeeds (not `--dry-run`) | Automatically posts if `deploy_webhook` / `GOCLI_TEAMS_DEPLOY_WEBHOOK` is set |

Messages are sent as plain text (`{"text":"…"}`) for standard Incoming Webhooks.

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
the slowest part of shipping a small change. This collapses the loop (including
optional Teams posts when a deploy finishes or work is handed to DevOps):

```text
pick ticket  →  branch / PR  →  AI review  →  deploy  →  Teams notify  →  transition ticket
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
