---
title: PostgreSQL vs MySQL：機制差異與選型
description: 用同一張短網址資料表在兩個資料庫實測 MVCC、隔離級別與連線模型的差異，學會依工作負載選型，並避開換庫時最常踩的坑。
sidebar:
  order: 15
---

## 先想想

短網址服務 v2 的資料量穩定成長，團隊在討論資料庫要用 PostgreSQL 還是 MySQL。同事說：「兩個都支援交易，也都有 MVCC，隨便選一個。」

你先寫了兩段很普通的程式。第一段是每次點擊都把 `clicks` 加 1。第二段是自訂短碼：先查 `code` 有沒有人用，沒有再 `INSERT`。

同樣的 SQL 丟到兩個資料庫跑，結果不一樣：一邊的資料表在更新後變大一倍；另一邊的 `INSERT` 莫名其妙卡住；而自訂短碼在其中一邊會出現重複。

兩個資料庫都宣稱自己有 ACID 與 MVCC，為什麼行為差這麼多？哪些差異會在服務上線後真的咬到你？

<details>
<summary>提示</summary>

從兩件事想起：「資料列的舊版本放在哪裡」，以及「交易預設看到的是哪一個時間點的資料」。

</details>

## 是什麼

**PostgreSQL 與 MySQL（InnoDB）** 是最常用的兩個開源關聯式資料庫。兩者都支援 SQL、交易與 MVCC，但在舊版本怎麼存、預設隔離級別、連線怎麼處理、資料怎麼複寫這幾件事上，做了不同的設計選擇。

本文的 MySQL 指 MySQL 8.4 搭配 InnoDB（預設的儲存引擎）。MariaDB 是 MySQL 的分支，在取捨一節另外比較。B+ 樹、叢集索引與 heap 的差別，已經在[主鍵選型](/backend/l2/2-2-database/primary-key-choice/)講過，本文直接沿用。

## 為什麼存在

兩者並存，是因為出發點不同。

PostgreSQL 源自 UC Berkeley 的 POSTGRES 研究專案，重視標準符合、正確性與可擴充性：使用者可以自訂型別、索引方法與 extension。MySQL 起初是為網站設計的簡單快速資料庫，早期的預設引擎 MyISAM 沒有交易；InnoDB 到 MySQL 5.5（2010 年）才成為預設。

如果你把兩者當成可互換的零件，換庫那天就會出事：同一份交易程式，在不同預設值下做出不同的事，大多數時候不會報錯，只會悄悄算錯。差異不是誰比較先進，而是設計時押注的方向不同。

## 怎麼運作

範例放在 `playground/backend/l2/2-2-database/postgres-vs-mysql/`，用 Docker Compose 起 PostgreSQL 18 與 MySQL 8.4，兩邊用同一份 DDL 與同一組 SQL。

```bash
make up    # 起 Postgres（55433）與 MySQL（53307）
make demo  # 約 40 秒，印出下面的結果
make down
```

### 一、舊版本放哪裡：MVCC 的兩種做法

**MVCC（多版本並行控制）** 的目標是讓讀取不擋寫入：資料被更新時，舊版本先留著，讓還在讀的交易繼續看到舊的。兩個資料庫的差別在於舊版本留在哪裡。

```text
UPDATE links SET clicks = clicks + 1 WHERE id = 7

PostgreSQL（heap）
  頁面：  [ id=7 clicks=0（舊，dead tuple） ][ id=7 clicks=1（新） ]
  舊版本留在原地，由 VACUUM 回收

MySQL InnoDB（叢集索引）
  葉節點：[ id=7 clicks=1（新） ]
  undo log：[ id=7 clicks=0（舊） ]
  原地更新，舊版本搬進 undo log，由 purge 執行緒回收
```

這張圖要看的是舊版本的位置：PostgreSQL 寫一份新的資料列，舊的留在同一張表；InnoDB 改寫原本那列，舊的複製到另一個區域。

PostgreSQL 留下的舊版本叫 **dead tuple** ，累積太多就是資料表膨脹（bloat）。 **VACUUM** 負責把它們標成可重用的空間；它通常不會把檔案縮小還給作業系統。InnoDB 的 **undo log** 放在獨立的空間，讀取舊快照時要順著 undo 把舊版本重建出來，由背景的 purge 回收。

