---
name: DelegationRecovery
description: Enforces proper delegation failure recovery with user pause for repeated failures. AUTO-LOADS at session start. USE WHEN subagent fails, delegation fails, files missing after task completion, or verification fails.
---

# DelegationRecovery - Subagent Failure Recovery Protocol

**Auto-loads at session start.** This skill enforces proper recovery when delegated subagents fail to deliver.

---

## CRITICAL: Why This Exists

Sisyphus's default behavior correctly delegates work to subagents, but when subagents report completion without delivering (missing files, broken tests), the fallback is direct implementation. This skill provides explicit recovery rules and a pause mechanism for repeated failures.

---

## Subagent Failure Recovery (MANDATORY)

### Step 1: Detect Failure

A subagent has failed when:
- Task reports completion but files don't exist
- Expected deliverables are missing
- Tests fail after subagent reports success
- Files exist but in wrong location

### Step 2: Re-Delegate (FIRST ATTEMPT)

**ALWAYS re-delegate with `session_id` before direct implementation:**

```typescript
// CORRECT: Continue the same session with fix instruction
task(
  session_id="ses_xxx",  // SAME session - preserves context
  category="deep",
  load_skills=[],
  prompt="Fix: The files were not created in the correct location.
- Previous attempt created files in worktree instead of main repo
- REQUIRED: Create files in [correct path]
- Verify with: pnpm test after creation"
)
```

**Re-delegation prompt MUST include:**
1. What went wrong (specific error)
2. Correct location/path (exact)
3. Verification step (test command, file check)

### Step 3: Second Failure = PAUSE FOR USER

**If re-delegation fails a SECOND time:**

```
🚨 DELEGATION FAILURE DETECTED

Task: [task name]
Subagent Session: [session_id]
Attempts: 2

What I tried:
1. [First delegation - what happened]
2. [Second delegation - what happened]

What's still broken:
- [Specific issue]

⏸️ PAUSING FOR USER DIRECTION

Options:
1. Debug with me - I'll investigate the root cause
2. Different approach - suggest an alternative strategy
3. Direct implementation - authorize me to implement directly (not recommended)

How would you like to proceed?
```

### Step 4: Direct Implementation (ONLY AFTER USER AUTHORIZATION)

Direct implementation is acceptable ONLY when:

| Condition | Authorization Required |
|-----------|----------------------|
| Re-delegation failed twice | User must explicitly authorize |
| Single-line fix (typo, missing import) | No authorization needed |
| Environment commands (git status, pnpm test) | No authorization needed |
| Git operations (commit, push) | No authorization needed |
| User explicitly requests direct work | User authorization = request itself |

---

## Verification After Every Delegation

After any `task()` completes, verify before proceeding:

```typescript
// Step 1: Check files exist
const files = glob("expected/path/**/*")

// Step 2: If missing, re-delegate
if (files.length === 0) {
  task(session_id="ses_xxx", prompt="Fix: Files not created...")
}

// Step 3: Run tests
bash({ command: "pnpm test" })

// Step 4: If tests fail, re-delegate
task(session_id="ses_xxx", prompt="Fix: Tests failing in [file]...")
```

---

## Failure Tracking

Track failures per session:

```
Session: ses_xxx
Task: Create classifier.ts
Attempt 1: Files not created
Attempt 2: Files in wrong location
Status: PAUSED - awaiting user direction
```

---

## Anti-Patterns (FORBIDDEN)

| Action | Why Forbidden |
|--------|---------------|
| Direct `write` after subagent fails | Bypasses delegation |
| Direct `edit` to fix subagent's code | Should re-delegate fix |
| Skipping verification | Leads to broken state |
| Third re-delegation without pause | Wastes tokens, need user input |
| Implementing after 1 failure | Should always try re-delegation first |

---

## Workflow Routing

| Workflow | Trigger | File |
|----------|---------|------|
| **RecoverFailedTask** | subagent completes but deliverables missing | `Workflows/RecoverFailedTask.md` |
| **PauseForUser** | repeated delegation failures | `Workflows/PauseForUser.md` |

---

## Examples

**Example 1: Files not created**
```
Subagent task_abc reports completion
→ glob("packages/agent/src/classifier.ts") returns empty
→ Re-delegate: task(session_id="ses_xxx", prompt="Fix: classifier.ts not created...")
→ Second failure
→ PAUSE: Show user what failed, ask for direction
```

**Example 2: Wrong location**
```
Subagent creates files in worktree instead of main repo
→ glob finds files in .worktrees/phase2/
→ Re-delegate: task(session_id="ses_xxx", prompt="Fix: Files created in wrong location...")
→ If fixes: Continue
→ If fails: PAUSE
```

**Example 3: Tests failing**
```
Subagent creates files but tests fail
→ Re-delegate: task(session_id="ses_xxx", prompt="Fix: Tests failing in classifier.test.ts...")
→ If fixes: Continue
→ If fails: PAUSE
```

---

## Quick Reference

**Re-delegation pattern:**
```
task(session_id="ses_xxx", prompt="Fix: [specific issue]. [correct location]. [verification step]")
```

**Pause pattern:**
```
🚨 DELEGATION FAILURE DETECTED
⏸️ PAUSING FOR USER DIRECTION
[Show options]
```