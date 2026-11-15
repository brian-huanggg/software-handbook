---
name: pr-message
description: Generate a pull request (merge request) title and body for this repo from the branch's commits. Generate-only: never runs `gh pr create`. Derives from the commit-message skill so the PR title and the commits use the same types and scopes. Use when the user asks for a PR or MR description, or when another skill needs one.
---

# PR message

Produce a PR title and body from the commits on the current branch. Output the text only. Do not run `gh pr create`, `git push`, or any command that changes git or GitHub state. The caller decides what to do with the message.

This repo is rebase-merged. Every commit lands on `main` as written, and the PR title does not enter history. So the commits carry the history, and the PR message carries the review context.

## Input

- Base branch: `main` unless the caller says otherwise.
- Commits: `git log --format='%h %s%n%b' main..HEAD`.
- Diff summary: `git diff --stat main...HEAD`.
- Optional hint from the caller: intent, linked ticket, risks.

Read the commits and the diff. Do not describe the branch from the branch name alone.

## Consistency with commits

1. Read `../commit-message/references/types-scopes.md`. Use only its types and scopes.
2. **Title** uses the commit subject format: `type(scope): subject`, with the same rules as `commit-message` (imperative, lowercase after the colon, no trailing period, 72 characters or fewer, English).
3. Title choice:
   - One commit: reuse its subject as the title.
   - Several commits of one type and scope: use that type and scope, and write a subject that covers them all.
   - Mixed types: use the type of the most significant change, and the scope that covers most of the diff. If no scope covers it, omit the scope.
4. **Body** facts come from the commit bodies and subjects. Do not invent a rationale that no commit or diff supports. If the why is missing, mark it `TODO: <question for the author>`.
5. If a commit subject breaks the `commit-message` rules, report it. Do not silently fix it, because rebase merge keeps the commit as written.

## Body template

```
## Summary

<1-3 sentences: why this change exists and what it does for the reader or the site.>

## Changes

- <group of related changes, in the order a reviewer should read them>
- <...>

## Test plan

- [ ] <how the change was checked, as a concrete command or action>
- [ ] <...>

## Notes

<Optional. Risks, follow-ups, things the reviewer should look at first. Omit the section when empty.>
```

Rules:

- **Summary** gives the why. It does not list files.
- **Changes** groups by theme, not one bullet per commit. The commit list is already in the PR's Commits tab.
- **Test plan** lists checks that were run or can be run: `mise` tasks, site build, link check, `conftest`, or a manual check. Do not claim a check passed unless the caller or the session shows it passed. Unrun checks stay unchecked.
- No footer. No `Co-Authored-By`. No `Closes #n` or other issue references.
- Omit empty sections. Keep the whole body short enough to read in one screen.

## Output contract

Return exactly this, so a citing skill can use it without parsing prose:

1. `Title:` followed by the title on one line, inside a fenced code block.
2. The body in a second fenced code block (markdown).
3. After the blocks, at most two short lines: commits that break the `commit-message` rules, and any `TODO` the author must answer.

## Checks before returning

- Title type and scope come from `types-scopes.md`.
- Title is 72 characters or fewer, imperative, lowercase after the colon, no period.
- Every claim in the body traces to a commit or the diff.
- No footer, no trailer, no issue reference.
