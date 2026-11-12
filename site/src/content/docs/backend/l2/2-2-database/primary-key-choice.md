---
title: 主鍵選型：自增 BIGINT vs UUID vs UUIDv7／Snowflake
description: 用 Postgres 與 MySQL 實測三種主鍵的插入、索引大小與分散式代價，學會挑主鍵，並把「對外 ID」與「內部主鍵」分開。
sidebar:
  order: 2
---

## 先想想

短網址服務 v2 的管理 API 長這樣：`GET /api/links/1042`。有天你發現競爭對手的爬蟲從 1 一路打到 1,000,000，把所有人的連結與點擊數抄走了。

同事提議：「全部換成 UUID v4，ID 猜不到就安全了。」DBA 聽到立刻皺眉：「links 表有兩千萬筆，主鍵換成隨機值，寫入會變慢，索引也會變大。」

兩個人都有道理。主鍵該換嗎？換成什麼？「猜不到」真的等於「安全」嗎？

<details>
<summary>提示</summary>

把「這個 ID 要給誰用」拆開想：資料庫內部用來定位資料列的 ID，和對外給使用者看的 ID，一定要是同一個欄位嗎？

</details>

## 是什麼

**主鍵（primary key）選型** 是替每一列資料決定「用什麼值當唯一識別碼」，常見的有三類：資料庫遞增的 `BIGINT`、隨機的 UUID（v4），以及帶時間順序的 ID（UUID v7、Snowflake ID）。

三者最大的差別是 **順序性** 與 **產生 ID 時要不要找中央協調者** ：遞增整數有順序、但要由資料庫發號；UUID v4 不用協調、但完全無序；v7 與 Snowflake 想兩邊都要。

## 為什麼存在

沒有選型的時候，大家預設用 `AUTO_INCREMENT`，直到遇到兩件事。

第一件是 **對外暴露** ：連續整數會洩漏業務量（訂單編號 5,000 號，就知道你一天賣幾單），也讓列舉攻擊變得零成本，這就是先想想裡爬蟲在做的事。

第二件是 **多個寫入端** ：資料庫自增只在單一節點有保證。多主複寫、分片、離線的手機 App 先建立資料再同步，都沒有一個「下一號」可以問。

UUID 因此被拿來用，但它帶來新問題：16 bytes 比 8 bytes 大，而且 v4 完全隨機，會打亂 B+ 樹的插入模式（見本節的前一個項目「索引原理」）。後來的 UUID v7 與 Snowflake ID，就是為了把「順序」帶回來。

## 怎麼運作

### 先看大小：8 bytes 對 16 bytes

| 型別 | 大小 | 備註 |
| --- | --- | --- |
| `BIGINT` | 8 bytes | 有號整數上限約 9.2 × 10^18 |
| `uuid`（Postgres）、`UUID`（MariaDB 10.7+） | 16 bytes | 原生型別，內部是二進位 |
| `BINARY(16)`（MySQL） | 16 bytes | 用 `UUID_TO_BIN()` 與 `BIN_TO_UUID()` 轉換 |
| `CHAR(36)` | 36 bytes 以上 | 存十六進位字串；大小是 `BIGINT` 的 4.5 倍以上，不要這樣存 |

一個欄位多 8 bytes 看起來不多，但主鍵會被複製到別的地方，這取決於資料庫怎麼存資料。

### InnoDB：主鍵就是資料本身

MySQL InnoDB 的表是 **叢集索引（clustered index）** ：整張表就是一棵以主鍵排序的 B+ 樹，資料列放在葉節點。 **次要索引（secondary index）** 的葉節點存的不是列的位置，而是主鍵值。

所以 InnoDB 裡，主鍵的大小會乘進每一個次要索引。主鍵從 8 變 16 bytes，每個次要索引的每個 entry 也多 8 bytes。

Postgres 不同：表是 **heap** （沒有排序的一堆頁面），主鍵索引和一般索引一樣是獨立的 B+ 樹，所有索引的葉節點存 6 bytes 的 `ctid`（資料列在 heap 的位置）。主鍵變大只影響主鍵索引本身。

### 順序決定插入落在哪裡

```text
遞增主鍵（BIGINT、UUID v7、Snowflake）      隨機主鍵（UUID v4）

   [ 根 ]                                      [ 根 ]
   /  |  \                                     /  |  \
 [1][2] [3 ← 新資料永遠從最右邊進來]          [a][b][c] ← 新資料落在任何一頁
          ↑ 只動最右一頁，其餘頁面是冷的        ↑ 每次插入都要把不同的頁讀進記憶體
          ↑ 頁寫滿就開新頁，頁幾乎是滿的        ↑ 頁寫滿時從中間 split，兩頁各剩約一半
```

