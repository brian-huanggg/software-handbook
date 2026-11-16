# SQL 100 題訓練營（PostgreSQL）

從零背景到深度掌握資料庫與 SQL。共 100 題，分 4 個級別，全程使用同一套電商資料集。

## 目錄結構

```
playground/code/sql/
  README.md              # 本檔：學習路徑
  docker-compose.yml     # PostgreSQL 16 環境
  schema.sql             # 建表
  seed.sql               # 範例資料
  questions/
    L1-basics.md         # Q1–Q25
    L2-relations.md      # Q26–Q50
    L3-advanced.md       # Q51–Q80
    L4-deep.md           # Q81–Q100
  answers/
    L1-basics.md
    L2-relations.md
    L3-advanced.md
    L4-deep.md
```

題目與解答分開存放。先自己寫，再對照 `answers/`。

## 環境

```bash
docker compose up -d          # 啟動，首次會自動執行 schema.sql 與 seed.sql
docker compose exec postgres psql -U postgres -d bootcamp   # 進入 psql
docker compose down -v        # 停止並刪除資料，下次啟動會重新載入
```

連線資訊：`localhost:5432`、使用者 `postgres`、密碼 `postgres`、資料庫 `bootcamp`。

初始化腳本只在資料目錄為空時執行。弄壞資料想重來，用 `docker compose down -v` 再 `up -d`。

## 資料集

| 表 | 說明 | 首次使用 |
|---|---|---|
| `customers` | 顧客 | L1 |
| `products` | 商品 | L1 |
| `categories` | 商品分類（含 `parent_id`，供遞迴查詢） | L2 |
| `orders` | 訂單 | L1 |
| `order_items` | 訂單明細 | L2 |
| `payments` | 付款紀錄 | L2 |
| `employees` | 員工（含 `manager_id`，供 self join） | L2 |

## 每題格式

1. **題目**：情境加需求。
2. **預期結果**：欄位名稱與筆數，用來自我驗證。
3. **提示**（可選）：卡關時才看。
4. **解答**（在 `answers/`）：參考 SQL、解題思路、常見錯誤。

## 學習方法

1. 先讀完該級的「必備概念」，再開始做題。
2. 每題先自己寫，執行後比對預期結果的欄位和筆數。
3. 寫錯時先讀錯誤訊息，不要馬上看解答。
4. 做完對照解答，比較寫法差異。
5. 每級最後一題是綜合題。答不出來就回頭重做該級。
6. L3 起，每題都用 `EXPLAIN` 檢查執行計畫。

## 學習路徑

### L1 基礎（Q1–Q25）：能查單表、能改資料

**必備概念**：關聯式資料庫、表、列、欄、主鍵、資料型別、`NULL` 的三值邏輯。

| 題號 | 主題 |
|---|---|
| Q1–Q3 | `SELECT`、欄位別名、`SELECT *` 的取捨 |
| Q4–Q6 | `WHERE`、比較運算子、`AND` / `OR` / `NOT` |
| Q7–Q9 | `ORDER BY`、`LIMIT` / `OFFSET`、`DISTINCT` |
| Q10–Q12 | `IN`、`BETWEEN`、`LIKE` / `ILIKE` |
| Q13–Q14 | `IS NULL`、`COALESCE`、`NULL` 的比較陷阱 |
| Q15–Q17 | 聚合函數：`COUNT`、`SUM`、`AVG`、`MIN`、`MAX` |
| Q18–Q19 | `GROUP BY`、`HAVING`，與 `WHERE` 的差異 |
| Q20 | `CASE WHEN` |
| Q21–Q22 | `INSERT`、`UPDATE`、`DELETE`（含 `WHERE` 的重要性） |
| Q23 | `CREATE TABLE`、資料型別、`DEFAULT` |
| Q24 | 約束：`PRIMARY KEY`、`NOT NULL`、`UNIQUE`、`CHECK` |
| Q25 | **綜合題**：單表報表（篩選、分組、排序、分頁） |

### L2 關聯（Q26–Q50）：能設計表、能跨表查詢

**必備概念**：外鍵、一對多與多對多、正規化（1NF 到 3NF）、集合論基礎。

