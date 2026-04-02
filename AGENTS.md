# AGENTS.md — Coding Practices & Conventions

Guidelines for AI agents and human contributors working in this repository.

---

## AI Installation (for agents)

Use the standalone install guide: **docs/ai-install.md**

```bash
# Quick install - latest version
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode --create-namespace -f values.yaml
```

---

## Git Workflow

### MANDATORY: Worktrees for ALL Edits

**Every code edit MUST use a worktree.** No exceptions.

**Why:** Multiple agent sessions may be working concurrently on this repository. Even a "trivial" single-file edit is a potential race condition with another session's work. Worktrees provide complete isolation.

```bash
# BEFORE making ANY code change:
cd /path/to/main/repo
git worktree list                                    # Check for other active sessions
git worktree add ../k8s-opencode-<task> -b agent/<task>  # Create isolated worktree
cd ../k8s-opencode-<task>                            # Work here, NOT in main repo

# Make changes, commit frequently
git add . && git commit -m "feat: description"

# BEFORE removing worktree: INTEGRATE the work
cd /path/to/main/repo
git checkout <target-branch>
git merge agent/<task>                               # Or cherry-pick

# ONLY THEN clean up
git worktree remove ../k8s-opencode-<task>
```

### Branch-First Development

Never commit directly to `main`. Always work on a feature branch (via worktree).

### Worktree Conventions

- **Directory naming**: `../<repo>-<short-task-description>`
- **Branch naming**: `agent/<short-task-description>`
- **Location**: Always OUTSIDE the main repo (use `../`)
- **Cleanup**: ALWAYS integrate work before removing worktree
- **Check first**: Run `git worktree list` before starting — another session may be active

### Integration Before Cleanup (CRITICAL)

Work in a worktree is NOT in the main codebase until merged. If you remove a worktree without integrating:
- Commits exist only in the branch
- Branch may be deleted
- **WORK IS EFFECTIVELY LOST**

```bash
# WRONG: Destroys work
git worktree remove ../k8s-opencode-task

# CORRECT: Integrate first
cd /path/to/main/repo
git checkout target-branch
git merge agent/task
git log --oneline -5  # Verify commits present
git worktree remove ../k8s-opencode-task
```

### Conflict Detection = STOP and Verify

**If you see ANY sign of concurrent work, STOP.** Do not dismiss it as "not my concern."

Signs of concurrent work:
- Stash conflicts or unexpected stash entries
- Merge conflicts when pulling/merging
- "Your branch has diverged" messages
- Uncommitted changes you didn't make

**Required response:**
1. Check `git worktree list` — are you in a worktree?
2. Check if other sessions are in worktrees
3. If BOTH in separate worktrees → safe to proceed
4. If EITHER in main repo → **STOP and coordinate with user**

"Not my concern" when seeing conflicts is **NEVER acceptable**. The other session's work IS your concern if you're about to overwrite it.

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

### Release Checklist (Chart or Image Updates)

Whenever a new chart version is tagged or Docker images are updated:

1. **Verify pipelines run cleanly** — check GitHub Actions workflow status
2. **Verify published artifacts are reachable** — confirm OCI chart and container images are publicly accessible
3. **Update README.md** — bump all version references, update install commands, ensure documentation matches the release
4. **Update docs/** — any affected documentation files (architecture, customization, maintenance) must reflect the changes

### Version Synchronization (CRITICAL)

**This is an automated process handled by CI, not manual edits.**

When a tag like `v0.1.4` is pushed, the CI workflows MUST automatically synchronize version numbers:

| File | What to Update | Example |
|------|----------------|---------|
| `chart/Chart.yaml` | `version` and `appVersion` | `version: 0.1.4`, `appVersion: "0.1.4"` |
| `chart/values.yaml` | `image.tag` | `tag: "0.1.4"` |
| `chart/values.yaml` | `auth.router.image.tag` | `tag: "0.1.4"` |

**Why this matters:**
- Chart version (Chart.yaml `version`) must match the tag, otherwise OCI push fails
- Image tags must be consistent, otherwise helm install pulls wrong image version
- Version mismatch causes "chart not found" and "image not found" errors

**CI Workflow Implementation:**

The publish workflows (`publish-chart.yml` and `publish-images.yml`) MUST:
1. Extract version from git tag: `VERSION=${GITHUB_REF#refs/tags/v}`
2. Update Chart.yaml: `yq -i ".version = \"$VERSION\"" chart/Chart.yaml`
3. Update values.yaml: `yq -i ".image.tag = \"$VERSION\"" chart/values.yaml`
4. Update auth-router tag: `yq -i ".auth.router.image.tag = \"$VERSION\"" chart/values.yaml`
5. Commit the version changes back to the branch
6. Then package and push

**Common failure patterns (DO NOT do):**
- ❌ Manually editing Chart.yaml/values.yaml before tagging
- ❌ Tagging without running CI (e.g., local tag push)
- ❌ Pushing tags but not letting CI update versions
- ❌ Forgetting to update auth.router.image.tag

**Correct workflow:**
```bash
# Just push a tag - CI handles everything else
git tag v0.1.5
git push origin v0.1.5
# CI will:
#   1. Update chart/Chart.yaml version to 0.1.5
#   2. Update chart/values.yaml image tags to 0.1.5
#   3. Commit those changes
#   4. Build and push images to ghcr.io/timothyclin/k8s-opencode/*:0.1.5
#   5. Build and push chart to oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s:0.1.5
```

### Deployment Commands

The chart is published to GHCR as an OCI artifact. Use these for actual deployments:

```bash
# Install latest version (recommended - omit --version to use latest)
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode --create-namespace \
  -f values.yaml

# Specific version (only if you need exact version - check GitHub releases)
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode --create-namespace \
  --version <version> -f values.yaml

# Upgrade
helm upgrade ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode -f values.yaml
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