這張圖要看的是新資料進來的位置：遞增主鍵只碰最右邊的葉節點，隨機主鍵會碰全部的葉節點。

隨機插入有兩個代價。第一是 **頁分裂（page split）** ：目標頁已滿就得把它切成兩頁，兩頁都只剩約一半滿，索引因此變大。第二是 **快取命中率** ：整棵索引都是「熱」的，一旦索引大過記憶體（buffer pool 或 shared_buffers），每次插入都可能是一次磁碟讀取。

### 實測：同一台機器上的三種主鍵

範例放在 `playground/golang/backend/l2/2-2-database/primary-key-choice/`，用 Docker Compose 起 Postgres 18 與 MySQL 8.4，各插入 200 萬筆（短網址 `links` 表：主鍵、短碼、網址）。記憶體刻意設小（64 MB），讓索引放不進去。

```bash
make up          # 起 Postgres（55432）與 MySQL（53306）
make bench-pg    # 約 1 分鐘
make bench-mysql # 約 7 分鐘
make down
```

Postgres 18 的結果：

| 主鍵 | 插入耗時 | WAL 量 | 主鍵索引大小 | 葉節點填充率 |
| --- | --- | --- | --- | --- |
| `bigint` | 14.1 秒 | 737 MB | 43 MB | 90% |
| `uuid` v4（`gen_random_uuid()`） | 21.3 秒 | 793 MB | 75 MB | 72% |
| `uuid` v7（`uuidv7()`） | 15.0 秒 | 764 MB | 60 MB | 90% |

MySQL 8.4 InnoDB 的結果：

| 主鍵 | 插入耗時 | 叢集索引（資料） | 次要索引（`code`） |
| --- | --- | --- | --- |
| `BIGINT AUTO_INCREMENT` | 74.5 秒 | 299 MB | 149 MB |
| `BINARY(16)` 隨機值（等同 v4） | 253.6 秒 | 510 MB | 172 MB |
| `BINARY(16)` v1 加 `UUID_TO_BIN(uuid, 1)` | 74.8 秒 | 318 MB | 172 MB |

這是在一台筆電各跑一次的結果，並關掉 `fsync` 與降低 durability 設定來突顯索引差異。絕對數字不能搬去你的環境，趨勢才是重點：

- 隨機主鍵在兩個資料庫都最慢，MySQL 上差了 3.4 倍，因為叢集索引被撐成 1.7 倍大，又放不進 64 MB 的 buffer pool。
- Postgres 的 v4 慢了約 1.5 倍；葉節點填充率只有 72%，證明頁分裂讓索引浪費了空間。
- v7（Postgres）與交換過欄位的 v1（MySQL）幾乎追平 `BIGINT`。
- MySQL 的次要索引在兩種 16 bytes 主鍵下都是 172 MB，比 `BIGINT` 的 149 MB 大，這就是「主鍵乘進每個次要索引」。Postgres 的 `code` 索引三種都一樣大。

### UUID_TO_BIN(uuid, 1) 在做什麼

UUID v1 的前面是 `time_low`（時間的低位，變化最快），所以字串排序幾乎是隨機。MySQL 的 `UUID_TO_BIN(uuid, 1)` 把 `time_hi` 與 `time_mid` 換到最前面，高位時間在前，排序就大致遞增。

這個技巧只對 v1 有效；對 v4 這種全隨機值，交換欄位沒有任何效果。MariaDB 10.7 起有原生的 `UUID` 型別，是 16 bytes 儲存。

### UUID v7 與 Snowflake：把順序帶回來

**UUID v7** （RFC 9562，2024 年定稿）的前 48 bits 是 Unix 毫秒時間戳，後面是隨機位元，所以字串比大小就是比時間先後：

```go title="uuidv7.go"
ms := uint64(now.UnixMilli())
u[0] = byte(ms >> 40)
u[1] = byte(ms >> 32)
u[2] = byte(ms >> 24)
u[3] = byte(ms >> 16)
u[4] = byte(ms >> 8)
u[5] = byte(ms)

u[6] = (u[6] & 0x0f) | 0x70 // version 7
u[8] = (u[8] & 0x3f) | 0x80 // variant 10xx
```

Postgres 18 起內建 `uuidv7()`；更早的版本要自己在應用程式產生。

