# Style guide

The articles are written in Traditional Chinese. The Chinese terms and punctuation in this guide are literal examples of the writing rules; do not translate them in the articles.

## Language and tone

- Traditional Chinese (Taiwan usage): 「程式」 not 「程序」, 「資料庫」 not 「數據庫」, 「伺服器」 not 「服務器」, 「執行緒」 not 「線程」.
- The first time a term appears, write 「中文（English）」, for example 「冪等（idempotency）」, and use only the Chinese afterwards. Terms with no common Chinese translation stay in English: goroutine, channel, handler, middleware. `docs/glossary.md` is the authority.
- Put a half-width space between Chinese and English, and between Chinese and digits: 「用 Go 寫一個 TCP 服務」.
- Use full-width punctuation (，。：；「」) in Chinese text; use half-width punctuation inside code and English sentences.
- Address the reader as 「你」 and the author as 「我們」. The tone is a senior colleague guiding a newcomer: direct, concrete, not preachy.
- No empty words: avoid 「眾所周知」「非常重要」「強大的」「簡單來說」.
- One idea per paragraph, and a paragraph is at most 5 lines.

## Readers of each field

The default reader of the whole book is a self-taught junior engineer, but each field starts at a different point. Follow the reader description at the top of that field's outline. Fields with their own rules have a file under `references/fields/`, and that file wins when it conflicts with this one:

| Field | Reader | Difference in writing |
| --- | --- | --- |
| `cs`, `backend` | Self-taught junior engineer | Start from zero, with the depth in the level table below |
| `devops`, `cyber` | Engineer who can already write a backend service | Do not re-teach Go and HTTP basics; link directly back to the backend field |
| `testing` | Engineer who wants to become a senior QA expert, has read backend L1 | See `references/fields/testing.md` |
| `ai` | Engineer who can write basic Python; prerequisites for each track are at the top of the track outline | See `references/fields/ai.md` |

## Length and depth (by level)

| Level | Body word count | Depth |
| --- | --- | --- |
| L0 | 2,000–3,500 | Start from zero, assume no background; use many analogies, and always follow an analogy with the real mechanism |
| L1 | 2,000–3,500 | Assume L0 was read; the focus is "being able to write it" |
| L2 | 3,000–5,000 | Assume the reader can write a basic service; the focus is "what goes wrong in production" |
| L3 | 3,000–5,000 | The focus is design and trade-offs, with quantitative estimates |
| L4 | 2,500–4,500 | The focus is decision frameworks and organization; code can be minimal |

Word count excludes code. If it goes over the upper limit, consider splitting into two articles, and ask the user first.

## The book-wide running project

Whenever the URL shortener service can be used as the example, use it, with the version that matches the level (L1→v1, L2→v2, L3→v3, L4→v4, see `docs/contents/backend.md`). Do not mention features of versions the reader has not reached.

## Code

- Go uses the latest stable version and by default only the standard library. Third-party packages are allowed only when the article topic is itself a package (for example a Redis client).
- Every Go example goes in `playground/<field>/<level>/<section>/<slug>/` and is a standalone module (module path `handbook/playground/<field>/<slug>`): `go.mod`, `main.go` or package files, and `*_test.go`.
- Code snippets in the article must be real excerpts of the example files. The code block title is only the file name, not the path: ` ```go title="main.go" `. The location of the example directory is stated in 「小練習」, not repeated in every block. Only when one article has two or more files with the same name, add the nearest parent directory in front of the file name to tell them apart, for example `policy/policy.go`.
- A single snippet is at most 40 lines. For longer code, paste only the key part and omit the rest with the comment `// ...`.
- Always handle errors; never write `_ = err`. Comments say "why", not "what".
- Commands go in a ` ```bash ` block, and their output goes in a separate ` ```text ` block.

## Exercise types in 「小練習」

「小練習」 always contains two kinds, 1 to 3 exercises in total, from easy to hard:

- **Hands-on**: complete or implement something in Go (Python for the `ai` field), verified by tests. At least 1.
- **Recognition**: show code that looks AI-generated and let the reader practice code review. At least 1. No code is written; the reader answers by reading the code.

Rules for recognition exercises:

- Choose one of two forms: **good vs. bad** (give implementations A and B, ask which has a problem and why), or **find the bugs** (give code that looks reasonable and hides 1 to 3 errors related to the article topic, and ask the reader to point them out).
- The errors must be of the kind AI-generated code often has: it looks like it runs and the tests pass, but it violates a principle from the article (for example missing error handling, over-broad configuration, forgotten resource release, a race condition, a demo shortcut left in production code). Do not use syntax errors or errors that fail to compile.
- Errors may test only concepts the article covered, and at least one error must fall under one item of 「常見坑與故障案例」.
- Put the code under `playground/.../<slug>/recognize/`, give the title as the file name only, and mark it 「AI 產生」. Put a build tag on the wrong version, or give it a file name that `go test ./...` does not scan, so it does not break the tests of the example.
- Put the answer in `<details><summary>參考解答</summary>…</details>`: list line by line what is wrong, what consequence it causes, and the correct way. For good-vs-bad exercises, also explain why the good one is good.
- Do not leak the answer: the question and code comments must not hint at what is wrong, and do not name functions `unsafeXxx` or `buggyXxx`.

## Figures

- In order of preference: Mermaid (a ` ```mermaid ` block; the site renders it with `astro-mermaid`), tables, text diagrams (a ` ```text ` block with box-drawing characters), SVG under `site/src/assets/`.
- Use Mermaid `sequenceDiagram` for message flows between parties, and `flowchart` or `stateDiagram-v2` for decisions and states. Use a text diagram when the layout itself matters (memory layout, byte positions, tree structures).
- Write participant names and labels in Traditional Chinese; keep protocol names, code, and URLs in English. Use `<br/>` for a line break inside a label.
- Under each figure, add one sentence saying what the reader should look at.

## Starlight components

Use only the asides below, at most 4 per article in total:

- `:::note`: supplementary; reading it is not required for understanding
- `:::tip`: practical advice
- `:::caution`: places that cause errors or security problems
- `:::danger`: operations that cause data loss or incidents

## Flashcards (「快閃卡」)

- 5 to 10 per article. One card tests one concept; the front is the question, and the answer on the back takes 1 to 3 sentences.
- Mix the question types and cover at least three: definition (what is X), judgment (will this situation cause Y), causation (why is it designed this way), comparison (how do X and Y differ), debugging (given this symptom, what are the possible causes).
- At least 2 judgment questions, with concrete input (a URL, header, command, or code snippet), not only abstract concepts.
- Answers come only from the article; do not test what the article did not cover.
- No "Is X correct?" / "Yes" questions that give only a conclusion; the back always includes the reason.
- Do not duplicate 「小練習」: exercises require writing code, flashcards are questions answerable in your head without writing anything.
- Order the cards in the order of the article sections, from easy to hard.
- For the format, see 「快閃卡」 in `assets/template.md` and rules 1 and 3 of `references/markdown-lint.md`.

## Citations and facts

- Numbers (latency, throughput, price) must cite a source or say they are estimates.
- When unsure, write 「通常」 or 「多數實作」 and explain the exceptions. Do not invent incidents, papers, or links.
