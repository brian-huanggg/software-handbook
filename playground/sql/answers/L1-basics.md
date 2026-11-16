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
