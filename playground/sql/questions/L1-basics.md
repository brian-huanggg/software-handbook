# L1 基礎（Q1–Q25）

必備概念：關聯式資料庫、表、列、欄、主鍵、資料型別、`NULL` 的三值邏輯。

## Q1：列出所有顧客的姓名與城市

**情境**：行銷部想先看顧客名單，只需要姓名和所在城市。

**需求**：從 `customers` 查出每位顧客的 `name` 和 `city`。

**預期結果**：

- 欄位：`name`、`city`
- 筆數：20

**提示**（卡關再看）：`SELECT 欄位1, 欄位2 FROM 表名;`

## Q2：用別名產出庫存價值報表

**情境**：財務部要一份庫存清單，看每項商品壓了多少錢在倉庫裡。報表會直接匯出給非工程人員，欄位名稱要好讀。

**需求**：從 `products` 查出每項商品的名稱、單價、庫存量，以及庫存價值（單價乘以庫存量）。欄位依序命名為：

| 欄位 | 來源 |
|---|---|
| `product_name` | `name` |
| `unit_price` | `price` |
| `stock` | `stock` |
| `inventory_value` | `price` 乘以 `stock` |

**預期結果**：

- 欄位：`product_name`、`unit_price`、`stock`、`inventory_value`
- 筆數：25
- 抽查：`Phone X1` 的 `inventory_value` 是 `83880.00`；`Old Phone Z0` 的是 `0.00`

**提示**（卡關再看）：`SELECT 運算式 AS 別名`。運算式可以是欄位之間的四則運算。

**延伸思考**：

1. 不寫 `AS`、只寫 `price * stock` 時，欄位名稱會變成什麼？
2. 想把別名取成 `Inventory Value`（含空白與大寫），要怎麼寫？
3. 在同一個查詢加上 `WHERE inventory_value > 50000`，會發生什麼事？為什麼？
