---
name: sdlc-pr-review
description: Structured code review for Bitbucket PRs using the sdlc-bridge MCP server. Use this skill whenever the user asks to review a PR, do code review, check a pull request, review recent PRs, or batch review PRs — even if they don't mention Bitbucket explicitly. Also trigger when the user says "review PR #123", "check the open PRs", "any PRs to review?", or similar.
---

# Structured PR Code Review

This skill guides thorough, structured code reviews for Bitbucket Cloud PRs using the sdlc-bridge MCP tools.

## Why This Exists

Ad-hoc reviews miss things. This skill enforces a consistent checklist so every review covers security, error handling, Go idioms, test coverage, and architecture — not just surface-level style.

## Available Tools

The sdlc-bridge MCP server provides these tools for PR review:

- **`read_pr_content`** — fetch single PR metadata + full unified diff
- **`review_open_prs`** — batch fetch all recent PRs with diffs (last N days)
- **`get_recent_prs`** — list open PRs without diffs (metadata only)
- **`submit_pr_review`** — post review comment on a PR

## Single PR Review

When reviewing one PR:

1. Fetch PR with `read_pr_content(workspace, repo_slug, pr_id)`
2. Analyze the diff against the checklist below
3. Write review in the structured format
4. Post with `submit_pr_review(workspace, repo_slug, pr_id, review_text)`

## Batch Review

When reviewing all recent PRs:

1. Fetch all with `review_open_prs(workspace, repo_slug, days=3)`
2. Review each PR individually against the checklist
3. Post individual reviews for each PR
4. Provide cross-cutting summary at the end

## Review Checklist

Work through each category. Skip categories that don't apply to the diff.

### Correctness
- Logic errors, off-by-one, nil dereference
- Race conditions in concurrent code
- Edge cases: empty inputs, zero values, max limits

### Security
- Injection risks (SQL, command, template)
- Auth/authz bypasses
- Secrets in code or logs
- Input validation at boundaries

### Go Idioms (for Go PRs)
- Error handling: wrapped errors with `%w`, no swallowed errors
- Naming: MixedCaps, receiver names, interface naming (-er suffix)
- Concurrency: proper channel/mutex usage, context propagation
- `defer` for cleanup, especially with `Close()`

### Error Handling
- All errors checked, not silently discarded
- Error messages include context (what operation failed, with what input)
- Typed errors where callers need to distinguish cases (like `bitbucket/client.go` does with `AuthError`, `NotFoundError`)

### Architecture
- Changes respect package boundaries (bitbucket/ has no MCP imports, tools/ bridges MCP to packages)
- No circular dependencies
- New code follows existing patterns in the codebase

### Tests
- New functionality has test coverage
- Edge cases tested
- Test names describe the scenario, not the function

### Performance
- No unnecessary allocations in hot paths
- Database/API calls not in loops without pagination
- Appropriate use of buffered I/O

## Review Output Format

Structure every review like this:

```markdown
## Summary
[1-2 sentences: what this PR does]

## Strengths
- [What was done well]

## Issues
[Ordered by severity: critical > major > minor]

### Critical
- **[file:line]** — [description of bug/security issue]

### Major
- **[file:line]** — [description]

### Minor
- **[file:line]** — [description]

## Suggestions
- [Optional improvements, not blocking]

## Verdict
[APPROVE | REQUEST_CHANGES | COMMENT] — [one-line rationale]
```

## Verdict Guidelines

- **APPROVE**: No critical/major issues. Minor issues can be noted but don't block.
- **REQUEST_CHANGES**: Any critical issue, or 2+ major issues, or a pattern of the same major issue.
- **COMMENT**: Questions that need answers before deciding, or significant design discussion needed.

## Cross-Cutting Summary (Batch Only)

After reviewing all PRs in batch mode, add:

```markdown
## Batch Summary
- **Reviewed:** N PRs
- **Approved:** N | **Changes requested:** N | **Comments:** N
- **Common patterns:** [issues seen across multiple PRs]
- **Recommendations:** [team-level suggestions based on patterns]
```
