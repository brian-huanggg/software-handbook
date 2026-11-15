# Delivery checklist

Check every item. If an item fails, fix it before delivering, or explain the reason in the report.

The nine section headings named below are the literal Chinese headings of `assets/template.md`: 「先想想」, 「是什麼」, 「為什麼存在」, 「怎麼運作」, 「取捨與替代方案」, 「常見坑與故障案例」, 「小練習」, 「快閃卡」, 「延伸閱讀」.

## Structure

- [ ] File path and frontmatter follow `SKILL.md`
- [ ] `title` is clear and short, at most 30 characters
- [ ] The nine `##` headings match `assets/template.md` exactly, in the correct order
- [ ] All template comments are deleted
- [ ] The end of 「怎麼運作」 answers the question from 「先想想」
- [ ] 「取捨與替代方案」 has a comparison table and a "when not to use it" paragraph
- [ ] Every item in 「常見坑與故障案例」 has symptom → cause → fix
- [ ] 「小練習」 has at least one hands-on exercise and one recognition exercise. The recognition answer lists the wrong place, the consequence, and the correct way, and the question does not leak the answer
- [ ] 「快閃卡」 has 5 to 10 cards, each is a `<details>` with a `<summary>`, and the `<summary>` contains no Markdown syntax
- [ ] The flashcards cover at least three question types, at least 2 are judgment questions, and every answer can be found in the article

## Markdown

- [ ] Follows `references/markdown-lint.md` rule by rule (especially bold next to full-width punctuation, blank lines inside `<details>`, and angle brackets wrapped in backticks)

## Content

- [ ] Word count is within the range for the level
- [ ] No concept the reader has not learned yet; when one is needed, add a link to its chapter
- [ ] At least one figure
- [ ] Examples use the URL shortener service (for the `ai` field, the running example of that track), and the version matches the level
- [ ] Terms match `docs/glossary.md`, and new terms are added
- [ ] If the field has `references/fields/<field>.md`, every rule in it is followed
- [ ] No invented links, numbers, or incidents

## Code

- [ ] `playground/<field>/...` exists and has tests
- [ ] Snippets in the article match the example files, and code block titles contain only the file name (no path)
- [ ] Code for recognition exercises is in `recognize/` and does not make `go test ./...` (or pytest) fail
- [ ] `go vet ./...` and `go test ./...` pass; for the `ai` field, `uv run ruff check .` and `uv run pytest` pass (or the report says they were not run, and why)

## Security topics (if applicable)

- [ ] Attacks target only local example programs, and the authorization reminder is included
- [ ] Every attack has a fix and a verification step

## Wrap-up

- [ ] `docs/contents/<field>.md` is checked off and has the link
