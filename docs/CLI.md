# gocli — CLI syntax reference

Complete command-line reference for **gocli**. For install and onboarding, see [README.md](../README.md).

**Binary name:** `gocli`  
**Help:** `gocli --help`, `gocli <command> --help`

---

## Command tree

```text
gocli
├── version
├── auth
│   ├── login
│   ├── status
│   └── logout
├── jira
│   ├── list
│   ├── view <ISSUE-KEY>
│   ├── open <ISSUE-KEY>
│   ├── transition <ISSUE-KEY> <STATUS>
│   └── branch <ISSUE-KEY>
├── gh
│   ├── prs
│   ├── pr-view <number>
│   ├── pr-checkout <number>
│   ├── pr-create
│   └── issues
├── pr
│   ├── create
│   └── review <number>
├── deploy
│   ├── list
│   ├── run <pipeline>
│   └── validate
└── teams
    └── notify
        ├── dev-complete
        └── deployment
```

---

## Global options

| Flag | Short | Description |
|------|-------|-------------|
| `--verbose` | `-v` | Verbose output (persistent on all commands) |

**Exit codes:** `0` on success; `1` on error (message printed to stderr).

**Version:**

```bash
gocli version
```

Prints build version, commit, and date.

---

## Authentication — `gocli auth`

### `gocli auth login`

Interactive setup. Prompts for Jira URL/email/project, GitHub user/default repo, AI provider, and optional model. Tokens are stored in the **OS keyring** (not in `config.yaml`). Leave a token prompt blank to keep an existing keyring value.

**AI providers:** `github-models` (default) or `copilot`.

```bash
gocli auth login
```

### `gocli auth status`

Shows config path and a table of settings (tokens masked).

```bash
gocli auth status
```

### `gocli auth logout`

Removes Jira and GitHub tokens from the keyring.

```bash
gocli auth logout
```

---

## Jira — `gocli jira`

Requires Jira config (`jira.base_url`, `jira.email`) and `JIRA_API_TOKEN` (keyring or env).

### `gocli jira list`

List issues. Default JQL: assigned to you, status category not Done, ordered by updated.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--jql` | | *(built)* | Raw JQL; overrides `--mine`, `--status`, `--assignee` |
| `--mine` | | `true` | Only issues assigned to current user |
| `--assignee` | `-u` | | Assignee filter: Jira id/email, or alias from `jira.assignee_aliases` |
| `--status` | | | Filter by status name (e.g. `"In Review"`) |
| `--limit` | `-n` | `25` | Max issues returned |

**Assignee behavior:** If `--assignee` is set, `--mine` is ignored for JQL. Aliases are matched case-insensitively against keys in config.

```bash
gocli jira list
gocli jira list --assignee bob
gocli jira list -u teammate@corp.com
gocli jira list --status "In Progress"
gocli jira list --jql 'project = ENG AND status = "To Do" ORDER BY created DESC'
gocli jira list -n 50
```

### `gocli jira view <ISSUE-KEY>`

Print issue summary, status, people, URL, and description (plain text).

| Flag | Short | Description |
|------|-------|-------------|
| `--markdown` | `-m` | Output as markdown (for pasting into Cursor Agent, etc.) |

```bash
gocli jira view ENG-123
gocli jira view ENG-123 --markdown
```

### `gocli jira open <ISSUE-KEY>`

Open the issue in the default browser (`rundll32` on Windows, `open` on macOS, `xdg-open` on Linux).

```bash
gocli jira open ENG-123
```

### `gocli jira transition <ISSUE-KEY> <STATUS>`

Move the issue to a workflow status (name matched case-insensitively).

| Flag | Description |
|------|-------------|
| `--notify-devops` | After success, post to Teams DevOps webhook (requires config or `GOCLI_TEAMS_DEVOPS_WEBHOOK`) |

```bash
gocli jira transition ENG-123 "In Progress"
gocli jira transition ENG-123 Done --notify-devops
```

### `gocli jira branch <ISSUE-KEY>`

1. Fetches issue summary from Jira.
2. Builds branch name: `{type}-{KEY}-{slug}` where `slug` is the first 20 characters of the title, lowercased, normalized to hyphens.
3. Optionally transitions Jira (default status: **In Progress**).
4. Runs `git checkout -b <branch>` in the chosen repo.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--type` | | *(required)* | Prefix: `bug`, `fea`, or `chg` |
| `--to-status` | | `In Progress` | Jira status after branch creation |
| `--workdir` | `-C` | current dir | Git repo path (same idea as `git -C`) |
| `--allow-dirty` | | `false` | Allow uncommitted changes |
| `--no-transition` | | `false` | Only create/checkout branch; skip Jira |
| `--dry-run` | | `false` | Print planned actions and branch name only |

**Errors:** Dirty tree (without `--allow-dirty`), branch already exists, not a git repo.

```bash
gocli jira branch ENG-123 --type bug
gocli jira branch ENG-123 --type fea --to-status "In Progress"
gocli jira branch ENG-123 --type chg --dry-run
gocli jira branch ENG-123 --type bug --no-transition
gocli jira branch ENG-123 --type fea -C C:\work\my-app
gocli jira branch ENG-123 --type bug --allow-dirty
```

