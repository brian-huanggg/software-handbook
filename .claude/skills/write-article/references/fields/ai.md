# Extra rules for the `ai` field

Read this file only when writing an article in the `ai` field. When this file conflicts with `references/style-guide.md`, this file wins.

## Readers and writing

- The main language is Python, not Go. Do not start from Python syntax, but when tools such as NumPy, pandas, and PyTorch first appear in L1, explain them clearly.
- Math in the theory foundations track: give intuition first, then the formula, and finally verify the formula with NumPy or PyTorch. Write formulas as inline code or in a ` ```text ` block (the site has no math renderer installed), and explain what every symbol means.
- Do not assume the reader has a GPU or a paid API. An example that needs a GPU must include a scaled-down setting that finishes on a CPU. An exercise that incurs API cost must state the approximate amount, and should prefer the local Ollama way first.
- Model names, prices, and benchmark scores change fast: state the date they were checked, and do not write claims that expire, such as "currently the strongest".
- Security topics in agents and applied engineering (prompt injection, data leakage) also follow "Extra rules for security topics" in `SKILL.md`: attacks target only your own example program, and come with a fix and a verification.

## Running examples

These replace the URL shortener service used by the rest of the book, and differ by track. See the top of each track outline:

- Theory foundations: tabular data, MNIST, and a Traditional Chinese corpus
- Applied engineering: the 「手冊問答助理」 (handbook Q&A assistant)
- Agents: the 「手冊維護 Agent」 (handbook maintenance agent)

The version matches the level (L1→v1, …). The URL shortener service is mentioned only when linking back to the backend field is needed.

## Python examples

- Every example goes in `playground/ai/<track>/<level>/<section>/<slug>/` and is a standalone `uv` project: `pyproject.toml` (pinning the Python version and package versions), `uv.lock`, source code, and `tests/` (pytest).
- Code passes `ruff check`, and public functions have type annotations.
- Training examples fix the random seed, and the README states the expected run time and hardware.
- When an API key is needed, read it only from environment variables and include a `.env.example`. Never include a real key.
- Snippet titles are written the same way as for Go: ` ```python title="qa.py" ` (file name only, no path), and omissions are marked with `# ...`.
- For the other rules (snippet length, comments say "why"), follow "Code" in `references/style-guide.md`.
