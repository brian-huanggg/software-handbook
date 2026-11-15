# Markdown rules

The site is Astro Starlight and articles are `.md`. Every rule below is a way of writing that really causes a rendering error. Examples that show Chinese text are part of the rule: the problem comes from the Chinese full-width punctuation, so do not translate them.

1. Do not mix HTML tags and Markdown syntax: on the same line as an HTML tag such as `<summary>`, use only HTML.

**Bad**

```html
<details><summary>**Example**</summary></details>

<details><summary>Run the `code`</summary></details>
```

**Good**

```html
<details><summary><strong>Example</strong></summary></details>

<details><summary>Run the <code>code</code></summary></details>
```

2. When bold or italic markers touch full-width punctuation (parentheses, quotation marks, commas, periods), leave one space on the outside, otherwise they do not render. This applies at both the start and the end.

**Bad**

```markdown
**同源政策（same-origin policy）**是瀏覽器的一條安全規則 （the closing marker touches "）", so the bold does not render）
這條規則叫做**「同源」**政策 （the opening marker touches "「", so the bold does not render）
```

**Good**

```markdown
**同源政策（same-origin policy）** 是瀏覽器的一條安全規則
這條規則叫做 **「同源」** 政策
```

3. When a `<details>` block contains Markdown (paragraphs, lists, code blocks), leave a blank line after the `<summary>` line and before `</details>`, otherwise the content becomes plain text.

**Bad**

````markdown
<details>
<summary>參考解答</summary>
```go
fmt.Println("hi")
```
</details>
````

**Good**

````markdown
<details>
<summary>參考解答</summary>

```go
fmt.Println("hi")
```

</details>
````

4. When the text contains angle brackets (Go generics, path placeholders, HTML tag names), always wrap them in backticks, otherwise they are swallowed as HTML tags.

**Bad**

```markdown
寫一個 func Map[T any](s []T) 回傳 <T>，檔案放在 playground/<field>/ 下
```

**Good**

```markdown
寫一個 `func Map[T any](s []T)` 回傳 `T`，檔案放在 `playground/<field>/` 下
```

5. Always give a language to a code block (`go`, `python`, `bash`, `http`, `text`). When a code block must contain another code block, use four backticks for the outer one.
6. Do not write a `#` level-1 heading in an article (Starlight uses the frontmatter `title` as the level-1 heading). Headings start at `##` and never skip a level (a `##` cannot be followed directly by `####`).
7. Leave a blank line before and after tables, lists, code blocks, and asides (`:::note`), otherwise they stick to the previous paragraph.
8. Write `|` inside a table cell as `\|`. A cell cannot contain a line break; when several lines are needed, use a list or split into several rows.
9. Internal links use the site path, end with `/`, and have no `.md` extension and no relative file path.

**Bad**

```markdown
[同源政策](../n-4-application/same-origin-policy.md)
```

**Good**

```markdown
[同源政策](/cs/n-network/n-4-application/same-origin-policy/)
```

10. Do not use two trailing spaces for a line break (invisible, and editors often strip them). To start a new paragraph, leave a blank line.
11. Do not write bare URLs. Always use `[description](URL)`. When Chinese punctuation follows a bare URL directly, the punctuation is treated as part of the URL.
12. Images must always have alt text: `![同源判斷流程](../../../assets/same-origin.svg)`. Never write `![](...)`.
