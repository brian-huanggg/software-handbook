---
name: study-topic
description: End-to-end study workflow for this book: pick a topic from the user's learning target, update the catalog, get the plan approved, write the article, make Anki cards, and commit. Use when the user wants to learn or study a topic and have it turned into an article plus flashcards (e.g. "我想學 X", "study X", "幫我規劃並寫一篇 X 加 Anki", "下一篇學什麼").
---

# Study a topic: plan, article, flashcards, commit

This skill only sequences other skills. It does not repeat their rules. The rules live in `write-article` and `create-flashcards`; when they change, this file does not need to.

Rule priority: the user's instruction in this request > this file > the called skills' own rules.

## Workflow

### 1. Choose field and topic

1. Ask the user for their learning target (what they want to be able to do or understand). If the request already states it, do not ask again.
2. Read the matching outline `docs/contents/<field>.md` (for `ai`, the track file under `docs/contents/ai/`). Find where the target fits: the field, the level, the chapter, and the item.
3. If the item exists, use it. If the next unchecked (`- [ ]`) item in that chapter matches the target, prefer it. If the target is not in the outline yet, add a new item as `- [ ]` in the right chapter, in the same style as its neighbours.
4. Only add or edit the item. Do not tick it `- [x]`; `write-article` does that when the article is done.
5. If the item needs a new section, `write-article` adds the sidebar entry. Do not do it here.

### 2. Explain the plan (review gate 1)

Show the user, then stop and wait:

- Field, level, section, and the exact catalog item (and whether it is new or existing).
- The previous and next articles in the outline, so the user sees where it fits.
- The nine-section outline: one or two sentences per section (see `write-article`, `assets/template.md`).
- Whether a code example is planned, and where (`playground/...`).
- If the topic is too large for one article, a proposed split.

Do not start writing until the user approves or changes the plan.

### 3. Write the article

Invoke the `write-article` skill with the approved field, topic, level, and "new". Tell it the outline is already approved, so it skips its own outline confirmation. Let it run its full workflow, including verification and its wrap-up (glossary, ticking the catalog item).

Report back its result before moving on: file paths, checks that failed or were skipped.

### 4. Choose and create flashcards (review gate 2)

1. Draft 5 to 10 candidate questions from the finished article. Prefer questions about concepts, trade-offs, and common mistakes the article explains, not trivia.
2. For each, give the question, a short answer, and the tag in the `<subject>:<topic>:<concept>` format required by `create-flashcards`.
3. Ask the user which to keep, change, or add. The user may add questions of their own.
4. Before calling `create-flashcards`, check that Anki is running and AnkiConnect is reachable. If not, tell the user and wait; do not skip the cards silently.
5. Invoke `create-flashcards` with the approved list. Report how many cards were added.

### 5. Commit

1. Run `git status` and review the diff. Include only files from this task: the article, `playground/` code, sidebar `.ts`, `docs/glossary.md`, and `docs/contents/` changes. If other unrelated changes exist, leave them out and mention them.
2. Commit with a Conventional Commits message, matching the repo history, for example `feat(backend): add cache-stampede article`. Use one commit, unless the catalog change in step 1 is large enough to deserve its own `docs(<field>): ...` commit.
3. Do not push unless the user asks.
4. Final report: commit hash, the article path, the number of cards added, and a reminder that cards live in Anki and are not in git.

## Rules

- Never skip a review gate (steps 2 and 4), even if the user seems in a hurry. The user can approve in one word.
- If a called skill reports a failure it cannot fix (verification failed, AnkiConnect unreachable), stop and tell the user. Do not commit a half-finished result without saying so.
- If the user wants only part of the workflow (only the plan, only cards for an existing article), do only that part.
