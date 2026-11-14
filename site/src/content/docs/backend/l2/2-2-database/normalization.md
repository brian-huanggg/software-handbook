---
title: 正規化：1NF 到 BCNF 與何時反正規化
description: 用函數相依判斷資料表該不該拆，把短網址服務的 links 表一路拆到 3NF 與 BCNF，並學會何時該刻意複製資料。
sidebar:
  order: 1
---

## 先想想

短網址服務 v2 的第一版資料表，把所有東西都放進 `links`：短碼、網址、標籤、擁有者的 email、擁有者的方案與每日配額。

Ann 有 10,000 條連結。她改了 email，背景工作更新到第 9,998 列時程式當掉重啟，沒有人注意到。隔天客服查：

```sql
SELECT DISTINCT owner_email FROM links WHERE owner_id = 7;
```

回來兩筆結果。另一件事：儀表板用 `tags LIKE '%go%'` 篩選 Go 相關的連結，卻撈出一堆標了 `django` 與 `mongo` 的連結。

哪個 email 才是對的？資料庫為什麼不能阻止這種事？要怎麼設計，才能讓「同一個人有兩個 email」變成根本寫不進去的狀態？

<details>
<summary>提示</summary>

先問「同一個事實，現在存了幾份？」。每一份都是一個必須同時更新的地方。

</details>

## 是什麼

**正規化（normalization）** 是依據資料欄位之間的相依關係拆表，讓每一個事實只存在一個地方的設計方法。第一正規化（1NF）、第二正規化（2NF）、第三正規化（3NF）與 Boyce-Codd 正規化（BCNF）是一組逐級加嚴的規則。

反過來，為了讀取效能刻意把同一份資料存在多處，叫做 **反正規化（denormalization）** 。它是正規化之後才做的決定，不是沒做正規化的藉口。

## 為什麼存在

沒有正規化，同一個事實存在很多列，會出三種異常（anomaly）：

- **更新異常（update anomaly）** ：改一個事實要改很多列。Ann 的 email 存在 10,000 列裡，漏改任何一列，資料就自相矛盾。這正是先想想裡的狀況。
- **新增異常（insert anomaly）** ：事實沒有地方放。想新增一個 `enterprise` 方案與它的配額，但 `plan` 與 `daily_quota` 只能依附在某條連結上，沒有連結就存不進去。
- **刪除異常（delete anomaly）** ：刪一個事實連帶刪掉別的。Bob 刪光他唯一的連結，他的 email 與方案也一起消失。

正規化的做法是先找出「誰決定誰」，再讓每個決定關係各有一張表。拆完之後，改 email 只動 `users` 的一列，資料庫保證只有一個答案。

## 怎麼運作

### 函數相依：誰決定誰

**函數相依（functional dependency）** 寫成 `X → Y`，意思是：只要兩列的 X 相同，它們的 Y 就一定相同。例如 `owner_id → owner_email`：同一個擁有者，只會有一個 email。

**超鍵（superkey）** 是能決定整列所有欄位的欄位組合；不能再縮小的超鍵叫 **候選鍵（candidate key）** ，主鍵就是從候選鍵挑出來的一個。`links` 的主鍵 `id` 決定其他所有欄位。

相依是業務規則，不是資料的巧合。不過資料可以拿來「推翻」一條相依，範例裡的 `HoldsFD` 就是這樣檢查的：

```go title="fd.go"
func HoldsFD(rows []Row, fd FD) bool {
	seen := make(map[string]string, len(rows))
	for _, r := range rows {
		l, rhs := project(r, fd.LHS), project(r, fd.RHS)
		if prev, ok := seen[l]; ok && prev != rhs {
			return false
		}
		seen[l] = rhs
	}
	return true
}
```

資料裡看起來成立的相依，不一定是真的規則：只有三位使用者時，`owner_id → plan` 看起來成立，但那是業務決定的，要問產品而不是看資料。

