# AGENTS.md

Rules for AI coding agents (and the humans driving them) contributing to this repository.

## Commits

- Use Conventional Commits: `type(scope): description`, imperative subject, about 72 characters.
- Sign off every commit: `git commit --signoff`.
- The body explains WHY the change exists. The diff already shows WHAT changed, so do not walk through it.
- Keep each commit to one logical change. Fold review fixes into the commit they belong to instead of stacking "address review comments" commits.
- Do not reference review rounds, planning steps, or ticket IDs as the explanation ("pass 2", "batch 1", "as discussed", "fix JIRA-123").
- If an LLM helped write the change, add exactly one trailer: `Assisted-by: LLM`. No model names, vendors, tool names, `Co-authored-by` lines for models, or links to chat sessions or transcripts.
- Commit messages, code, comments, and docs are in English.

## Tests and linting

- Write tests first, then the implementation.
- Tests cover the full contract: what is accepted and what is rejected (invalid input, boundaries, error paths), not only the happy path.
- When a test fails, fix the code, not the test.
- Before pushing, all of these pass with zero issues:

```bash
go test ./...
golangci-lint run
markdownlint-cli2 "**/*.md"
```

- Never disable a linter or a test to get a change through. If a rule is wrong for the project, change `.golangci.yaml` or `.markdownlint-cli2.yaml` with a justification in the same PR.

## Code comments

- A comment says what the code cannot: why it is written this way, constraints, invariants, links to specs or upstream bugs.
- Do not narrate the next line, add section banners, restate function signatures, or leave changelog-style comments ("now handles X"). Git history covers that.
- Keep comment density close to the surrounding code.

## Pull requests

- Fill in `.github/pull_request_template.md` completely. Keep every section and tick only the boxes that are actually done.
- Title in the same `type(scope): description` form as commits.
- Describe what changed and why at a high level. Leave implementation details to the diff.
- Call out breaking changes explicitly.
- Do not include internal hostnames, cluster names, or other private infrastructure details from your own testing.
