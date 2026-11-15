---
name: commit-message
description: Generate a Conventional Commits message for this repo from the staged or working-tree diff. Generate-only: never runs `git commit`. Use when the user asks for a commit message, or when another skill (for example study-topic) needs one before committing. This skill is the single source of truth for types, scopes, and subject rules, and the pr-message skill derives from it.
---

# Commit message

Produce one commit message in the format this repo already uses. Output the text only. Do not run `git commit`, `git add`, or `git push`. The caller (the user, or a skill that cites this one) decides what to do with the message.

## Input

- The diff to describe: staged changes by default (`git diff --cached`). If nothing is staged, use `git diff` and say so.
- Optional hint from the caller: intended type, scope, or a one-line intent.

Read the diff itself. Do not rely on the hint alone.

## Format

```
type(scope): subject

optional body
```

- No footer. No `Co-Authored-By` trailer. No issue references. No `Signed-off-by`.
- `scope` is optional, but omit it only when no scope fits (for example repo-wide `ci:` or `docs:`).
- Types and scopes: read `references/types-scopes.md`. Use only values listed there. If the diff truly needs a new scope, pick a short lowercase one and tell the caller it is new.

## Subject rules

- Imperative mood: `add`, `fix`, `update`, not `added` or `adds`.
- Start lowercase after the colon. No trailing period.
- 72 characters or fewer for the whole first line.
- Say what changed at the level a reader of `git log --oneline` needs. Name the thing (`add db-clock article`), not the activity (`update files`).
- Keep proper nouns and code identifiers in their original case (`Helm`, `Mermaid`, `regal`).
- Write in English, even when the changed content is Chinese.

## Body rules

Add a body only when the subject cannot carry the reason. A body explains why, or what a reviewer could not guess from the diff. Do not restate the diff.

- Blank line between subject and body.
- Wrap at 72 characters.
- Plain sentences or short `-` bullets.
- Breaking change: add `!` after the scope (`feat(site)!: ...`) and state the break in the body.

## One commit, one change

This repo is rebase-merged. Every commit lands on `main` as written, so each commit must stand alone: one logical change, builds and reads on its own.

If the diff mixes unrelated changes (for example an article plus a site bug fix), do not write one blended message. Say the diff should be split, list the proposed groups with the files in each, and give one message per group.

## Output contract

Return exactly this, so a citing skill can use it without parsing prose:

1. The message in one fenced code block.
2. After the block, at most two short lines: any new scope introduced, and any split recommendation.

## Checks before returning

- Type and scope come from `references/types-scopes.md`.
- First line is 72 characters or fewer, imperative, lowercase after the colon, no period.
- No footer or trailer.
- The message describes the diff you read, not the hint.
