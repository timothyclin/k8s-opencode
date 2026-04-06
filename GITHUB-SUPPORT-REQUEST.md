# GitHub Support Request - Incorrect Contributor Attribution

## Summary

A user "Sisyphus-dev-ai" (GitHub account ID: 238992291) is appearing on the contributors page for my repository `timothyclin/k8s-opencode` with 1 commit attributed, but I cannot locate this commit in any accessible git history or API data.

## Investigation Performed

I conducted an exhaustive investigation including:

1. **Local git history** - `git log --all` shows only 3 authors:
   - Timothy Lin (tim@timlin.dev) - 161 commits
   - Timothy Lin (timothylin@interpres.net) - 28 commits
   - github-actions[bot] - 2 commits
   - No commits from "Sisyphus-dev-ai"

2. **GitHub API queries**:
   - `/repos/{owner}/{repo}/commits` - Returns 118 total commits, all from timothyclin or github-actions[bot]
   - `/repos/{owner}/{repo}/commits?author=sisyphus-dev-ai` - Returns empty array
   - `/repos/{owner}/{repo}/stats/contributors` - Shows sisyphus-dev-ai with 1 commit, 28 additions, 2 deletions
   - `/repos/{owner}/{repo}/pulls` - No PRs from sisyphus-dev-ai
   - `/repos/{owner}/{repo}/events` - No events from sisyphus-dev-ai

3. **Timeline data from stats API**:
   - Week of 2026-03-28: 28 additions, 2 deletions, 1 commit (authored)
   - Week of 2026-04-04: 0 activity

4. **Verified user exists**: sisyphus-dev-ai is a valid GitHub user (ID: 238992291)

## The Issue

- The contributor page shows sisyphus-dev-ai with 1 commit
- The commit is not accessible via any GitHub API or web interface
- No branch, PR, or event data shows this contribution
- Local git history contains no such commit

## Request

Please investigate and remove this incorrect attribution from my repository's contributor statistics. The commit appears to be phantom or from a deleted branch that should no longer be counted.

## Repository Details

- Repository: timothyclin/k8s-opencode
- Contributor showing incorrectly: sisyphus-dev-ai
- Expected commit count discrepancy: 1 ghost commit