實測：200,000 筆資料，把每一列的 `clicks` 全部加 1，並關掉這張表的 autovacuum。大小包含索引：

| 階段 | PostgreSQL 18 | MySQL 8.4 |
| --- | --- | --- |
| 載入後 | 15.8 MB | 9.5 MB |
| 全表 UPDATE 第 1 次後 | 31.6 MB | 9.5 MB |
| 全表 UPDATE 第 2 次後 | 44.8 MB | 9.5 MB |
| VACUUM 後 | 44.8 MB | 不適用 |
| VACUUM 後再 UPDATE 一次 | 44.8 MB | 不適用 |

這是在一台筆電各跑一次的結果，絕對數字不能搬去你的環境。趨勢有四個：

- PostgreSQL 的表在第一次全表更新後變成兩倍，這就是 dead tuple。
- 第二次沒有再翻倍，因為 PostgreSQL 讀寫頁面時會順手清掉一部分舊版本；實際數字會浮動。
- VACUUM 之後大小沒有縮小，但再更新一次也沒有再變大，證明空間被重用了。
- MySQL 的表大小沒有變。舊版本沒有消失，是搬進了不在這個數字裡的 undo 空間。

代價各有不同。PostgreSQL 要留意 autovacuum 跟不跟得上，高頻更新的小表尤其要注意。InnoDB 則是長交易會讓 undo 越積越多。

### 二、預設隔離級別：同一個交易，看到不同的世界

**隔離級別（isolation level）** 決定一個交易能看到其他交易的哪些變動。兩個常見的異常：不可重複讀（同一個交易內，同一個查詢兩次結果不同）與幻讀（同一個範圍查詢，第二次多出了新的列）。

兩個資料庫的預設值不同：

```sql
SHOW transaction_isolation;        -- PostgreSQL：read committed
SELECT @@transaction_isolation;    -- MySQL：REPEATABLE-READ
```

實驗一：在交易內數兩次 `links` 的筆數，兩次之間由另一條連線新增一筆並提交。

```text
PostgreSQL 18：同一個交易內數兩次筆數：1 → 2
MySQL 8.4：    同一個交易內數兩次筆數：1 → 1
```

PostgreSQL 的 Read Committed 讓每個語句看到「當下最新已提交」的資料，所以第二次看到新的一筆。MySQL 的 Repeatable Read 在第一次讀取時拍下快照，整個交易都用它。

實驗二更重要：交易 A 鎖定一個沒有任何資料列的範圍，交易 B 想插入範圍內的 id。

```go title="isolation.go"
rows, err := a.QueryContext(ctx, d.Rebind(`SELECT id FROM links WHERE id BETWEEN ? AND ? FOR UPDATE`), 15, 25)
// ...
_, err = b.ExecContext(bctx, d.Rebind(`INSERT INTO links (id, code) VALUES (?, ?)`), 20, "new")
```

資料表裡只有 id 10 與 30，15 到 25 之間是空的。

```text
PostgreSQL 18：插入 id 20 是否被擋住：false
MySQL 8.4：    插入 id 20 是否被擋住：true
```

原因是 InnoDB 在 Repeatable Read 下，加鎖的讀取會使用 **gap lock（間隙鎖）** ：不只鎖住既有的列，也鎖住列與列之間的空隙，防止別人插進來造成幻讀。PostgreSQL 的 `FOR UPDATE` 只鎖得到實際存在的列，空範圍什麼都沒鎖。

PostgreSQL 的 Repeatable Read 是快照隔離，沒有 gap lock；Serializable 用 SSI 偵測衝突，衝突時回報序列化失敗，要求應用程式重試。

這個差異造成兩種相反的問題。MySQL 的 gap lock 會讓不相干的插入互相等待甚至死結。PostgreSQL 的預設級別則擋不住「先查再插」的競態。兩者都放在常見坑。

### 三、連線模型：行程與執行緒

PostgreSQL 為每一條連線啟動一個作業系統行程（process-per-connection）。每條連線有自己的記憶體，連線越多成本越高；預設的 `max_connections` 是 100。MySQL 是每條連線一個執行緒，成本較低，預設是 151。

