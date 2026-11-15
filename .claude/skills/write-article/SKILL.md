---
name: write-article
description: Write or rewrite one chapter article of this book (computer science, backend, frontend, DevOps, cybersecurity, testing, AI). Use when the user asks to "write an article/chapter" (e.g. "寫一篇", "寫這章"), or names an item in docs/contents/<field>.md or in a track outline under docs/contents/ai/.
---

# Write one book article

The book is written in Traditional Chinese (繁體中文, Taiwan usage) for self-taught junior engineers. The main language is Go; the `ai` field is the exception and uses Python. The book is split into fields, each with one outline file: `docs/contents/cs.md`, `docs/contents/backend.md`, `docs/contents/frontend.md`, `docs/contents/devops.md`, `docs/contents/cyber.md`, `docs/contents/testing.md`. The website is `site/` (Astro Starlight), and each field has its own sidebar.

The `ai` field is split into three tracks, each with one outline file: `docs/contents/ai/foundations.md` (theory foundations), `docs/contents/ai/engineering.md` (applied engineering), `docs/contents/ai/ai-agents.md` (agents). The three tracks share one sidebar, `site/src/sidebar/ai.ts`. Wherever this document writes `<field>`, read it as `ai/<track>` for the `ai` field: the outline is the track file above (the agents track file is named `ai-agents.md` so it is not mistaken for an instruction file for agents, but `<track>` is still `agents`), the article path is `site/src/content/docs/ai/<track>/...`, and the example path is `playground/ai/<track>/...`.
Every article must follow this skill, so that chapters written by different agents read as if written by one author.

Rule priority: the user's instruction in this request > this file > `references/fields/<field>.md` > `references/style-guide.md` > your own habits.

Files of this skill: `assets/template.md` (article template), `references/style-guide.md` (book-wide style), `references/markdown-lint.md` (Markdown writing rules), `references/checklist.md` (delivery checklist), `references/fields/` (extra rules for individual fields; currently `ai.md` and `testing.md`). All paths to these files are relative to this skill directory.

**Language note.** The skill instructions are in English, but the articles are in Traditional Chinese. The nine section headings in `assets/template.md` are literal Chinese strings and must be copied exactly. Examples in these files that show Chinese typography are part of the rules; do not translate them.

## Input

Confirm these four things before starting. If any is missing, ask the user. Do not guess:

1. **Field**: `cs`, `backend`, `frontend`, `devops`, `cyber`, `testing`, or `ai`. For `ai`, also confirm the track: `foundations`, `engineering`, or `agents`.
2. **Topic**: which row (table row) of `docs/contents/<field>.md` (for example "2.3 快取穿透、擊穿、雪崩與對策").
3. **Level**: L0–L4, determined by the chapter the item belongs to. The level sets depth and length (see `references/style-guide.md`). The `cs` field has no levels; it is grouped by topic (C, D, O, N, T, Z). Length and depth always follow L0, and `<level>` in paths becomes the `dir` of that topic group in `site/src/sidebar/cs.ts` (for example `n-network`).
4. **New or rewrite**: when rewriting, read the existing article in full first.

## Workflow

1. **Read the context**
   - `docs/contents/<field>.md`: find the previous and next article, so you know what the reader has already learned and what comes next.
   - Articles already written in the same section (`site/src/content/docs/<field>/<level>/<section>/`): avoid repeating explanations and reuse the same examples.
   - `docs/glossary.md`: use terms exactly as listed.
   - `references/style-guide.md`, `references/markdown-lint.md`, and `assets/template.md`.
   - If the field has `references/fields/<field>.md`, read it too (for `ai`, read `references/fields/ai.md`, regardless of track).
2. **Write the outline**: following the nine sections of `assets/template.md`, write one or two sentences per section about what it will cover. If the topic spans two or more articles, or the length clearly exceeds the range, show the outline to the user for confirmation first.
3. **Write the body**: copy `assets/template.md` and fill it in section order. Keep section headings exactly as they are. Do not add, remove, or reorder sections.
4. **Write the code examples**: Go examples go in `playground/<field>/<level>/<section>/<slug>/`, a standalone Go module that can run on its own. Code for recognition exercises goes under `recognize/` in that directory. Python examples for the `ai` field go in `playground/ai/<track>/<level>/<section>/<slug>/`, a standalone `uv` project (see "Code" in `references/style-guide.md` and `references/fields/ai.md`).
5. **Verify**: go through `references/checklist.md` item by item. If `go` is available, run `go vet ./...` and `go test ./...` for Go examples. If `uv` is available, run `uv run ruff check .` and `uv run pytest` for Python examples. If a tool is missing, or the example needs a GPU or a paid API and cannot run, write "code not verified by execution" and the reason in the report.
6. **Wrap up**
   - Add new terms to `docs/glossary.md`.
   - In `docs/contents/<field>.md`, In the section's table in `docs/contents/<field>.md`, change the row's status from `⬜` to `✅`, turn the Title into a link to the article (`[Title](relative/path.md)`), and set the Description cell to the article's frontmatter `description`.
   - Report: file paths, word count, checks that failed or were skipped, and anything you want the user to decide.

## File locations and naming

```
site/src/content/docs/<field>/<level>/<section>/<slug>.md
playground/<field>/<level>/<section>/<slug>/      # Go example
playground/ai/<track>/<level>/<section>/<slug>/   # Python example for the ai field
```

- `<field>`: `cs`, `backend`, `frontend`, `devops`, `cyber`, `testing`, or `ai/foundations`, `ai/engineering`, `ai/agents`.
- `<level>`: `l0` … `l4`; for the `cs` field it is the `dir` of the topic group, for example `n-network`.
- `<section>`: **always use the `dir` of that section in `site/src/sidebar/<field>.ts`** (for `ai`, the `dir` of that track in `site/src/sidebar/ai.ts`), for example `2-3-cache`. Do not invent a name. If the section is not in the file yet (a new field or a new section), name it "section number plus English kebab-case", and add `{ label, dir }` to the `sections` of the matching level in that file. `label` matches the section heading in the outline (exception for the `cs` field: `label` drops the code letter, so for the heading "N.4 應用層" the `label` is 「應用層」).
- The sidebar shows only sections that already have articles. Articles are ordered by `sidebar.order` in the frontmatter, so no separate registration is needed.
- `<slug>`: English kebab-case, 2 to 5 words, for example `cache-stampede`.

The frontmatter is always:

```yaml
---
title: 快取穿透、擊穿、雪崩與對策     # same as the Title column in docs/contents/<field>.md
description: 一句話說明讀完能做到什麼    # same as the Description column; within 60 characters
sidebar:
  order: 5                             # position of this item within the section (starts at 1)
---
```

- `title` is the Title column of the outline row, 4 to 16 Chinese characters (hard cap 20). If the topic is not in the outline yet, add a row first (see `study-topic`). If a title does not fit the cap, shorten it in the outline row and the frontmatter together.

## Extra rules for security topics

- Attack demonstrations target only programs the reader writes and runs locally (for example an example service on `localhost`). Never target third-party systems or real websites.
- Every attack must come with its fix in the same article, and you must verify that the attack no longer works after the fix.
- At the start of the 「怎麼運作」 section, put a `:::caution` aside reminding readers to test only on systems they own or are authorized to test.
- Do not provide content on evading detection, hiding traces, or large-scale scanning.
