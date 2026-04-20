---
name: sdlc-rfc-to-pr
description: Orchestrate the full SDLC workflow from Confluence RFC to Bitbucket PR, with structured RFC analysis, repo scanning, comprehensive task breakdown, human review gate, and Jira deduplication. Use this skill when the user wants to implement a feature from an RFC, start an SDLC workflow, pick up a Jira ticket with an RFC, go from spec to PR, or mentions "implement from RFC/confluence". Also trigger when the user provides a confluence URL + jira ticket and expects implementation work.
---

# RFC-to-PR Workflow

This skill orchestrates the complete development lifecycle as a phased pipeline: analyze RFC with repo scanning, break down into atomic tasks, human review gate, deduplicate and publish to Jira, then branch → implement → verify → PR. Uses three MCP servers working together.

## MCP Servers Involved

| Server | Handles |
|--------|---------|
| **sdlc-bridge** (this project) | Branch setup (from v_next), Go verification, PR submission (targeting v_next) |
| **Atlassian MCP** | Reading Confluence RFCs + page comments |
| **Jira MCP** | Dedup via JQL, creating subtasks under epic/parent, logging progress |

## Workflow Phases

Execute each phase with fresh context. Re-read data at each boundary rather than relying on memory — this prevents state drift in long workflows.

### Phase 1: RFC Analysis + Repository Scanning

#### 1a. Fetch the RFC
Use Atlassian MCP to fetch the Confluence RFC page. Also fetch:
- **Page comments** (inline + footer) — decisions from reviewers often override or refine the RFC body
- **Linked pages** — parent PRD, sibling design docs for additional context

#### 1b. Scan the Repository
Search the codebase for RFC keywords: entity names, endpoint paths, table names, service names. Use Grep/Glob to identify affected packages and files. Read relevant source files to understand existing patterns and interfaces.

Be targeted — do NOT read the entire codebase.

#### 1c. Produce Structured Analysis
Present analysis in this format:
- **Summary**: 3-5 sentences — what and why
- **Functional Requirements**: Numbered (FR-1, FR-2...) with Given/When/Then acceptance criteria
- **Non-Functional Requirements**: Performance, availability, observability
- **Affected Files**: Concrete paths from repo scan (not guesses)
- **Out of Scope**: What the RFC explicitly excludes
- **Open Questions**: Anything ambiguous

**If open questions exist → STOP.** Present questions to user and wait for resolution before Phase 2. Do NOT proceed with unresolved ambiguity — bad requirements compound through every later phase.

### Phase 2: Comprehensive Task Breakdown

Re-read Phase 1 analysis. Break work into atomic, ordered tasks.

The workflow takes two Jira inputs:
- **`jira_epic`** — epic or parent ticket to create subtasks under
- **`jira_ticket`** — specific work item ticket for this implementation
- **`project_key`** — (optional) Jira project key for dedup; derived from epic prefix if omitted

#### Task Format
Number tasks T01, T02... in execution order (dependencies first). Each task:

```
### T{nn} — {Concise title}
- Type: feature | refactor | bugfix | infra | test-only
- Complexity: S | M | L  (S<100 LOC, M=100-300, L=300-500)
- Affected files: [concrete paths]
- Depends on: T{mm} or none
- Jira labels: sdlc, [domain labels]

Acceptance criteria:
- Given [state], when [action], then [outcome].

Test cases:
1. happy_path: [description]
2. edge_case: [description]
3. error_handling: [description]
```

#### Atomicity Rules
- Each task < 500 LOC (production + test combined)
- Each task = one PR
- Minimum 1 acceptance criterion + 3 test cases per task
- No circular dependencies
- Tasks must be testable in isolation

#### Summary Table
Present tasks as a summary table after the detailed breakdown.

### HUMAN REVIEW GATE

**MANDATORY STOP.** Present the complete task breakdown to the user and wait for explicit approval ("approved", "proceed", "publish").

The user may:
- **Approve** as-is
- **Request changes** — revise and re-present
- **Cancel** the workflow

NEVER create Jira issues without explicit approval. The breakdown review is the value of this pipeline.

### Phase 2b: Jira Issue Creation with Deduplication

After user approval, create Jira issues with dedup:

1. **Dedup first**: For each task, search Jira via JQL: `project = {project_key} AND labels = sdlc AND summary ~ "[{jira_epic}] T{nn}"`. If match exists, reuse it.
2. **Create new issues** only when no match found. Issue type "Task", under epic, linked to jira_ticket. Labels from breakdown + always "sdlc".
3. **Report**: Table of task / jira key / created or reused.

This makes the workflow idempotent — safe to re-run without creating duplicates.

### Phase 3: Set Up Branch

Use `setup_bitbucket_branch` with:
- `workspace` and `repo_slug` from user
- `branch_name` — typically matches Jira ticket key (e.g., `PROJ-123-feature-name`)
- `from_branch` — **always `v_next`**, not main

Branches must be created from `v_next`. This is the integration branch for all feature work.

### Phase 4: Implement

Re-read RFC requirements, task breakdown from Phase 2, and Jira issues from Phase 2b. Work through tasks in dependency order (T01, T02...).

Follow existing code patterns:
- Check how similar features are implemented in the codebase
- Respect package boundaries (see `go-mcp-tool` skill if adding MCP tools)
- Match naming conventions, error handling style, test patterns

Log progress to Jira via Jira MCP — add comments to the parent ticket as you complete each task.

### Phase 5: Verify

Run `run_go_verification` to execute:
1. `go fmt ./...`
2. `go vet ./...`
3. `go test ./...`

Stops on first failure. Read error output carefully. Fix and re-run until all green. Do NOT proceed to Phase 6 with any failures.

### Phase 6: Submit PR

Use `submit_bitbucket_pr` with:
- `target_branch` — **always `v_next`**, not main
- Clear title: `[PROJ-123] Brief description of what was built`
- Description includes: RFC link, Jira epic + ticket links, summary of changes, testing notes

## Handling Common Issues

### RFC is ambiguous
Phase 1 catches this via Open Questions. STOP and present to user.

### Verification loop won't converge
If you've fixed the same test 3+ times, stop. Present error to user — deeper design issue or flaky test.

### Branch already exists
`setup_bitbucket_branch` handles this — detects existing branches, just checks out locally.

### Push fails
Usually means remote has commits not in local. Run `git pull --rebase origin <branch>` before retrying.

### Jira duplicates
Phase 2b dedup handles this. JQL search before each create. Safe to re-run.

## When NOT to Use This Workflow

- **Hotfix/bugfix**: Skip Phase 1-2b. Just branch → fix → verify → PR.
- **Documentation only**: No verification needed.
- **Refactoring with no RFC**: Skip Phase 1, use existing Jira context.

For these cases, use the relevant subset of phases.