**Example branch name:** `bug-LPAD-26763-login-form-validation`

---

## GitHub — `gocli gh`

Requires GitHub config and `GITHUB_TOKEN`. Repo resolution: `--repo` / `-R`, else `origin` remote in cwd, else `github.default_repo` in config.

### `gocli gh prs`

List pull requests.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--repo` | `-R` | | `owner/name` |
| `--state` | | `open` | `open`, `closed`, or `all` |
| `--mine` | | `false` | Only PRs authored by configured GitHub user |
| `--limit` | `-n` | `30` | Max PRs |

```bash
gocli gh prs
gocli gh prs --mine --state open
gocli gh prs -R myorg/myrepo -n 10
```

### `gocli gh pr-view <number>`

Show PR metadata, mergeability, check runs, and body.

| Flag | Short | Description |
|------|-------|-------------|
| `--repo` | `-R` | `owner/name` |

```bash
gocli gh pr-view 482
gocli gh pr-view 482 -R myorg/myrepo
```

### `gocli gh pr-checkout <number>`

Runs external **`gh pr checkout <number>`** (GitHub CLI must be installed and on `PATH`).

```bash
gocli gh pr-checkout 482
```

### `gocli gh pr-create`

Create a PR from the **current git branch** (must already be pushed). **`--title` is required.**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--repo` | `-R` | | `owner/name` |
| `--title` | `-t` | *(required)* | PR title |
| `--body` | `-b` | | PR description |
| `--base` | | `main` | Base branch |
| `--draft` | | `false` | Create as draft |

```bash
gocli gh pr-create --title "Fix login" --base main
gocli gh pr-create -t "Add feature" -b "Details…" --base develop --draft
```

### `gocli gh issues`

List repository issues (excludes pull requests).

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--repo` | `-R` | | `owner/name` |
| `--state` | | `open` | `open`, `closed`, or `all` |
| `--limit` | `-n` | `30` | Max issues |

```bash
gocli gh issues
gocli gh issues --state all -R myorg/myrepo
```

---

## Pull requests — `gocli pr`

Preferred commands for opening PRs and AI review. Overlaps partially with `gocli gh` (see differences below).

### `gocli pr create`

Create a PR via GitHub API. Head branch must exist on GitHub (push first). For fork PRs use `--head forkowner:branch`.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--repo` | `-R` | | `owner/name` |
| `--title` | | last commit subject | PR title |
| `--body` | | | PR description (markdown) |
| `--body-file` | | | Read body from file (overrides `--body`) |
| `--base` | | repo default branch | Merge target |
| `--head` | | current branch | Source branch |
| `--draft` | | `false` | Open as draft |
| `--workdir` | `-C` | | Git clone for remote/branch resolution |

**Repo resolution order:** `--repo`, else `origin` in workdir (or cwd), else `github.default_repo`.

```bash
gocli pr create --title "Fix login"
gocli pr create --base main --head my-feature
gocli pr create -R myorg/therepo --workdir C:\other\clone --draft
gocli pr create --body-file pr-body.md
```

### `gocli pr review <number>`

AI code review using configured provider (`github-models` or `copilot`). Fetches PR diff from GitHub API.

| Flag | Short | Description |
|------|-------|-------------|
| `--repo` | `-R` | `owner/name` |

```bash
gocli pr review 482
gocli pr review 482 -R myorg/myrepo
```

### `pr create` vs `gh pr-create`

| | `gocli pr create` | `gocli gh pr-create` |
|---|-------------------|----------------------|
| Title | Optional (defaults to last commit) | **Required** (`-t`) |
| Base branch | Repo default on GitHub | Default `main` |
| Workdir | `--workdir` / `-C` | Current directory only |
| Body file | `--body-file` | `--body` only |

---

## Deploy — `gocli deploy`

Runs pipelines from a YAML spec (default file: `deploy.yml` in the current directory).

### Shared flag (all deploy subcommands)

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | `deploy.yml` | Path to deploy spec |

### `gocli deploy list`

List pipeline names, step counts, and descriptions.

```bash
gocli deploy list
gocli deploy list -f examples/deploy.yml
```

### `gocli deploy run <pipeline>`

Execute a named pipeline.

| Flag | Description |
|------|-------------|
| `--dry-run` | Print plan only; do not run steps |

On successful run (not dry-run), posts to Teams **deploy** webhook if configured.

```bash
gocli deploy run staging
gocli deploy run docker-release --dry-run
gocli deploy run vm-prod -f deploy.yml
```

### `gocli deploy validate`

Parse and validate `deploy.yml` without executing.

```bash
gocli deploy validate -f deploy.yml
```

### `deploy.yml` schema

```yaml
version: 1                    # required

env:                          # optional; merged into all pipelines
  KEY: value

pipelines:
  <name>:
    description: "…"          # optional
    env:                      # optional; pipeline-level env
      KEY: value
    steps:
      - name: "step label"    # required
        type: script | docker | k8s | ssh
        env: { … }            # optional; step-level env
        # … type-specific fields (see below)
```

#### Step type: `script`

