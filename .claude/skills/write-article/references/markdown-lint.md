# Markdown 規範

本站是 Astro Starlight，文章是 `.md`。以下規則都是實際會造成渲染錯誤的寫法。

1. 不要混用 HTML 標籤與 Markdown 語法：`<summary>` 等 HTML 標籤的同一行內只用 HTML

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

2. 粗體或斜體符號緊貼全形標點（例如：括號、引號、逗號、句號）時，在外側留一個空白，否則無法渲染。開頭與結尾都適用

**Bad**

```markdown
**同源政策（same-origin policy）**是瀏覽器的一條安全規則 （結尾緊貼「）」，粗體不會渲染）
這條規則叫做**「同源」**政策 （開頭緊貼「「」，粗體不會渲染）
```

**Good**

```markdown
**同源政策（same-origin policy）** 是瀏覽器的一條安全規則
這條規則叫做 **「同源」** 政策
```

3. `<details>` 內要寫 Markdown（段落、清單、程式碼區塊）時，`<summary>` 那一行之後與 `</details>` 之前都要空一行，否則內文會變成純文字

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

4. 內文出現角括號（Go 泛型、路徑佔位符、HTML 標籤名稱）時，一律用反引號包起來，否則會被當成 HTML 標籤吃掉

**Bad**

```markdown
寫一個 func Map[T any](s []T) 回傳 <T>，檔案放在 playground/<field>/ 下
```

**Good**

```markdown
寫一個 `func Map[T any](s []T)` 回傳 `T`，檔案放在 `playground/<field>/` 下
```

5. 程式碼區塊一定要標語言（`go`、`python`、`bash`、`http`、`text`）；區塊內還要放程式碼區塊時，外層改用四個反引號
6. 文章內不寫 `#` 一級標題（Starlight 以 frontmatter 的 `title` 當一級標題）；標題從 `##` 開始，不跳級（`##` 下面不能直接接 `####`）
7. 表格、清單、程式碼區塊、aside（`:::note`）前後都要空一行，否則會黏進上一段
8. 表格儲存格內的 `|` 要寫成 `\|`；儲存格內不能換行，需要多行時改用清單或拆成多列
9. 站內連結用網站路徑，結尾加 `/`，不寫 `.md` 副檔名、不寫相對檔案路徑

**Bad**

```markdown
[同源政策](../n-4-application/same-origin-policy.md)
```

**Good**

```markdown
[同源政策](/cs/n-network/n-4-application/same-origin-policy/)
```

10. 不用行尾兩個空白換行（看不見，也常被編輯器刪掉）；要分段就空一行
11. 不寫裸網址，一律用 `[說明](URL)`；裸網址後緊接中文標點時，標點會被當成網址的一部分
12. 圖片一定要有替代文字：`![同源判斷流程](../../../assets/same-origin.svg)`，不寫 `![](...)`