### 第一正規化：每格只放一個值

`tags` 欄位存 `'go,db,tutorial'`，一格裡有多個值。後果有三個：`LIKE '%go%'` 會撈到 `django`；沒辦法對單一標籤建索引或外鍵；要改一個標籤得重寫整串字。

做法是把標籤拆到另一張表，每個標籤一列。

### 第二正規化：不能只依賴鍵的一部分

拆出 `link_tags(link_id, tag, tag_color)` 之後，主鍵是 `(link_id, tag)`。但 `tag_color` 只由 `tag` 決定，跟 `link_id` 無關：

```text
鍵 = (link_id, tag)
tag → tag_color        ← 只依賴鍵的一部分（部分相依）
```

同一個標籤貼在 500 條連結上，顏色就存 500 份。解法是把 `tag → tag_color` 搬到獨立的 `tags` 表。

2NF 只在主鍵由多個欄位組成時才會出問題；單一欄位的主鍵，不可能有「一部分」。

### 第三正規化：不能靠非鍵欄位間接決定

`links` 裡還有兩個遞移相依（transitive dependency）：

```text
id → owner_id → owner_email    （連結決定擁有者，擁有者決定 email）
id → plan → daily_quota        （擁有者的方案決定配額）
```

`owner_email` 不是由連結決定，而是經過 `owner_id` 才決定。這就是先想想裡 10,000 份副本的來源。口訣是「鍵、整個鍵、除了鍵之外什麼都沒有」：每個欄位都只該依賴主鍵，而且是整個主鍵。

拆完之後，整個設計是這樣：

```text
links(id, code, url, tags, owner_id, owner_email, plan, daily_quota)
  │ 1NF：tags 一列一個值
  │ 2NF：tag_color 搬去 tags
  │ 3NF：owner_email 搬去 users；daily_quota 搬去 plans
  ▼
links(id, code, url, owner_id)      users(id, email, plan)
link_tags(link_id, tag)             plans(plan, daily_quota)
tags(name, color)
```

上圖看箭頭下方：每個事實都只出現在一張表，其他表用外鍵引用它。對應的 SQL 如下：

```sql title="after.sql"
CREATE TABLE plans (
    plan        TEXT PRIMARY KEY,
    daily_quota INT  NOT NULL CHECK (daily_quota > 0)
);

CREATE TABLE users (
    id    BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    plan  TEXT NOT NULL REFERENCES plans (plan)
);

CREATE TABLE links (
    id       BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code     VARCHAR(16) NOT NULL UNIQUE,
    url      TEXT        NOT NULL,
    owner_id BIGINT      NOT NULL REFERENCES users (id)
);

CREATE TABLE tags (
    name  TEXT PRIMARY KEY,
    color TEXT NOT NULL
);

CREATE TABLE link_tags (
    link_id BIGINT NOT NULL REFERENCES links (id) ON DELETE CASCADE,
    tag     TEXT   NOT NULL REFERENCES tags (name),
    PRIMARY KEY (link_id, tag)
);
```

### BCNF：每個決定者都必須是鍵

3NF 允許一個例外：決定者不是超鍵，但被決定的欄位是某個候選鍵的一部分。BCNF 把這個例外拿掉：**每一條非平凡的相依，決定者都必須是超鍵。**

舉例：審核員只負責一個網域，一條連結在一個網域上只有一位審核員。

```text
link_review(link_id, domain, reviewer_id)
候選鍵：(link_id, domain)、(link_id, reviewer_id)
(link_id, domain) → reviewer_id    ← 決定者是鍵，沒問題
reviewer_id → domain               ← reviewer_id 不是鍵，違反 BCNF
```

`domain` 屬於候選鍵的一部分，所以 3NF 放行；但 BCNF 不放行：審核員與網域的對應被重複存在每一列。拆成 `reviewer(reviewer_id, domain)` 與 `link_review(link_id, reviewer_id)` 即可。