| Field | Description |
|-------|-------------|
| `run` | Shell command (required) |
| `shell` | `bash`, `sh`, `pwsh`, `cmd` (default: `powershell` on Windows, `bash` elsewhere) |
| `workdir` | Working directory for the command |

#### Step type: `docker`

| Field | Description |
|-------|-------------|
| `image` | Image name/tag |
| `context` | Build context path |
| `dockerfile` | Dockerfile path |
| `push` | `true` to push after build |
| `args` | Extra args for `docker build` / `run` / `push` |

#### Step type: `k8s`

| Field | Description |
|-------|-------------|
| `action` | `apply`, `delete`, or `rollout` |
| `manifest` | Manifest file or directory |
| `namespace` | Kubernetes namespace (`-n`) |
| `kube_context` | `kubectl --context` (e.g. EKS) |
| `args` | Extra args (for `rollout`: e.g. `status`, `deployment/name`) |

#### Step type: `ssh`

| Field | Description |
|-------|-------------|
| `host` | Remote host (required) |
| `user` | SSH user |
| `port` | SSH port (default 22) |
| `identity_file` | Private key path |
| `run` | Script run on remote via `bash -s` on stdin |
| `args` | Extra `ssh` options before destination |

See [examples/deploy.yml](../examples/deploy.yml) for full examples (`ci`, `docker-release`, `staging`, `vm-prod`, `eks-prod`).

---

## Microsoft Teams — `gocli teams`

Incoming Webhook URLs in config or environment (env wins).

### `gocli teams notify dev-complete`

Post to DevOps / handoff channel.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--message` | `-m` | generic dev-complete text | Message body |

```bash
gocli teams notify dev-complete
gocli teams notify dev-complete -m "ENG-123 merged; ready for release prep."
```

### `gocli teams notify deployment`

Post to deployment-complete channel.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--message` | `-m` | generic deployment text | Message body |

```bash
gocli teams notify deployment
gocli teams notify deployment -m "Production deploy finished."
```

**Automatic Teams posts:**

- `jira transition … --notify-devops` → DevOps webhook
- Successful `deploy run` (not `--dry-run`) → deploy webhook

---

## Configuration file

**Path:** `~/.gocli/config.yaml` (macOS/Linux) or `%USERPROFILE%\.gocli\config.yaml` (Windows)

**Overrides:** `GOCLI_HOME` (directory), `GOCLI_CONFIG` (full file path)

**Example:**

```yaml
jira:
  base_url: https://yourcompany.atlassian.net
  email: you@company.com
  project: ENG
  sprint_field: auto          # auto | customfield_XXXXX | none
  assignee_aliases:
    bob: bob@company.com

github:
  user: your-github-login
  default_repo: yourorg/yourrepo

ai:
  provider: github-models     # github-models | copilot
  model: openai/gpt-4o-mini

teams:
  devops_webhook: "https://…"
  deploy_webhook: "https://…"
```

Secrets are **not** stored in this file. Use `gocli auth login` or environment variables.

---

## Environment variables

| Variable | Purpose |
|----------|---------|
| `JIRA_API_TOKEN` | Jira API token (fallback if keyring empty) |
| `GITHUB_TOKEN` | GitHub token (fallback if keyring empty) |
| `GOCLI_CONFIG` | Full path to config YAML |
| `GOCLI_HOME` | Config/cache directory (default `~/.gocli`) |
| `GOCLI_NO_DOTENV` | Set to `1` to disable `.env` loading |
| `GOCLI_JIRA_SPRINT_FIELD` | Sprint custom field id, or `none` / `-` to skip |
| `GOCLI_TEAMS_DEVOPS_WEBHOOK` | DevOps Teams webhook (overrides YAML) |
| `GOCLI_TEAMS_DEPLOY_WEBHOOK` | Deploy Teams webhook (overrides YAML) |
| `GOCLI_NO_HYPERLINK` | Disable OSC 8 hyperlinks in terminal tables |
| `GOCLI_GITHUB_REPO` | Used by install scripts only (`owner/repo`) |

### Project `.env`

At startup, gocli loads (if present, without overwriting existing OS env):

1. `./.env`
2. `./.env.local`
3. `<binary-dir>/.env`

Copy from `.env.example` for local development.

---

## Recommended workflow

```text
pick ticket  →  branch / PR  →  AI review  →  deploy  →  Teams notify  →  transition ticket
```

```bash
gocli jira list
gocli jira branch KAN-224 --type fea
gocli pr create --title "Implement feature X"
gocli pr review 482
gocli deploy run staging
gocli jira transition KAN-224 Done --notify-devops
```

---

## External dependencies

| Tool | Used by |
|------|---------|
| `git` | `jira branch`, `pr create`, repo detection |
| `gh` | `gh pr-checkout`, `copilot` AI provider |
| `docker` | `deploy` docker steps |
| `kubectl` | `deploy` k8s steps |
| `ssh` | `deploy` ssh steps |
| `aws` | Example EKS kubeconfig step in deploy.yml |

---

## Help discovery

```bash
gocli --help
gocli jira --help
gocli jira branch --help
gocli deploy run --help
```
