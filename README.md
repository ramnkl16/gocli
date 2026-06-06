# gocli Setup Guide

This guide explains how to build `gocli`, make it runnable from any terminal,
and configure Jira, GitHub, and Microsoft Teams access.

## 1. Build The CLI

From the repository root:

```powershell
go build -o gocli.exe .
```

If you run from the same folder as the executable, use:

```powershell
.\gocli.exe auth status
```

If the executable is on your `PATH`, you can use:

```powershell
gocli auth status
```

## 2. Add gocli To Windows Path

Add the folder that contains `gocli.exe` to your Windows `Path`.

For this local checkout, the folder is usually:

```text
C:\Yourpath\gocli
```

Steps:

1. Open Windows **Settings**.
2. Go to **System** -> **About**.
3. Click **Advanced system settings**.
4. Click **Environment Variables**.
5. Under **User variables**, select **Path** and click **Edit**.
6. Click **New**.
7. Paste the folder path that contains `gocli.exe`.
8. Click **OK** on all dialogs.
9. Close and reopen PowerShell.


## 3. Create Jira API Token

Jira commands require a Jira API token.

1. Open [https://id.atlassian.com](https://id.atlassian.com).
2. Sign in with your Atlassian/Jira account.
3. Open **API tokens**:
   [https://id.atlassian.com/manage-profile/security/api-tokens](https://id.atlassian.com/manage-profile/security/api-tokens).
4. Click **Create API token**.
5. Use a label like `gocli`.
6. Copy the token immediately.

You will also need:

- Jira base URL, for example `https://yourcompany.atlassian.net`
- Jira email address
- Default project key, optional, for example `ENG`

## 4. Create GitHub Token

GitHub commands and the default `github-models` AI provider require a GitHub
token.

1. Open [https://github.com/settings/tokens](https://github.com/settings/tokens).
2. Click **Generate new token**.
3. Choose **Generate new token (classic)** unless your organization requires a
   fine-grained token.
4. Enable at least:
   - `repo`
   - `read:org`
5. Click **Generate token**.
6. Copy the token immediately.

For a fine-grained token, create it under GitHub **Settings** -> **Developer
settings** and give it access to the repositories you want to use with `gocli`.

## 5. Create Teams Webhook

Teams messages use a webhook URL.

For Teams Workflows:

1. Open Microsoft Teams.
2. Go to the Team/channel where messages should appear.
3. Open **Workflows**.
4. Search for `webhook`.
5. Select **Send webhook alerts to a channel**.
6. Configure it for the target channel.
7. Create the workflow.
8. Click **Copy webhook link**.

For older Teams tenants, you may also see **Connectors** -> **Incoming Webhook**.
That can also provide a webhook URL, but Microsoft is retiring Office 365
Connectors in many tenants.

## 6. Configure gocli

Run:

```powershell
gocli auth login
```

If `gocli` is not on your `PATH`, run:

```powershell
.\gocli.exe auth login
```

The login command asks for:

- Jira base URL
- Jira email
- Jira API token
- GitHub username
- GitHub default repo, optional, like `owner/repo`
- GitHub token
- AI provider and model, optional

Check the result:

Verify:

```powershell
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
 
``````

## 7. Configure Teams Webhook

You can set Teams webhook URLs in `.env`:

```env
GOCLI_TEAMS_DEVOPS_WEBHOOK=https://your-workflow-webhook-url
GOCLI_TEAMS_DEPLOY_WEBHOOK=https://your-workflow-webhook-url
```

Or set them for the current PowerShell session:

```powershell
$env:GOCLI_TEAMS_DEVOPS_WEBHOOK = "https://your-workflow-webhook-url"
$env:GOCLI_TEAMS_DEPLOY_WEBHOOK = "https://your-workflow-webhook-url"
```

Or add them to `%USERPROFILE%\.gocli\config.yaml`:

```yaml
teams:
  devops_webhook: "https://your-workflow-webhook-url"
  deploy_webhook: "https://your-workflow-webhook-url"
```

Test Teams:

```powershell
gocli teams notify dev-complete -m "Test message from gocli"
```

## 8. Local .env

`gocli` automatically loads `.env` and `.env.local` from the current directory.
Real OS environment variables override `.env` values.

Create a local `.env` from the example:

```powershell
Copy-Item .env.example .env
```

Do not commit real tokens or webhook URLs.


# To know about the command and how to use gocli
````FOLLOW OUR runbook.md````