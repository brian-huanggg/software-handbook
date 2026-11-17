# L1 解答

## Q1

```sql
SELECT name, city
FROM customers;
```

**思路**：`SELECT` 決定要哪些欄，`FROM` 決定從哪張表取。只列需要的欄位，不用 `*`。

**常見錯誤**：

- 欄位名拼錯，出現 `column "xxx" does not exist`。
- 用 `SELECT *`：能跑，但多取不需要的欄位，表加欄位後結果也會跟著變。

## Q2

```sql
SELECT
    name          AS product_name,
    price         AS unit_price,
    stock,
    price * stock AS inventory_value
FROM products;
```

**思路**：`AS` 給欄位或運算式取一個輸出名稱。別名只改結果的欄位名，不改表的結構。運算式沒有天生的名字，最需要別名。

**延伸思考解答**：

1. 欄位名會變成 `?column?`。PostgreSQL 無法替運算式推出名稱，所以用這個預設值。
2. 用雙引號：`price * stock AS "Inventory Value"`。沒加引號的識別字會被轉成小寫，空白也不合法。
3. 會出現 `column "inventory_value" does not exist`。邏輯執行順序是 `FROM`、`WHERE`、`SELECT`，`WHERE` 執行時別名還不存在。要嘛在 `WHERE` 重寫運算式 `price * stock > 50000`，要嘛等 L3 用子查詢或 CTE 包一層。

**常見錯誤**：

- 別名用單引號：`AS 'unit_price'` 會報語法錯誤。單引號是字串，雙引號才是識別字。
- 別名含空白卻沒加雙引號，例如 `AS inventory value`。
- 以為別名可以在 `WHERE` 使用（見延伸思考 3）。`ORDER BY` 例外：它在 `SELECT` 之後執行，可以用別名。
