---
name: GitWorkflow
description: Enforces safe git branching workflow. USE WHEN making code changes, editing files, or committing. MANDATORY worktrees for ALL editing tasks — multiple agent sessions may be concurrent.
---

# GitWorkflow

Enforces the project's git branching protocol before any code changes.

## Core Principle: Worktrees Are MANDATORY

**Every editing task MUST use a worktree.** No exceptions.

**Why:** At any given time, multiple agent sessions may be working concurrently on the same repository. Even a "simple" single-file edit is a potential race condition with another session's work.

- Worktrees provide complete isolation
- Each session works in its own directory
- No risk of one session's uncommitted changes conflicting with another's
- Integration is explicit and deliberate

## Workflow Routing

| Workflow | Trigger | File |
|----------|---------|------|
| **BeforeChanges** | Before editing any code file | `Workflows/BeforeChanges.md` |
| **Commit** | Before committing changes | `Workflows/Commit.md` |
| **Cleanup** | After merge/PR completion | `Workflows/Cleanup.md` |

## Rules

### MANDATORY Checks (STOP if violated)

1. **NEVER** commit directly to `main` or `master`
2. **NEVER** edit files in the main repository directory
3. **ALWAYS** create a worktree before editing ANY file
4. **ALWAYS** integrate work before removing worktree
5. **ALWAYS** run tests, lint, build before committing

### Worktree Workflow (MANDATORY for ALL edits)

```bash
# 1. Create worktree with new branch
git worktree add ../<repo>-<task-name> -b agent/<task-name>

# 2. Work in the worktree
cd ../<repo>-<task-name>

# 3. Make changes, commit frequently
git add . && git commit -m "feat: description"

# 4. When done: integrate to target branch
cd /path/to/main/repo
git checkout <target-branch>
git merge agent/<task-name>

# 5. Only then: cleanup
git worktree remove ../<repo>-<task-name>
```

### NO Fallback to Single-Directory Branching

~~Branch fallback is NOT allowed.~~ Even for "trivial" tasks:

- Another session might be working in the main repo
- Your branch switch could interrupt their work
- Uncommitted changes from either session could be lost

**Every edit = worktree. No exceptions.**

## Pre-Work Checklist

Before starting ANY code changes:

```bash
# 1. Check for existing worktrees
git worktree list

# 2. Check recent agent branches (may indicate other sessions)
git branch -a --sort=-committerdate | grep -E "agent/|feat/" | head -10

# 3. Confirm you're about to create a NEW worktree
```

| Finding | Action |
|---------|--------|
| Existing worktrees | **WARN** user — another session may be active |
| In main repo directory | **STOP** — must create worktree first |
| Already in a worktree | **PROCEED** if it's YOUR worktree for THIS task |

## Cross-Session Awareness

When starting work:

```bash
git worktree list
```

If you see active worktrees:
1. **WARN** user: "I see worktree 'X' — another session may be working"
2. **ASK**: "Should I create a separate worktree, or coordinate with that work?"
3. **NEVER** assume the other worktree is abandoned

## Conflict Detection = STOP (CRITICAL)

**If you see ANY sign of concurrent work, STOP and verify.** Signs include:

- Stash conflicts or stash entries you didn't create
- Merge conflicts when pulling/merging
- "Your branch has diverged" messages
- Uncommitted changes you didn't make
- Recent commits from another session/agent

**WRONG response:**
```
"The stash conflict is from the concurrent session's work — not my concern."
```

**CORRECT response:**
```
⚠️ I detected signs of concurrent work:
- [describe what you saw: stash conflict, merge conflict, etc.]

Before proceeding, I need to verify:
1. Am I in a worktree? (I should be)
2. Is the other session also in a worktree?
3. Are we working on conflicting changes?

[Run git worktree list, git status, git stash list]

If BOTH sessions are in separate worktrees → safe to proceed independently.
If EITHER session is in main repo → STOP and coordinate with user.
```

**Why this matters:**
- "Not my concern" attitude is how work gets lost
- Conflicts are SIGNALS that isolation may have failed
- Taking 30 seconds to verify saves hours of lost work
- The other session's work IS your concern if you're about to overwrite it

## Integration Before Cleanup (CRITICAL)

**NEVER** remove a worktree without integrating the work:

```bash
# WRONG: Destroys work
git worktree remove ../<repo>-<task>

# CORRECT: Integrate first
cd /path/to/main/repo
git checkout <target-branch>
git merge agent/<task-name>
# Verify commits are present
git log --oneline -5
# THEN cleanup
git worktree remove ../<repo>-<task>
```

## Examples

**Example 1: Any edit (even single file)**
```
User: "Fix typo in README"
→ MUST use worktree (another session might be working)
→ git worktree add ../myrepo-fix-typo -b agent/fix-typo
→ cd ../myrepo-fix-typo
→ Make edit, commit
→ Merge to target branch
→ Cleanup worktree
```

**Example 2: Multi-file task**
```
User: "Implement auth flow"
→ git worktree add ../myrepo-auth -b agent/auth
→ cd ../myrepo-auth
→ Implement across multiple files
→ Commit each logical unit
→ Merge to target branch
→ Cleanup worktree
```

**Example 3: Detecting parallel work**
```
→ git worktree list shows: ../myrepo-operator-work
→ "I see worktree 'myrepo-operator-work'. Another session may be active."
→ "I'll create a separate worktree for my task to avoid conflicts."
→ git worktree add ../myrepo-my-task -b agent/my-task
```

## Integration

This skill should be loaded at session start. Add to your skill configuration:

```json
{
  "auto_load": ["GitWorkflow"]
}
```
