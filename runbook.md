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
