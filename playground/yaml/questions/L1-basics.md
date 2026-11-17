# L1 基礎（Q1–Q25）

必備概念：縮排、scalar、list、map、多行字串、型別推斷。

## Q1：用 map 描述 `api` 元件

**題型**：撰寫

**情境**：團隊要替 `shop` 建一份元件清冊，每個元件一個 YAML 檔，由部署腳本讀取。先從 `api` 開始。腳本會依型別處理欄位，例如把 `port` 當整數比大小，所以型別寫錯會讓腳本出錯。

**需求**：在 `code/L1/Q01.yaml` 寫一個 map，只放下列 5 個 key：

| key | 值 | 型別 |
|---|---|---|
| `name` | `api` | 字串 |
| `port` | `8080` | 整數 |
| `health_path` | `/healthz` | 字串 |
| `public` | 否（不對外開放） | 布林 |
| `owner` | 尚未指派 | null |

**驗收標準**：

```bash
yamllint code/L1/Q01.yaml                  # 沒有 error（warning 可接受）
yq 'keys | length' code/L1/Q01.yaml        # 5
yq '.name' code/L1/Q01.yaml                # api
yq '.port | tag' code/L1/Q01.yaml          # !!int
yq '.health_path | tag' code/L1/Q01.yaml   # !!str
yq '.public | tag' code/L1/Q01.yaml        # !!bool
yq '.owner | tag' code/L1/Q01.yaml         # !!null
```

**提示**（卡關再看，由淺到深）：

1. 方向：map 的每一行是一組 `key: value`，YAML 依值的寫法推斷型別。
2. 欄位：冒號後面要有一個空格。布林用 `true` / `false`，null 用 `null` 或 `~`。檔案開頭可加 `---` 標示文件開始。
3. 近乎答案：數字不要加引號，`port: 8080` 才是整數。

**延伸思考**：

1. 把 `port` 寫成 `"8080"`，`yamllint` 會報錯嗎？哪一行驗收會失敗？
2. 整行刪掉 `owner`，和寫 `owner: null` 有什麼差別？用哪個 `yq` 查詢分得出來？
3. `health_path: /healthz` 沒加引號也是字串。什麼情況下字串一定要加引號？
