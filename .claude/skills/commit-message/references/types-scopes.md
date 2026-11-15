# Types and scopes

Shared vocabulary for the `commit-message` and `pr-message` skills. Edit here only. Both skills read this file.

## Types

| Type | Use for |
| --- | --- |
| `feat` | New article, new site feature, new playground module |
| `fix` | Bug fix in site, playground, or CI |
| `docs` | Edits to existing prose, catalogs, glossary, skills, style guide |
| `refactor` | Restructure with no behavior or content change (diagram rewrite, asset cleanup) |
| `ci` | GitHub Actions, CodeQL, dependabot, policy lint |
| `chore` | Tooling and housekeeping (`.gitignore`, `mise.toml`) |
| `test` | Tests only |
| `style` | Formatting only, no meaning change |
| `perf` | Performance change |

Rules of thumb:

- A new article is `feat(<field>)`. Later edits to it are `docs(<field>)`.
- Catalog changes under `docs/contents/` are `docs(contents)`.
- A diff that both adds an article and its catalog entry is one `feat(<field>)` commit, unless the catalog change is large (see `study-topic`).

## Scopes

Pick the scope that names where the change lives.

| Scope | Where |
| --- | --- |
| `backend`, `devops`, `cs`, `cyber`, `frontend`, `testing`, `ai`, `tools` | Article content for that field |
| `contents` | Catalogs in `docs/contents/` |
| `site` | Site features and layout under `site/` |
| `astro` | Astro-specific site code (components, config, outline, header) |
| `skills` | `.claude/skills/` |
| `policy` | Policy lint (regal, conftest) |
| `auth-spring`, `password`, `code` | Playground modules |
| `assets` | Images, favicon, logo |
| `cloudflare` | Deployment config |
| `git` | Git config files |

No scope: repo-wide `ci:` and `docs:` changes (for example a glossary term replaced across the book).

If a change fits none of these, choose a short lowercase noun and report it as a new scope. Then add it to this table.
