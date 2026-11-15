---
name: create-flashcards
description: Create Anki flashcards and add them to the deck "🍎 Software" through AnkiConnect. Use when the user asks to make Anki cards or flashcards, add questions to Anki, or turn a list of questions into cards (e.g. "做 Anki 卡", "建立 flashcards", "加進 Anki", "把這些問題做成卡片").
---

# create-flashcards

Turn the questions (or Q&A) the user gives into Anki Basic cards and add them to `🍎 Software` through AnkiConnect.

## Fixed rules

1. **Deck**: always `🍎 Software`. Never create or use another deck.
2. **Note type**: default `Basic` (fields `Front`, `Back`). Use `Cloze` only when the user asks for fill-in-the-blank.
3. **Tag**: exactly one tag per card, format `<subject>:<topic>:<concept>`. Three levels, all lowercase, join multiple words with `-`.
   - `subject` must be one of: `cs`, `backend`, `frontend`, `devops`, `cyber`, `testing`, `ai`, `tools` (matches the fields in `docs/contents/`).
   - No prefix such as `se:`.
   - Examples: `cs:networking:origin`, `backend:auth:token`, `cyber:auth:cors`, `devops:docker:images`.
   - If a topic spans subjects, pick the subject the concept mainly belongs to. If unsure, ask the user.
4. **Breadcrumb on the front**: the first line of `Front` is always a breadcrumb derived from the tag, followed by the question.
   - Conversion: capitalize the first letter of each level (exception: `cs` is fully uppercase, `CS`), join with ` &gt; `.
   - Example: `cs:networking:origin` becomes `CS &gt; Networking &gt; Origin`.
   - Style: `<div style="color:#888;font-size:0.8em;margin-bottom:0.8em">CS &gt; Networking &gt; Origin</div>`
5. **HTML only, no Markdown** (Anki does not support it). Never output `**`, backticks, `#`, `- ` lists, or `[]()`.
6. **Leave a space between a closing inline tag and the punctuation that follows it.** If punctuation comes right after an inline tag (`</code>`, `</b>`, etc.), put a space before it. Otherwise Anki does not render it correctly.
   - Bad: `<code>fetch(...)</code>,`
   - Good: `<code>fetch(...)</code> ,`
   - Applies to full-width punctuation too (`，`, `：`, `。`).
7. **Language of card content**: Traditional Chinese (繁體中文). Keep technical terms, code, and protocol names in English.

## HTML usage

| Purpose | Tag |
| --- | --- |
| Emphasize conclusion or keyword | `<b>` |
| Inline code, identifier, URL, header | `<code>` |
| Multi-line code | `<pre><code>…</code></pre>` |
| Unordered list (parallel items) | `<ul><li>` |
| Ordered list (steps, sequence) | `<ol><li>` |
| Block | `<div>` |

- Write `<`, `>`, `&` in text as `&lt;`, `&gt;`, `&amp;`.
- Do not hard-code text or background colors (they become unreadable in dark mode). The gray breadcrumb is the only exception.
- No images, tables, or `<br>` for layout. Use `<div>`, `<ul>`, `<ol>` to separate parts.

## Card quality

- **One card asks one thing.** If a question has several sub-questions, split it into several cards, or confirm with the user that it stays as one.
- Back structure: a **bold conclusion** first, then a `<ul>` or `<ol>` with reasons or key points.
- Keep the back short: one-line conclusion, at most about 5 list items, one sentence each.
- Fix typos and sentences broken by line wraps in the input (e.g. "瀏覽酱" becomes "瀏覽器").
- If the user gives only questions, write the answers yourself. In the report, say "answers written by Claude, please review".

## Workflow

1. **Check AnkiConnect**: `curl -s -m 5 localhost:8765 -X POST -d '{"action":"version","version":6}'`. If there is no response, stop and tell the user to open Anki. Do not fall back to another method.
2. **Confirm the deck and note type exist**: `deckNames`, `modelNames`. If `🍎 Software` is missing, stop and report.
3. **Decide the tag.** Ask the user when needed.
4. **Write the cards**, applying all rules above.
5. **Check for duplicates**: for each card run `findNotes` (e.g. `deck:"🍎 Software" "front:*keyword*"`). If it already exists, skip it and do not overwrite.
6. **Add**: use `addNotes`, always with `"options":{"allowDuplicate":false}`.
   - Build and send JSON containing quotes, Chinese, and HTML with a Python script, to avoid shell-escaping errors. Keep the script in the scratchpad, not in the project.
7. **Report**: number of cards added, note ID and tag of each, skipped cards with reasons, and which answers were written by Claude.
   - A `null` result for an item means that card failed. State it explicitly.

## Prohibited

- Do not modify or delete existing notes unless the user explicitly asks. To modify, call `updateNoteFields` by note ID, and read the target first to confirm its content.
- Do not create decks, change note types, or change Anki settings.
- Do not put the user's personal information in any outgoing request (AnkiConnect is local, but this still applies).

## Example

Input: `origin 由哪三個部分組成？`, tag `cs:networking:origin`

Front:

```html
<div style="color:#888;font-size:0.8em;margin-bottom:0.8em">CS &gt; Networking &gt; Origin</div>
<div>origin 由哪三個部分組成？</div>
```

Back:

```html
<div><b>scheme ＋ host ＋ port</b></div>
<ol>
<li><b>scheme</b> （協定）：<code>https</code></li>
<li><b>host</b> （主機名）：<code>sho.rt</code></li>
<li><b>port</b> （埠）：<code>443</code></li>
</ol>
<div>三者<b>全部相同</b> 才算同源，任一不同即為不同源。</div>
```