**Snowflake ID** 是 Twitter 提出的 64 bits 整數：41 bits 毫秒時間、10 bits 機器編號、12 bits 同一毫秒內的序號。它保留了 `BIGINT` 的 8 bytes，也能在多台機器上各自產生而不撞號：

```go title="snowflake.go"
ms := s.now().Sub(Epoch).Milliseconds()
switch {
case ms < s.lastMS:
	return 0, ErrClockBackwards
case ms == s.lastMS:
	s.sequence++
	if s.sequence > maxSequence {
		// 這一毫秒的 4096 個序號用完，等到下一毫秒。
		for ms <= s.lastMS {
			ms = s.now().Sub(Epoch).Milliseconds()
		}
		s.sequence = 0
	}
default:
	s.sequence = 0
}
s.lastMS = ms

return ms<<(workerBits+sequenceBits) | s.workerID<<sequenceBits | s.sequence, nil
```

每台機器要有不重複的 `workerID`，而且依賴系統時鐘不能倒退，這是它引入的新維運成本。

回到一開始的問題： **不要只為了「猜不到」就把主鍵整個換成 v4。** 列舉攻擊的根本解法是授權檢查：每個請求都要驗證「這個人能不能看這筆資料」。ID 難猜只是多一層障礙，不是防線。實務上常見的做法是兩個欄位分工：內部主鍵繼續用 `BIGINT`（或 Snowflake ID），讓叢集索引與次要索引保持小而有序；另外加一個 `public_id`（UUID）放在 API 的網址裡，並加上唯一索引。這個次要索引仍然是隨機插入，但它只是一棵獨立的小索引，不是整張表。

## 取捨與替代方案

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| `BIGINT` 自增 | 最小、最快、有序、好讀 | 單一發號者；連續值會洩漏量與易被列舉 | 單機或單主資料庫的內部主鍵 |
| UUID v4 | 任何地方都能產生，不需協調；碰撞機率可忽略 | 16 bytes、完全無序、索引膨脹與寫入變慢 | 資料量小、寫入不頻繁，或只當 `public_id` |
| UUID v7 | 不需協調；時間有序，接近 `BIGINT` 的寫入效能 | 16 bytes；ID 洩漏建立時間；同毫秒內順序依實作而定 | 多寫入端又要保有索引局部性 |
| Snowflake ID | 8 bytes、有序、可分散產生 | 要管理機器編號；依賴時鐘；要自己寫或部署 | 分片或多服務各自發號的大系統 |
| 內部 `BIGINT` 加 `public_id` | 內部效率與對外不可猜測兼得 | 多一個欄位與一棵唯一索引；要做兩種 ID 的轉換 | 有公開 API 的大多數服務 |

**什麼時候不該用 UUID（任何版本）當叢集主鍵：**

- 表很大、次要索引很多的 InnoDB 表：16 bytes 會乘進每個次要索引，空間與快取壓力都放大。
- 你還沒有多寫入端：沒有「不需協調」的需求，就只剩 16 bytes 的代價。

**什麼時候不該用 BIGINT 自增當對外 ID：**

- ID 會出現在網址、API 回應，而且資料有業務敏感性（訂單量、使用者數）。
- 你預期很快要做多主複寫、分片或離線同步，遷移主鍵的成本遠高於一開始就規劃。

## 常見坑與故障案例

### 把 UUID 存成 CHAR(36)

**症狀** ：表一長大，磁碟用量與查詢耗時都比預期差很多，次要索引尤其大。

**原因** ：十六進位字串加四個連字號是 36 個字元，是 16 bytes 二進位的 2.25 倍以上；在 `utf8mb4` 字元集下，索引長度的計算還會用最壞情況（每字元 4 bytes）。InnoDB 又把主鍵複製進每個次要索引。

**怎麼避免** ：Postgres 與 MariaDB 用原生 `uuid` 型別；MySQL 用 `BINARY(16)` 加 `UUID_TO_BIN()`。

### 把「UUID 猜不到」當成授權

**症狀** ：把網址裡的 ID 換成 UUID 後，安全測試仍然通過「用 A 帳號的 token 讀 B 的資料」。

**原因** ：UUID 只是難猜，不是難取得。ID 會出現在瀏覽器歷史、日誌、轉寄的連結、第三方的 Referer 裡；一旦外洩，沒有授權檢查就能被直接使用。v1 還會洩漏產生它的機器位址與時間，v7 會洩漏建立時間。這類漏洞稱為 IDOR（不安全的直接物件參照），詳見資安領域的「壞掉的存取控制與 IDOR」。

