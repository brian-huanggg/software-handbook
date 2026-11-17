# L1 基礎（Q1–Q25） 解答

## Q1

```yaml
---
name: api
port: 8080
health_path: /healthz
public: false
owner: null
```

**思路**：map 是一組 `key: value`，key 不可重複，順序不影響意義。YAML 不需要宣告型別，解析器依值的寫法推斷：純數字是 `!!int`，`true` / `false` 是 `!!bool`，`null`、`~` 或空值是 `!!null`，其餘是 `!!str`。`yq` 的 `tag` 會印出推斷結果，適合用來驗證型別。

**延伸思考解答**：

1. `yamllint` 不會報錯，它只檢查語法與風格，`"8080"` 是合法的字串。失敗的是 `.port | tag`，結果是 `!!str`。語法正確不代表型別正確，所以驗收要另外查型別。
2. 兩者用 `yq '.owner'` 都印出 `null`，分不出來。用 `yq 'has("owner")'` 才分得出：刪掉是 `false`，寫 `null` 是 `true`。`keys | length` 也會從 5 變 4。很多工具把「沒寫」當成用預設值，把「寫了 null」當成明確清空，兩者行為可能不同。
3. 值會被誤判成別的型別或語法時要加引號，例如：
   - 看起來像其他型別：`"8080"`、`"true"`、`"null"`、`"1.0"`（L1 Q7–Q8 會深入）。
   - 含 `: `（冒號加空格）或 ` #`（空格加井號），會被當成巢狀 map 或註解。
   - 以 YAML 特殊字元開頭：`*`、`&`、`!`、`{`、`[`、`|`、`>`、`@`、`` ` ``、`%`。

**常見錯誤**：

- 少了開頭的 `---`：`yamllint` 預設會發出 `document-start` warning。不影響解析，加上就沒有 warning。
- 冒號後沒空格：`port:8080` 會被解析成一個字串 `"port:8080"`，不是 map。`yamllint` 不會報錯，`yq 'keys'` 會失敗：`cannot get keys of !!str`。
- 布林寫成 `no`：YAML 1.1 解析器（例如 PyYAML、舊版 Kubernetes 工具）會當成 `false`，YAML 1.2 解析器（例如 `yq` v4）會當成字串 `"no"`。`yamllint` 會對它發出 `truthy` warning，但結束碼仍是 0，所以驗收要靠 `.public | tag`。一律用 `true` / `false`（L1 Q7 會深入）。
- `owner` 寫成 `owner: ""`：這是空字串 `!!str`，不是 null。
- 用 Tab 縮排：YAML 禁止 Tab，雖然本題沒有巢狀，仍建議編輯器設定成插入空格。