代價是拆完之後，「一條連結在同一個網域只有一位審核員」這條規則，無法再用單純的鍵或外鍵表達，要靠觸發器或應用程式檢查。這是 BCNF 少數會丟掉約束的例子，所以實務上多數人停在 3NF，遇到這種形狀才處理。

回到一開始的問題：兩個 email 出現，是因為 `owner_email` 依賴 `owner_id` 卻存在 `links` 裡，等於同一事實存了 10,000 份。拆到 3NF 之後，email 只存在 `users` 一列，加上 `UNIQUE`，「同一個人有兩個 email」就寫不進去。`LIKE '%go%'` 的問題由 1NF 解決：標籤成為獨立的列，用 `tag = 'go'` 精確比對。

## 取捨與替代方案

正規化的代價是讀取要 JOIN。大多數時候這個代價很小：主鍵與外鍵都有索引，JOIN 幾張表通常是毫秒等級。反正規化是在量到瓶頸之後才做的選項：

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| 3NF | 每個事實一份，不會自相矛盾；寫入簡單 | 讀取要 JOIN；查詢比較長 | 交易型系統的預設 |
| 複製欄位（`links.owner_email`） | 少一次 JOIN | 要同步每份副本；漏改就矛盾 | 量測過 JOIN 是瓶頸，且複製的欄位極少變動 |
| 計數欄位（`click_count`） | 讀取是 O(1)，免掃明細 | 與明細可能不一致；要處理重試與並行 | 明細表很大、計數被頻繁讀取 |
| 摘要表或物化檢視 | 讀取快，可重算 | 資料落後一段時間；要排程重建 | 報表與儀表板 |
| JSON 欄位 | 結構彈性大，整包讀寫 | 沒有型別與外鍵約束，難以局部更新與查詢 | 不用拆開查詢的附屬資料 |

**什麼時候要先問清楚才反正規化：** 判斷標準有三個，三個都要成立。第一，用 `EXPLAIN` 量過，確定 JOIN 真的是瓶頸。第二，有明確的人或程式負責同步副本。第三，萬一副本壞掉，能從原始資料重新算出來。

注意，有些「複製」其實不是重複。訂單記下下單當時的收件地址，是另一個事實：「當時寄去哪裡」，使用者之後搬家不該改寫它。這種快照要保留，不要把它「正規化」掉。

**什麼時候不該過度正規化：**

- 資料是不可變的事件或日誌（點擊紀錄、稽核紀錄）：寫入後不再更新，沒有更新異常可言，拆開反而讓每次讀取都要 JOIN。
- 取值固定、很少變動的小集合（例如 `status` 只有四種）：用 `CHECK` 約束或 enum 就夠，不必為它開一張查詢表。
- 分析與報表用的資料倉儲：讀多寫少，常以寬表（星狀結構）換取掃描速度。

## 常見坑與故障案例

### 把清單塞進一個欄位

**症狀** ：依標籤篩選的結果混入不相干的連結；改名一個標籤要掃全表。

**原因** ：違反 1NF，`'go,db'` 對資料庫只是一段文字，`LIKE '%go%'` 比的是子字串，不是元素，也無法使用索引。

**怎麼避免** ：用關聯表（`link_tags`）一列一個元素，並在 `(tag, link_id)` 上建索引。Postgres 的陣列或 JSONB 可以當折衷，但仍然沒有元素層級的外鍵。

### 同一個可變事實存在多處

**症狀** ：同一個使用者在不同頁面顯示不同的 email；對帳時 `SELECT DISTINCT` 出現兩筆。

**原因** ：違反 3NF，事實存了多份，而更新只改了其中一部分（程式中斷、漏改某個程式路徑、兩個服務各自寫）。

**怎麼避免** ：每個可變事實只放一張表，其他表存外鍵。需要複製時，先看上一節的三個條件。

### 反正規化的計數器逐漸漂移

**症狀** ：`links.click_count` 與實際點擊數差了幾個百分點，而且沒人說得出哪個才對。