**怎麼避免** ：每個以 ID 取資料的查詢都要帶上使用者條件（`WHERE id = ? AND owner_id = ?`），不要依賴 ID 的不可猜測性。

### Snowflake 在時鐘倒退時重複發號

**症狀** ：機器重啟或 NTP 校正後，出現主鍵重複的錯誤，或更糟：兩筆不同資料拿到同一個 ID，其中一筆靜默覆寫另一筆。

**原因** ：Snowflake 的唯一性建立在「同一台機器、同一毫秒、序號不重複」，時鐘倒退會讓它重新發出已經發過的時間戳加序號。

**怎麼避免** ：偵測 `ms < lastMS` 就回錯誤或等待時鐘追上（範例的 `ErrClockBackwards`），不要繼續發號；機器編號用租約或註冊中心分配，避免兩台機器共用同一個編號。

### BIGINT 在 JavaScript 失去精度

**症狀** ：前端拿到的 ID 比資料庫裡的少一點點，點進去就 404。例如 `9007199254740993` 在瀏覽器中被讀成 `9007199254740992`。

**原因** ：JavaScript 的 `Number` 是雙精度浮點數，只能精確表示到 2^53（約 9 × 10^15）。Snowflake ID 是 63 bits，很容易超過這個範圍，而 `JSON.parse` 把它當 `Number` 讀。

**怎麼避免** ：API 把大整數 ID 序列化成字串（Go 的 `json:",string"`），或改用 UUID。這是 Snowflake 與 `BIGINT` 進入前端時共同的問題。

這四個都是示意案例；公開的事故報告不容易對應單一原因，我們沒有硬湊一個。

## 小練習

### 動手寫：ParseV7

範例目錄：`playground/golang/backend/l2/2-2-database/primary-key-choice/`。

先跑 `go test ./...`，確認 `NewV7` 與 `Snowflake` 的測試通過。接著補完 `exercise/parse.go` 的 `ParseV7`：把字串轉回 `UUID`，並拒絕長度錯誤、非十六進位、版本不是 7、variant 不是 `10xx` 的輸入。

用下面這行驗證，測試全過就完成了：

```bash
go test -tags exercise ./exercise
```

<details>
<summary>參考解答</summary>

重點是先檢查長度與連字號的位置，再用 `encoding/hex` 解碼去掉連字號的 32 個字元，最後檢查 `u[6]>>4 == 7` 與 `u[8]>>6 == 0b10`。三個檢查缺任何一個，對應的拒絕測試就會失敗。

</details>

### 辨識

下面是 AI 產生的短網址服務 v2 資料表與 API，連結量已經超過兩千萬筆，跑在 MySQL InnoDB 上。請做 code review，找出 3 個與本文主題相關的問題。

```sql title="schema.sql"
-- AI 產生：短網址服務 v2 的 MySQL 資料表
CREATE TABLE links (
    id         CHAR(36)     NOT NULL PRIMARY KEY,
    owner_id   BIGINT       NOT NULL,
    code       VARCHAR(16)  NOT NULL,
    url        TEXT         NOT NULL,
    created_at DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_code (code),
    KEY idx_owner (owner_id, created_at)
) ENGINE=InnoDB;
```

```go title="handler.go"
func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func getLink(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var l Link
		err := db.QueryRow(
			"SELECT id, owner_id, code, url FROM links WHERE id = ?", r.PathValue("id"),
		).Scan(&l.ID, &l.OwnerID, &l.Code, &l.URL)
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(l)
	}
}
```

完整程式碼在 `recognize/` 目錄（AI 產生）。

<details>
<summary>參考解答</summary>

1. **`id CHAR(36)`（schema.sql）** ：UUID 以字串存，是二進位的 2.25 倍以上，而且 InnoDB 把這個主鍵複製進 `uk_code` 與 `idx_owner` 兩個次要索引，三棵樹都被撐大。改成 `BINARY(16)`，用 `UUID_TO_BIN()` 寫入與 `BIN_TO_UUID()` 讀出。
2. **`newID()` 產生 v4 隨機值並直接當叢集主鍵** ：每次插入都落在 B+ 樹的隨機位置，兩千萬筆的表會頻繁頁分裂，索引放不進記憶體後寫入明顯變慢（本文實測 3.4 倍）。改用 v7（或 Snowflake 當主鍵、UUID 只當 `public_id`），讓插入落在索引尾端。
3. **`getLink` 只用 `id` 查詢，沒有檢查呼叫者是否擁有這筆資料，而且回傳了 `owner_id`** ：UUID 難猜不等於有授權，ID 外洩（日誌、轉寄的連結）就能讀到別人的連結與擁有者。改成 `WHERE id = ? AND owner_id = ?`，其中 `owner_id` 來自驗證過的 session，不是請求參數。

