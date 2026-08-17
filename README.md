# Stackboard

A local dashboard for GitHub pull request stacks. Requires Go and an
authenticated [GitHub CLI](https://cli.github.com/) session.

```bash
gh auth login
gh extension install bkach/gh-stackboard
gh stackboard --repo OWNER/REPO
```

Repeat `--repo` to include multiple repositories. Stackboard opens at
<http://127.0.0.1:4387> and refreshes every minute.

Use `--mock` for sample data or `--no-open` to prevent the browser opening.