**原因** ：計數欄位是明細的衍生資料。請求重試會重複計數，部分失敗會漏掉，而且沒有明細表可以重算。

**怎麼避免** ：保留明細表當作事實來源，計數欄位只當快取，並定期對帳、重建；或用摘要表按時間窗口重算。

### 過度正規化，一頁要 JOIN 九張表

**症狀** ：列表頁很慢；用 ORM 時請求量暴增，每列再多打幾次查詢。

**原因** ：把每個列舉值、每個很少變的屬性都拆成獨立的表，查一頁要串很多表；加上 ORM 的延遲載入，變成 N+1 查詢。

**怎麼避免** ：只在「這個事實獨立變動」或「它被很多列共用」時才拆表；穩定的小集合用 `CHECK` 或 enum。

這四個都是示意案例，不是特定公司的事故報告。

## 小練習

### 動手寫：找出違反 BCNF 的相依

範例目錄：`playground/backend/l2/2-2-database/normalization/`。

先跑 `go test ./...`，確認 `HoldsFD` 的測試通過。接著補完 `exercise/bcnf.go` 的兩個函式：`IsSuperkey` 判斷一組欄位能不能唯一決定每一列；`BCNFViolations` 從一串相依中，挑出在資料中成立、非平凡（右邊不是左邊的子集），而且左邊不是超鍵的那些。

用下面這行驗證，測試全過就完成了：

```bash
go test -tags exercise ./exercise
```

<details>
<summary>參考解答</summary>

`IsSuperkey`：把每列在指定欄位上的值接成一個 key（用 `\x00` 分隔），放進 `map`，碰到重複就回傳 `false`。

`BCNFViolations`：對每條相依依序檢查三件事，任何一件不符就略過。第一，右邊全部包含在左邊裡（平凡相依）就略過。第二，`HoldsFD` 為 `false`（資料推翻了它）就略過。第三，`IsSuperkey(rows, fd.LHS)` 為 `true` 就略過。剩下的就是違規。

最後一個測試 `TestBCNFIgnoresFDsThatDoNotHold` 會抓到忘了檢查 `HoldsFD` 的實作。

</details>

### 辨識

下面是 AI 產生的短網址服務 v2 資料表與查詢，請做 code review，找出 3 個與本文主題相關的問題。

```sql title="schema.sql"
-- AI 產生：短網址服務 v2 的資料表。
CREATE TABLE links (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    owner_id    BIGINT       NOT NULL REFERENCES users (id),
    owner_email TEXT         NOT NULL,
    code        VARCHAR(16)  NOT NULL UNIQUE,
    url         TEXT         NOT NULL,
    tags        TEXT         NOT NULL DEFAULT '',
    click_count BIGINT       NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX links_owner_idx ON links (owner_id);
```

```sql title="queries.sql"
-- AI 產生：使用者改 email、依 tag 篩選、顯示點擊數。
UPDATE users SET email = $1 WHERE id = $2;

SELECT id, code, url FROM links WHERE tags LIKE '%go%';

UPDATE links SET click_count = click_count + 1 WHERE code = $1;
```

完整檔案在 `recognize/` 目錄（AI 產生）。

<details>
<summary>參考解答</summary>

1. **`links.owner_email`（schema.sql）搭配 `UPDATE users SET email = ...`（queries.sql）** ：email 同時存在 `users` 與 `links`，卻只更新了 `users`，使用者的所有連結都留著舊 email（更新異常）。刪掉 `links.owner_email`，需要時 JOIN `users`。
2. **`tags TEXT` 與 `tags LIKE '%go%'`** ：違反 1NF。查詢會命中 `django`、`mongo`，也無法用索引加速。改成 `link_tags(link_id, tag)`，用 `WHERE tag = 'go'` 精確比對。
3. **`click_count` 是唯一的點擊紀錄，沒有明細表** ：重試會重複計數、失敗會漏計，而且無法對帳或重算。新增 `clicks` 明細表當事實來源，`click_count` 若保留，只當可重建的快取。

