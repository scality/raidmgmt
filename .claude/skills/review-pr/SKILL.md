---
name: review-pr
description: Review a PR on raidmgmt (Go library abstracting hardware/software RAID controllers via hexagonal architecture)
argument-hint: <pr-number-or-url>
disable-model-invocation: true
allowed-tools: Read, Bash(gh repo view *), Bash(gh pr view *), Bash(gh pr diff *), Bash(gh pr comment *), Bash(gh api *), Bash(git diff *), Bash(git log *), Bash(git show *)
---

# Review GitHub PR

You are an expert code reviewer. Review this PR: $ARGUMENTS

## Determine PR target

Parse `$ARGUMENTS` to extract the repo and PR number:

- If arguments contain `REPO:` and `PR_NUMBER:` (CI mode), use those values directly.
- If the argument is a GitHub URL (starts with `https://github.com/`), extract `owner/repo` and the PR number from it.
- If the argument is just a number, use the current repo from `gh repo view --json nameWithOwner -q .nameWithOwner`.

## Output mode

- **CI mode** (arguments contain `REPO:` and `PR_NUMBER:`): post inline comments and summary to GitHub.
- **Local mode** (all other cases): output the review as text directly. Do NOT post anything to GitHub.

## Steps

1. **Fetch PR details:**

```bash
gh pr view <number> --repo <owner/repo> --json title,body,headRefOid,author,files
gh pr diff <number> --repo <owner/repo>
```

2. **Read changed files** to understand the full context around each change (not just the diff hunks).

3. **Analyze the changes** against these criteria:

| Area | What to check |
|------|---------------|
| Error handling | Wrap errors with `github.com/pkg/errors` (`errors.Wrap`/`Wrapf`) to preserve stack context — matching the existing code; do not silently drop errors or return bare `err` where a wrap adds useful context. Reuse package-level sentinels (`core.Err*`, `ports.ErrFunctionNotSupportedByImplementation`) instead of inventing duplicate error strings. |
| Hexagonal boundaries | Respect the entity/port/adapter separation (`DESIGN.md`). Domain entities and ports must not import adapter or vendor-CLI packages. New controller support belongs in `pkg/implementation/` behind the existing port interfaces. |
| Port interface compliance | When an adapter changes, verify it still satisfies its port interface and that unsupported operations return `ErrFunctionNotSupportedByImplementation` rather than `nil` or a panic. Interface signature changes ripple to every adapter — check they were all updated. |
| CLI output parsing | Adapters parse external tool output (storcli2, perccli2, ssacli, mdadm, lsblk, smartctl, udevadm). Check for unguarded map/slice/pointer access, missing fields, nil dereferences, and tool-version/format drift. New or changed parsing must have a corresponding `testdata/` fixture. |
| Command execution | Validate how external commands are built and run (`commandrunner`). Watch for unsanitized inputs interpolated into command arguments, missing non-zero exit-code handling, and ignored stderr. |
| Units & conversions | Sizes are in bytes (`uint64`); enums (DiskType, PDStatus, RAID levels) map vendor-specific strings. Check for integer overflow/truncation, wrong unit conversions, and unhandled enum values that should map to an `Unknown` sentinel. |
| Tests | New behavior needs table-driven tests with `testify`. If a mocked interface changed, mockery-generated mocks must be regenerated and committed. Confirm `testdata/` fixtures match the code paths they exercise. |
| Concurrency | If goroutines are introduced, ensure they have clear exit conditions, shared state is guarded, and errors propagate rather than being lost. |
| Public API & compatibility | This is an imported library (`github.com/scality/raidmgmt`). Flag breaking changes to exported entities, port signatures, or sentinel errors, since they affect downstream consumers. |
| Security | No credentials/serials/secrets logged or hardcoded; no command injection via untrusted device paths or identifiers. |

4. **Deliver your review:**

### If CI mode: post to GitHub

#### Part A: Inline file comments

For each issue, post a comment on the exact file and line. Keep comments short (1-3 sentences), end with `— Claude Code`. Use line numbers from the **new version** of the file.

**Without suggestion block** — single-line command, `<br>` for line breaks:
```bash
gh api -X POST -H "Accept: application/vnd.github+json" "repos/<owner/repo>/pulls/<number>/comments" -f body="Issue description.<br><br>— Claude Code" -f path="file" -F line=42 -f side="RIGHT" -f commit_id="<headRefOid>"
```

**With suggestion block** — use a heredoc (`-F body=@-`) so code renders correctly:
```bash
gh api -X POST -H "Accept: application/vnd.github+json" "repos/<owner/repo>/pulls/<number>/comments" -F body=@- -f path="file" -F line=42 -f side="RIGHT" -f commit_id="<headRefOid>" <<'COMMENT_BODY'
Issue description.

```suggestion
first line of suggested code
second line of suggested code
```

— Claude Code
COMMENT_BODY
```

Only suggest when you can show the exact replacement. For architectural or design issues, just describe the problem.

#### Part B: Summary comment

Single-line command, `<br>` for line breaks. No markdown headings — they render as giant bold text. Flat bullet list only:

```bash
gh pr comment <number> --repo <owner/repo> --body "- file:line — issue<br>- file:line — issue<br><br>Review by Claude Code"
```

If no issues: just say "LGTM". End with: `Review by Claude Code`

### If local mode: output the review as text

Do NOT post anything to GitHub. Instead, output the review directly as text.

For each issue found, output:

```
**<file_path>:<line_number>** — <what's wrong and how to fix it>
```

When the fix is a concrete line change, include a fenced code block showing the suggested replacement.

At the end, output a summary section listing all issues. If no issues: just say "LGTM".

End with: `Review by Claude Code`

## What NOT to do

- Do not comment on markdown formatting preferences
- Do not suggest refactors unrelated to the PR's purpose
- Do not praise code — only flag problems or stay silent
- If no issues are found, post only a summary saying "LGTM"
- Do not flag style issues already covered by the project's linter (golangci-lint, configured in `.golangci.yaml`)
