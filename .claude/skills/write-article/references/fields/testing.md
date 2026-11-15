# Extra rules for the `testing` field

Read this file only when writing an article in the `testing` field. When this file conflicts with `references/style-guide.md`, this file wins.

## Readers and writing

- The reader is an engineer who wants to become a senior QA expert and has read backend L1.
- Do not re-teach Go syntax or the basic use of `go test`; when needed, link back to backend 1.5. L0 and L1 still start from zero on the concepts of "quality and test design", but not from zero on the programming language.
- The readers of L3 and L4 are moving toward senior roles: the tone is a peer discussing trade-offs, not a step-by-step walkthrough. Write more about the basis for decisions, counterexamples, and "when you should not do this", and less screenshot-style tutorial.
- Every article answers at least one question a senior QA gets asked: which kind of risk this practice reduces, what it costs, and how to convince the team to adopt it.

## Systems under test

- There are two systems under test: the URL shortener service (our own system) and Gitea (someone else's system). The split is in the mapping table at the end of `docs/contents/testing.md`. Gitea examples must state the version used.
- The reader has read backend L1, but has not necessarily finished backend L2–L4. When using features of URL shortener service v2 or later (for example 2.4 contract testing needs the service split of v3), explain the architecture of that version in a sentence or two in the text and link back to the backend field. Do not assume the reader has learned it.

## Code

- The topic of a testing article is often itself a test package (Testcontainers, Pact Go, `playwright-go`, …). This is a case that the "Code" rule of `references/style-guide.md` allows third-party packages for.
