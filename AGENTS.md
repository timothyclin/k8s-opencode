# AGENTS.md — Coding Practices & Conventions

Guidelines for AI agents and human contributors working in this repository.

---

## Git Workflow

### Branch-First Development

Never commit directly to `main`. Always work on a feature branch.

```bash
git checkout -b feat/my-feature
# ... make changes ...
git add -A && git commit -m "feat: add my feature"
git push -u origin feat/my-feature
```

### Git Worktree

Use `git worktree` for parallel work streams — reviewing PRs, hotfixes, or running tests on another branch without stashing or switching context.

```bash
# Add a worktree for a feature branch
git worktree add ../k8s-omo-feat-xyz feat/xyz

# Add a worktree for reviewing a PR
git worktree add ../k8s-omo-pr-review pr-branch

# List active worktrees
git worktree list

# Remove when done
git worktree remove ../k8s-omo-feat-xyz
```

**When to use worktrees:**

- Reviewing a PR while your current branch has uncommitted work
- Running long tests on one branch while developing on another
- Hotfixing `main` without disrupting your feature branch
- Comparing behavior across branches side-by-side

**Worktree conventions:**

- Name worktree directories with a clear suffix: `<repo>-<purpose>`
- Clean up worktrees promptly after use (`git worktree prune`)
- Never nest worktrees inside the main repo directory

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add user authentication endpoint
fix: resolve helm chart default values override
docs: update architecture diagram
refactor: extract validation into shared utility
chore: bump dependency versions
```

### Atomic Commits

Each commit should represent one logical change. Don't mix unrelated changes in a single commit.

---

## Code Changes

### Minimal Diffs

- Change only what's necessary to accomplish the task
- Don't refactor unrelated code in the same PR
- Don't fix pre-existing lint/style issues unless that's the explicit goal

### Type Safety

- Never suppress type errors with `as any`, `@ts-ignore`, or `@ts-expect-error`
- Fix the root cause instead of silencing the compiler

### Error Handling

- No empty `catch` blocks — always handle or propagate errors
- Use structured error types when available
- Log with sufficient context for debugging

---

## Helm / Kubernetes Conventions

### Values Files

- Provide sensible defaults in `values.yaml`
- Document every value with inline comments
- Use `--set` overrides sparingly; prefer values files for complex configs

### Template Safety

- Always use `default` and `required` functions for critical values
- Quote strings in templates: `{{ .Values.name | quote }}`
- Test templates with `helm template` before applying

---

## Testing & Verification

### Before Marking Work Complete

1. Run `helm template` or `helm lint` on chart changes
2. Verify no regressions in existing functionality
3. Check that all new files are tracked by git

### Validation Commands

```bash
# Lint the Helm chart (local development)
helm lint chart/

# Render templates locally (local development)
helm template my-release chart/ -f chart/values.yaml

# Dry-run against a cluster (local development)
helm install my-release chart/ --dry-run --debug
```

### Deployment Commands

The chart is published to GHCR as an OCI artifact. Use these for actual deployments:

```bash
# Install from GHCR (production)
helm install ok8s oci://ghcr.io/timothyclin/k8s-omo/chart -n opencode --create-namespace \
  --version 0.1.0 -f values.yaml

# Upgrade from GHCR (production)
helm upgrade ok8s oci://ghcr.io/timothyclin/k8s-omo/chart -n opencode -f values.yaml
```

> **Local `./chart` paths** are for development and testing only — they require a clone of the repository. Published deployments should always use the OCI URI.

---

## File Organization

- Documentation goes in `docs/`
- Example configurations go in `examples/`
- Scripts go in `scripts/`
- Keep the repo root clean — only top-level config and docs belong here

---

## Agent Delegation & Failure Recovery

### Delegate First, Always

The primary agent (Sisyphus) should **maximize token efficiency by delegating work to specialized sub-agents**. Never do manually what a sub-agent can handle.

| Task Type | Delegate To |
|---|---|
| Codebase search, pattern discovery | `explore` agent |
| External docs, library lookups | `librarian` agent |
| Architecture decisions, debugging | `oracle` agent |
| Frontend / UI / styling | `visual-engineering` category |
| Trivial single-file edits | `quick` category |
| Complex logic, algorithms | `ultrabrain` category |

**Rules:**

- Fire multiple explore/librarian agents **in parallel** for non-trivial questions
- Always run explore/librarian in the **background** (`run_in_background=true`)
- After delegating a search, **do not repeat that same search manually** — wait for the agent's result
- Use `session_id` to continue with a sub-agent instead of starting fresh — preserves context and saves tokens

### When Sub-Agents Fail: Stop, Don't Take Over

If a delegated task fails, retry **once** with a corrected prompt or additional context via `session_id`. If it fails again:

1. **Stop.** Do not attempt a third retry.
2. **Do not silently take over the work yourself.** Taking over masks the root cause and wastes tokens on a potentially doomed approach.
3. **Inform the user immediately** with:
   - What task was delegated
   - Which agent was used
   - What failed (error, incorrect output, partial result)
   - How many attempts were made
4. **Triage together.** Let the user decide whether to:
   - Adjust the approach and retry
   - Escalate to a different agent (e.g., Oracle for debugging)
   - Handle it manually
   - Skip the task entirely

**Example failure report:**

```
⚠️ Delegation failed after 2 attempts.

Task: "Add responsive grid layout to dashboard"
Agent: visual-engineering (session: ses_abc123)
Attempt 1: Agent produced layout but ignored existing CSS variables
Attempt 2 (via session_id): Agent acknowledged CSS variables but broke the sidebar

I've stopped retrying. Options:
1. I can consult Oracle for a diagnosis
2. You can provide additional constraints and I'll retry
3. We skip this for now and move on
```

### Why This Matters

- **Token efficiency**: Sub-agents use cheaper, domain-optimized models. The primary agent's tokens are expensive — don't waste them on work a sub-agent should handle.
- **Failure visibility**: Silent takeover hides systemic issues (bad prompts, wrong agent choice, missing context). Surfacing failures lets us fix the root cause.
- **User agency**: The user should always be in the loop when the automated path breaks down.

---

## PR Etiquette

- One concern per PR
- Include a summary of **what** changed and **why**
- Link related issues
- Self-review your diff before requesting review