這三個錯都能「正常運作」：測試只用一位使用者、一個標籤、一次點擊，就看不出來。

</details>

## 快閃卡

<details>
<summary>函數相依 X → Y 是什麼意思？</summary>

只要兩列的 X 值相同，它們的 Y 值就一定相同。例如 owner_id → owner_email：同一個擁有者只會有一個 email。

</details>

<details>
<summary>三種異常分別是什麼？各舉一個 links 表的例子。</summary>

更新異常：改 email 要改 10,000 列。新增異常：沒有連結就無法新增 enterprise 方案。刪除異常：刪光某人唯一的連結，他的 email 也消失。

</details>

<details>
<summary>表 tags(link_id, tag) 的 tag 欄位寫成 'go,db'，違反哪一級正規化？會有什麼後果？</summary>

違反 1NF（一格多個值）。LIKE '%go%' 會誤中 django，無法對單一標籤建索引或外鍵，改名也要重寫整串字。

</details>

<details>
<summary>主鍵是單一欄位 id 的表，可能違反 2NF 嗎？</summary>

不可能。2NF 檢查的是「只依賴複合主鍵的一部分」，單一欄位的主鍵沒有「一部分」可言。

</details>

<details>
<summary>links(id, owner_id, owner_email) 違反 3NF，原因是什麼？</summary>

id → owner_id → owner_email：owner_email 是靠非鍵欄位 owner_id 間接決定的（遞移相依）。同一個擁有者的 email 會重複存在每條連結裡。

</details>

<details>
<summary>表 link_review(link_id, domain, reviewer_id) 有 reviewer_id → domain，但 reviewer_id 不是鍵。符合 3NF 嗎？符合 BCNF 嗎？</summary>

符合 3NF，因為 domain 是候選鍵 (link_id, domain) 的一部分；不符合 BCNF，因為 BCNF 要求每條非平凡相依的決定者都是超鍵。

</details>

<details>
<summary>把 BCNF 拆表後，原本哪種約束可能丟掉？</summary>

跨欄位的相依約束，例如「一條連結在同一個網域只有一位審核員」。拆完後無法用單純的鍵表達，要靠觸發器或應用程式檢查。

</details>

<details>
<summary>訂單表記著下單當時的收件地址，使用者之後搬家。這算違反 3NF 嗎？要不要同步更新？</summary>

不算。它記的是「當時寄去哪裡」這個歷史事實，不是使用者目前地址的副本，不該被同步更新。

</details>

<details>
<summary>什麼情況下可以考慮在 links 加 click_count 這類計數欄位？</summary>

量過讀取確實是瓶頸、有明確的程式負責更新，而且能從 clicks 明細重新算出來。三個條件缺一，就別加。

</details>

## 延伸閱讀

- [Database normalization（Wikipedia）](https://en.wikipedia.org/wiki/Database_normalization)：各級正規化的正式定義與範例，可以對照本文的例子
- [Boyce–Codd normal form（Wikipedia）](https://en.wikipedia.org/wiki/Boyce%E2%80%93Codd_normal_form)：BCNF 的定義，以及 3NF 與 BCNF 差異的標準範例
- [PostgreSQL：Constraints](https://www.postgresql.org/docs/current/ddl-constraints.html)：`CHECK`、`UNIQUE`、外鍵的官方說明，拆表之後靠它們保住一致性
- [PostgreSQL：JSON Types](https://www.postgresql.org/docs/current/datatype-json.html)：JSON 與 JSONB 的行為與索引方式，評估「塞進 JSON 欄位」時要先讀
- [SQL Antipatterns（Pragmatic Bookshelf）](https://pragprog.com/titles/bksqla/sql-antipatterns/)：把逗號分隔欄位等常見資料表設計錯誤整理成反模式的書