另外，`rand.Read(b)` 的錯誤被忽略，在實務上幾乎不會失敗，但違反本書「錯誤一律處理」的規則。

</details>

## 快閃卡

<details>
<summary>InnoDB 的次要索引葉節點存什麼？這對主鍵選型有什麼影響？</summary>

存的是主鍵值，不是資料列位置。所以主鍵越大，每個次要索引的每個 entry 都跟著變大。

</details>

<details>
<summary>為什麼 Postgres 換成 UUID 主鍵時，其他索引不會跟著變大？</summary>

Postgres 的表是 heap，所有索引的葉節點存的都是 6 bytes 的 ctid，而不是主鍵。主鍵變大只影響主鍵索引本身。

</details>

<details>
<summary>隨機 UUID 當主鍵，為什麼會讓插入變慢？</summary>

新資料落在 B+ 樹的隨機葉節點：目標頁滿了就要頁分裂，頁只剩約一半滿；整棵索引都是熱的，放不進記憶體時每次插入都可能讀一次磁碟。

</details>

<details>
<summary>MySQL 對一個 UUID v4 呼叫 UUID_TO_BIN(uuid, 1)，插入順序會變好嗎？</summary>

不會。swap flag 只是把 v1 的高位時間欄位換到最前面；v4 沒有時間欄位，換了仍是隨機。

</details>

<details>
<summary>UUID v7 和 v4 最大的差別是什麼？</summary>

v7 前 48 bits 是毫秒時間戳，所以大致隨時間遞增，插入落在索引尾端；代價是 ID 會洩漏建立時間。v4 全隨機、無序，但不洩漏時間。

</details>

<details>
<summary>一台 Snowflake 機器的時鐘被 NTP 往回調了 5 秒，繼續用 time.Now() 發號會怎樣？</summary>

會重新發出已經用過的「時間戳加序號」組合，造成重複 ID。正確做法是偵測到時鐘倒退就拒絕發號或等待時鐘追上。

</details>

<details>
<summary>API 回傳 {"id": 9007199254740993}，瀏覽器的 JSON.parse 讀到的值會是什麼？</summary>

9007199254740992。JavaScript 的 Number 只能精確表示到 2^53，超過的整數會被捨入。大整數 ID 要序列化成字串。

</details>

<details>
<summary>網址改用 UUID 之後，GET /links/{uuid} 還需要檢查擁有者嗎？</summary>

需要。UUID 只是難猜，不是難取得；ID 會從日誌、歷史紀錄或轉寄的連結外洩。授權檢查才是防線。

</details>

<details>
<summary>為什麼很多服務用「內部 BIGINT 加對外 public_id」的組合？</summary>

內部主鍵小而有序，叢集索引與次要索引都維持緊湊；對外 ID 難猜又不暴露筆數。代價是多一個欄位與一棵唯一索引。

</details>

## 延伸閱讀

- [RFC 9562：Universally Unique IDentifiers (UUIDs)](https://www.rfc-editor.org/rfc/rfc9562)：v1 到 v8 的正式定義，含 v7 的位元配置與時鐘倒退的處理建議
- [PostgreSQL 18：UUID Functions](https://www.postgresql.org/docs/18/functions-uuid.html)：`gen_random_uuid()` 與 `uuidv7()` 的官方說明
- [MySQL 8.4：Miscellaneous Functions（UUID_TO_BIN）](https://dev.mysql.com/doc/refman/8.4/en/miscellaneous-functions.html#function_uuid-to-bin)：swap flag 的行為與限制
- [MySQL 8.4：Clustered and Secondary Indexes](https://dev.mysql.com/doc/refman/8.4/en/innodb-index-types.html)：InnoDB 次要索引為何帶著主鍵
- [Instagram Engineering：Sharding & IDs at Instagram](https://instagram-engineering.com/sharding-ids-at-instagram-1cf5a71e5a5c)：在 Postgres 上自己拼出類 Snowflake ID 的實例
- [MariaDB：UUID Data Type](https://mariadb.com/kb/en/uuid-data-type/)：10.7 起原生 UUID 型別的儲存與排序行為