所以 PostgreSQL 前面通常要放連線池，例如 [PgBouncer](https://www.pgbouncer.org/)，讓大量應用程式連線共用少數幾條真正的資料庫連線。MySQL 在連線數上比較寬鬆。

### 四、複寫與擴展

- PostgreSQL 的串流複寫傳送 WAL（預寫日誌），從節點是主節點的物理副本；另有邏輯複寫，可以只複寫部分資料表。自動切換主節點要靠外部工具，例如 Patroni，或託管服務。
- MySQL 的複寫傳送 binlog，8.4 的預設格式是以資料列為單位（ROW）。官方另有 Group Replication 與 InnoDB Cluster 處理高可用。

分片的生態也不同：MySQL 有 [Vitess](https://vitess.io/docs/)，PostgreSQL 有 Citus 這類 extension。兩邊都做得到，但成熟度與維運經驗不同。

### 五、功能差異速覽

下面幾項都在範例的資料庫上實測過：

| 功能 | PostgreSQL 18 | MySQL 8.4 |
| --- | --- | --- |
| DDL 能否回滾 | 能，`CREATE TABLE` 之後 `ROLLBACK`，表不存在 | 不能，DDL 會隱式提交（實測：交易中 `CREATE TABLE` 後 `ROLLBACK`，先前的 `INSERT` 也已提交） |
| 部分索引（`CREATE INDEX ... WHERE`） | 支援 | 語法錯誤 |
| `UPDATE ... RETURNING` | 支援 | 語法錯誤 |
| 擴充性 | extension 生態（PostGIS、pgvector 等） | 以內建功能為主 |

回到一開始的問題：兩個資料庫都有 MVCC，但 PostgreSQL 把舊版本留在表裡、預設每個語句看最新資料；MySQL 把舊版本搬進 undo、預設整個交易看同一個快照，還會用 gap lock 保護範圍。所以資料表在 PostgreSQL 變大一倍，是 dead tuple；MySQL 的 `INSERT` 被卡住，是 gap lock；重複短碼則是 PostgreSQL 預設級別下，先查再插的檢查沒有生效。

## 取捨與替代方案

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| PostgreSQL | 標準符合度高；型別與索引豐富；DDL 可回滾；extension 生態；複雜查詢的規劃器強 | 要調 autovacuum；連線昂貴，需要連線池；自動切換要外部工具 | 需要地理資訊、向量搜尋、JSON 混合查詢，或查詢複雜、規則嚴格的服務 |
| MySQL 8.4（InnoDB） | 主鍵查詢與簡單交易很快；連線便宜；複寫與維運經驗豐富；Vitess 分片成熟 | 隔離行為與 gap lock 容易誤解；沒有部分索引與 `RETURNING`；DDL 不可回滾 | 簡單的 OLTP、讀多寫少、團隊熟悉 MySQL，或要走 Vitess 分片 |
| MariaDB | 與 MySQL 相近，部分功能領先（例如原生 `UUID` 型別、`INSERT ... RETURNING`） | 與 MySQL 已逐漸分歧，不能當作無痛替代；生態與託管選擇較少 | 已經使用 MariaDB 的系統，或偏好社群治理的專案 |
| MySQL 加 Vitess | 水平分片、連線池與查詢路由 | 維運複雜度大幅上升；跨分片查詢與交易受限 | 單機寫入已到瓶頸的大型服務 |

Uber 在 2016 年公開寫過把部分服務從 PostgreSQL 搬到 MySQL 的理由，包含寫入放大與複寫的考量，見[延伸閱讀](#延伸閱讀)。那是特定規模與特定時間點的決定，後來兩個資料庫都在演進，不要直接當成結論。

**什麼時候不該選 PostgreSQL：**

- 團隊沒有人願意盯 autovacuum 與連線數，而工作負載是大量更新小表。
- 已經有成熟的 MySQL 加 Vitess 分片架構，換庫的成本遠大於收益。

**什麼時候不該選 MySQL：**

- 需要地理資訊、向量搜尋或部分索引這類功能，而你不想自己在應用層補。
- 交易邏輯依賴嚴格的隔離行為與可回滾的 DDL，又沒人熟悉 gap lock 的後果。

## 常見坑與故障案例

### 「先查再插」在 PostgreSQL 的預設級別上撞號

**症狀** ：自訂短碼 API 偶爾產生兩筆 `code` 相同的資料，之後查詢回傳錯誤的網址。

**原因** ：PostgreSQL 預設是 Read Committed。兩個請求同時 `SELECT COUNT(*)`，都看到 0，都通過檢查，都 `INSERT`。`SELECT ... FOR UPDATE` 也救不了：該 `code` 還不存在，沒有任何列可以鎖，這就是實驗二 PostgreSQL 那一半的結果。如果資料表又沒有唯一約束，資料庫不會報錯。

**怎麼避免** ：把唯一性交給資料庫：`code` 加 `UNIQUE`，用 `INSERT ... ON CONFLICT DO NOTHING`（MySQL 是 `INSERT IGNORE` 或捕捉錯誤 1062）判斷是否搶到。不要只靠應用層的先查再插。

### MySQL 的 gap lock 造成插入卡住與死結

**症狀** ：寫入偶爾出現 `ERROR 1205 (HY000): Lock wait timeout exceeded; try restarting transaction`，或 `ERROR 1213 (40001): Deadlock found when trying to get lock; try restarting transaction`。這些語句看起來彼此不相干。

**原因** ：Repeatable Read 下，加鎖讀取或範圍條件的 `UPDATE` 與 `DELETE` 會鎖住索引上的間隙。gap lock 之間不互斥，所以兩個交易可以同時鎖住同一段間隙；接著各自想插入，就會互相等對方放掉鎖，形成死結。

**怎麼避免** ：交易盡量短；用唯一索引的等值條件取代範圍條件加鎖；應用層對 1213 重試；確定不需要 Repeatable Read 時，可以把隔離級別降到 Read Committed，讓大部分 gap lock 不再出現。

### 長交易讓舊版本無法回收

**症狀** ：PostgreSQL 的資料表與索引越長越大，autovacuum 一直在跑卻清不乾淨；MySQL 的 undo 空間增長、查詢變慢。

**原因** ：只要有一個交易還開著，它可能還要讀的舊版本就不能回收。最常見的是交易裡夾了慢操作，例如呼叫外部 HTTP 服務，或交易開了忘記提交，在 `pg_stat_activity` 裡顯示為 `idle in transaction`。VACUUM 長期跟不上，最後會走到 transaction ID wraparound：Sentry 工程團隊公開記錄過這類問題，見[延伸閱讀](#延伸閱讀)。

**怎麼避免** ：交易裡不做網路請求；PostgreSQL 設 `idle_in_transaction_session_timeout`；監控最老的交易的存活時間。

### 連線數爆掉：`too many clients`

**症狀** ：部署新版本後，PostgreSQL 回應 `FATAL: sorry, too many clients already`，整個服務連不上資料庫。

**原因** ：每個應用程式實例都有自己的連線池。20 個實例、每個池 10 條，就是 200 條，超過 `max_connections` 的 100。PostgreSQL 一條連線一個行程，不能隨便調高上限。這是示意案例。

**怎麼避免** ：把連線池大小乘上實例數，檢查是否低於資料庫上限；前面放 PgBouncer 這類連線池。啟用交易層級的連線池時，要確認你用到的 session 層級功能（例如 `SET`、advisory lock）是否相容。

## 小練習

### 動手寫：跨資料庫的 Hit

範例目錄：`playground/backend/l2/2-2-database/postgres-vs-mysql/`。

先跑 `make up` 與 `make demo`，看過三個實驗的輸出。接著補完 `exercise/hit.go` 的 `Hit`：記一次點擊，`code` 不存在就建立並回傳 1，存在就加 1，回傳加完之後的值。同一份函式要在兩個資料庫上都通過，並且 50 個 goroutine 同時呼叫時，每次回傳的值都不同。

```bash
make exercise
```

兩個資料庫的測試都過了就完成。提示：PostgreSQL 有 `RETURNING`，MySQL 沒有。

<details>
<summary>參考解答</summary>

PostgreSQL 用單一語句：`INSERT ... ON CONFLICT (code) DO UPDATE SET clicks = hits.clicks + 1 RETURNING clicks`。

MySQL 用 `INSERT ... ON DUPLICATE KEY UPDATE clicks = clicks + 1`，再 `SELECT clicks`，兩個語句必須在同一個交易裡。這個交易執行 upsert 時已經鎖住那一列，同一個交易內接著讀到的，就是自己寫的值，別人要等這個交易提交才能再加。如果不開交易，兩個語句之間別人可能又加了一次，讀回來的值就會重複。

</details>

### 辨識

下面是 AI 產生的「自訂短碼」API，跑在 PostgreSQL（預設隔離級別）上，請求量不小。請做 code review，找出 3 個與本文主題相關的問題。

```sql title="schema.sql"
-- AI 產生：短網址服務 v2 的 PostgreSQL 資料表
CREATE TABLE links (
    id         BIGSERIAL    PRIMARY KEY,
    owner_id   BIGINT       NOT NULL,
    code       VARCHAR(16)  NOT NULL,
    url        TEXT         NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_links_code ON links (code);
```

```go title="handler.go"
db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
if err != nil {
	http.Error(w, "internal error", http.StatusInternalServerError)
	return
}

tx, err := db.BeginTx(r.Context(), nil)
// ...
var n int
if err := tx.QueryRowContext(r.Context(),
	`SELECT COUNT(*) FROM links WHERE code = $1`, req.Code).Scan(&n); err != nil {
	// ...
}
if n > 0 {
	http.Error(w, "code already taken", http.StatusConflict)
	return
}

// 向外部的安全瀏覽服務確認網址沒有被列入黑名單。
if !isSafeURL(req.URL) {
	http.Error(w, "unsafe url", http.StatusUnprocessableEntity)
	return
}

if _, err := tx.ExecContext(r.Context(),
	`INSERT INTO links (owner_id, code, url) VALUES ($1, $2, $3)`,
	req.OwnerID, req.Code, req.URL); err != nil {
	// ...
}
```

完整程式碼在 `recognize/` 目錄（AI 產生）。

<details>
<summary>參考解答</summary>

1. **先 `SELECT COUNT(*)` 再 `INSERT`，而且 `code` 沒有唯一約束（`idx_links_code` 只是普通索引）** ：PostgreSQL 預設 Read Committed，兩個請求同時通過檢查，就會寫入兩筆相同短碼。這裡用 `FOR UPDATE` 也沒用，因為還沒有那一列可以鎖。改成 `code` 加 `UNIQUE`，直接 `INSERT ... ON CONFLICT DO NOTHING`，看影響的列數判斷有沒有搶到。
2. **交易開著的時候呼叫外部服務 `isSafeURL`** ：這個請求在等 HTTP 回應的那幾百毫秒，交易與連線都被占住，PostgreSQL 的 VACUUM 也無法回收這個交易之後可能還要讀的舊版本。量大時連線池會被耗盡，表也會膨脹。把檢查移到 `BeginTx` 之前，或乾脆整個交易只包住最後的 `INSERT`。
3. **每個請求都呼叫 `sql.Open`，而且從不 `Close`** ：每次都建立一個新的連線池，每個池開出自己的連線，PostgreSQL 為每條連線啟動一個行程，很快撞上 `max_connections`，出現 `too many clients already`。在程式啟動時開一次，之後所有請求共用同一個 `*sql.DB`。

另外，`isSafeURL` 把使用者傳入的網址直接接在查詢字串後面，應該用 `url.QueryEscape`。這和本文主題無關，但 AI 產生的程式碼常有這類疏漏。

</details>

## 快閃卡

<details>
<summary>PostgreSQL 與 MySQL InnoDB 的 UPDATE，舊版本各放在哪裡？</summary>

PostgreSQL 在同一張表（heap）寫一份新的資料列，舊的留在原地成為 dead tuple，由 VACUUM 回收。InnoDB 原地改寫那一列，舊版本放進 undo log，由 purge 回收。

</details>

<details>
<summary>全表 UPDATE 一次之後，PostgreSQL 的資料表大小變成兩倍，之後執行 VACUUM，檔案會縮回去嗎？</summary>

不會。VACUUM 把 dead tuple 占的空間標成可重用，但通常不還給作業系統。之後的更新會重用這些空間，大小不會再增長。

</details>

<details>
<summary>PostgreSQL 與 MySQL 的預設隔離級別各是什麼？差在哪裡？</summary>

PostgreSQL 是 Read Committed：每個語句看當下最新已提交的資料。MySQL 是 Repeatable Read：第一次讀取時拍快照，整個交易沿用。

</details>

<details>
<summary>交易內執行 SELECT COUNT(*) 得到 1，另一條連線新增一筆並提交，同一個交易再次 SELECT COUNT(*)。預設級別下 PostgreSQL 與 MySQL 各得到多少？</summary>

PostgreSQL 得到 2，MySQL 得到 1。Read Committed 每個語句重新看最新資料，Repeatable Read 沿用第一次的快照。

</details>

<details>
<summary>表裡只有 id 10 與 30。MySQL 預設級別下，交易 A 執行 SELECT id FROM links WHERE id BETWEEN 15 AND 25 FOR UPDATE（沒有結果），交易 B 能插入 id 20 嗎？PostgreSQL 呢？</summary>

MySQL 不能，B 會等待：InnoDB 對加鎖讀取使用 gap lock，鎖住 10 到 30 之間的間隙。PostgreSQL 可以，因為 FOR UPDATE 只鎖實際存在的列，空範圍沒有東西可鎖。

</details>

<details>
<summary>為什麼 PostgreSQL 前面通常要放 PgBouncer？</summary>

PostgreSQL 為每條連線啟動一個行程，連線成本高，預設 max_connections 只有 100。連線池讓大量應用程式連線共用少數真正的資料庫連線。

</details>

<details>
<summary>MySQL 出現 ERROR 1205：Lock wait timeout exceeded，可能的原因是什麼？</summary>

有別的交易持有你要的鎖太久：可能是長交易，也可能是 Repeatable Read 下的 gap lock 鎖住了你要插入的間隙。先找出持有鎖的交易並縮短它。

</details>

<details>
<summary>在 MySQL 的交易裡先 INSERT，再 CREATE TABLE，然後 ROLLBACK。那筆 INSERT 還在嗎？</summary>

還在。MySQL 的 DDL 會隱式提交，CREATE TABLE 之前的變動就已經提交了。PostgreSQL 的 DDL 可以回滾。

</details>

<details>
<summary>PostgreSQL 預設級別下，自訂短碼先 SELECT 檢查再 INSERT，為什麼會出現重複？該怎麼修？</summary>

兩個請求可以同時看到「沒有這個 code」，都通過檢查。該 code 還不存在，FOR UPDATE 也沒有列可鎖。正確做法是把唯一性交給資料庫：加 UNIQUE，用 INSERT ... ON CONFLICT 判斷。

</details>

## 延伸閱讀

- [PostgreSQL：Concurrency Control](https://www.postgresql.org/docs/current/mvcc.html)：MVCC 與各隔離級別的官方說明
- [PostgreSQL：Routine Vacuuming](https://www.postgresql.org/docs/current/routine-vacuuming.html)：VACUUM、autovacuum 與 transaction ID wraparound 的官方說明
- [MySQL 8.4：InnoDB Locking](https://dev.mysql.com/doc/refman/8.4/en/innodb-locking.html)：record lock、gap lock 與 next-key lock
- [MySQL 8.4：InnoDB Multi-Versioning](https://dev.mysql.com/doc/refman/8.4/en/innodb-multi-versioning.html)：undo log 與舊版本的重建方式
- [Uber Engineering：Why Uber Engineering Switched from Postgres to MySQL](https://www.uber.com/blog/postgres-to-mysql-migration/)：2016 年的搬遷理由，閱讀時留意它的規模與時間背景
- [Sentry Blog：Transaction ID Wraparound in Postgres](https://blog.sentry.io/transaction-id-wraparound-in-postgres/)：VACUUM 跟不上時會發生什麼的真實經驗