| 題號 | 主題 |
|---|---|
| Q26–Q28 | 外鍵、參照完整性、`ON DELETE` 行為 |
| Q29–Q32 | `INNER JOIN`、多表 join、join 條件寫錯造成筆數暴增 |
| Q33–Q35 | `LEFT` / `RIGHT` / `FULL JOIN`、找出「沒有」的資料 |
| Q36 | `CROSS JOIN` |
| Q37 | self join（員工與主管） |
| Q38–Q40 | 子查詢：純量、`IN`、衍生表 |
| Q41–Q42 | `EXISTS` / `NOT EXISTS`、`NOT IN` 遇到 `NULL` 的陷阱 |
| Q43–Q44 | `UNION` / `UNION ALL`、`INTERSECT`、`EXCEPT` |
| Q45 | 字串函數：`CONCAT`、`SUBSTRING`、`TRIM`、`REPLACE` |
| Q46 | 日期函數：`DATE_TRUNC`、`EXTRACT`、區間運算 |
| Q47 | 型別轉換：`CAST`、`::` |
| Q48 | `ALTER TABLE`：加欄、改型別、加約束 |
| Q49 | 正規化：把一張寬表拆成多張表 |
| Q50 | **綜合題**：跨 4 張表的銷售報表 |

### L3 進階（Q51–Q80）：能做分析報表、能讀執行計畫

**必備概念**：查詢的邏輯執行順序、index 與 B-tree、交易、ACID。

| 題號 | 主題 |
|---|---|
| Q51–Q54 | CTE（`WITH`）、多層 CTE、CTE 與子查詢的取捨 |
| Q55–Q57 | 遞迴 CTE：分類樹、組織圖 |
| Q58–Q62 | Window function 基礎：`OVER`、`PARTITION BY`、`ROW_NUMBER`、`RANK`、`DENSE_RANK` |
| Q63–Q66 | `LAG` / `LEAD`、累計總和、移動平均、frame（`ROWS BETWEEN`） |
| Q67–Q68 | 每組取前 N 名、去除重複資料 |
| Q69–Q70 | Pivot：`CASE` 加聚合、`FILTER` |
| Q71–Q72 | `UPSERT`（`ON CONFLICT`）、`RETURNING` |
| Q73–Q74 | `VIEW`、`MATERIALIZED VIEW` 入門 |
| Q75–Q77 | Index：建立、使用時機、`EXPLAIN` 與 `EXPLAIN ANALYZE` |
| Q78–Q79 | 交易：`BEGIN` / `COMMIT` / `ROLLBACK`、`SAVEPOINT` |
| Q80 | **綜合題**：用 CTE 與 window function 做留存率或排行分析 |

### L4 深度（Q81–Q100）：能調優、能設計、能診斷

**必備概念**：查詢規劃器、統計資訊、MVCC、鎖。

| 題號 | 主題 |
|---|---|
| Q81–Q83 | 慢查詢診斷：讀懂 `EXPLAIN (ANALYZE, BUFFERS)`、Seq Scan 與 Index Scan |
| Q84–Q86 | 查詢改寫：相關子查詢改 join、`OFFSET` 分頁改 keyset 分頁、避免在欄位上用函數 |
| Q87–Q89 | Index 進階：composite index 欄位順序、partial index、covering index（`INCLUDE`） |
| Q90–Q91 | Isolation level：dirty read、non-repeatable read、phantom read 的重現 |
| Q92–Q93 | 鎖：`SELECT ... FOR UPDATE`、`SKIP LOCKED`、重現並解決 deadlock |
| Q94 | `JSONB`：存取運算子、`GIN` index |
| Q95 | Partition：range partition 與 partition pruning |
| Q96 | Trigger 與 function（PL/pgSQL） |
| Q97 | Gaps-and-islands：連續登入天數 |
| Q98 | Sessionization：用時間間隔切分使用者行為 |
| Q99 | 設計題：為一個新需求設計 schema 與 index |
| Q100 | **總結題**：給一個慢查詢，診斷並優化到指定效能 |

## 完成標準

| 級別 | 達成條件 |
|---|---|
| L1 | 不看資料也能寫出單表的篩選、分組與修改，並說得出 `NULL` 的行為 |
| L2 | 能畫出表之間的關係，能判斷該用哪種 join，能寫出「沒有」類型的查詢 |
| L3 | 能用 CTE 和 window function 寫分析查詢，能看懂 `EXPLAIN` 的主要節點 |
| L4 | 能找出並解決慢查詢，能說明鎖與隔離等級造成的現象，能獨立設計 schema |

## 進度追蹤

- [x] 環境：`docker-compose.yml`、`schema.sql`、`seed.sql`
- [ ] L1：Q1–Q25
- [ ] L2：Q26–Q50
- [ ] L3：Q51–Q80
- [ ] L4：Q81–Q100
