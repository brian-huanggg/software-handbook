---
title: 資料庫鎖
description: 在 PostgreSQL 實測行鎖、悲觀鎖、樂觀鎖與 deadlock，學會依衝突頻率選擇鎖的策略，並用固定加鎖順序避開死結。
sidebar:
  order: 16
---

## 先想想

短網址服務 v2 替每個短網址加上「點擊額度」：`quota` 欄位記錄還能被點幾次，額度用完就不再轉址。某個短網址的額度只剩 1，這時有兩個請求幾乎同時進來。

你的程式很直覺：先 `SELECT quota`，確認大於 0，做完安全檢查，再 `UPDATE` 把額度減 1。兩個請求各自都讀到 1，各自通過檢查，各自寫回 0。

兩個請求都成功轉址了，額度卻只被用掉一次。如果換成每個點擊要付費，這就是真金白銀的損失。

你要怎麼讓其中一個請求乖乖等？另外，如果還有「把 A 短網址的額度搬給 B」這種操作，同時有人在做「把 B 的額度搬給 A」，會發生什麼事？

<details>
<summary>提示</summary>

第一題想想：資料庫能不能讓「讀」和「寫」變成一個不可分割的動作，或者讓第二個請求先在門外排隊。第二題想想兩個人各自拿著一把鑰匙，又各自想要對方手上的那一把。

</details>

## 是什麼

**資料庫鎖（lock）** 是資料庫用來協調多個交易同時存取同一份資料的機制：一個交易先宣告「這份資料我要用」，其他想以衝突方式使用的交易就必須等待，直到持有者提交或回滾。

鎖有不同的粒度。**表鎖（table lock）** 鎖整張表，**行鎖（row lock）** 只鎖單一資料列。行鎖又分模式，最常談的兩類是 **共享鎖（shared lock）** （多人可以同時持有，代表「我在讀，請不要改」）與 **排他鎖（exclusive lock）** （同一時間只有一人，代表「我要改」）。

**悲觀鎖** 與 **樂觀鎖** 不是資料庫裡的兩種物件，而是應用程式面對衝突的兩種態度。悲觀鎖（pessimistic locking）假設衝突很常發生，所以先鎖再改；樂觀鎖（optimistic locking）假設衝突很少，所以不加鎖，寫入時才檢查資料有沒有被動過。

最後一個名詞是 **死結（deadlock）**：兩個以上的交易各自持有一些鎖，又各自在等對方手上的鎖，誰都無法前進。

本文以 PostgreSQL 16 實測。MySQL InnoDB 的差異在取捨一節簡短對照；MVCC 與隔離級別的預設值，已在 [PostgreSQL vs MySQL](/backend/l2/2-2-database/postgres-vs-mysql/) 講過。

## 為什麼存在

交易（transaction）保證一組操作一起成功或一起失敗，但不保證兩個交易互相看不見對方正在做的事。「先讀再寫」在單一請求裡看起來完全合理，並行之後就多了一種結果：兩個交易讀到同樣的舊值，各自算出新值，後寫的蓋掉先寫的。這叫 **更新遺失（lost update）**。

沒有鎖，後果是數字悄悄錯掉。我們在範例裡讓 50 個請求同時扣一個額度 30 的短網址，每個請求在讀和寫之間停 5 毫秒，模擬安全檢查：

```text
沒有鎖（先讀再寫）   成功 50，剩餘額度 28
```

50 個請求都成功了，但額度只少了 2。資料庫沒有報錯，沒有任何警告，帳就是不對。這個數字每次執行都不同，但一定出錯，而且負載越高越嚴重。

鎖把「誰先誰後」的決定交給資料庫。代價是等待：持有鎖的交易越久，後面排隊的人越多。整篇文章其實都在談同一件事：怎麼在「不出錯」和「不要等太久」之間找平衡。

## 怎麼運作

範例放在 `playground/backend/l2/2-2-database/database-locks/`，用 Docker Compose 起一個 PostgreSQL 16。

```bash
make up    # 起 Postgres（55434）
make demo  # 約 10 秒，印出下面所有結果
make down
```

### 一、行鎖：誰會加、有哪幾種

行鎖記錄在資料列本身，不是一張要塞滿記憶體的鎖表，所以鎖十萬列也不會用光資源。你不需要手動加鎖，也會碰到它：

- `UPDATE` 與 `DELETE` 會自動對命中的列加行鎖。
- `SELECT ... FOR UPDATE` 等語句，讓你「只讀也加鎖」。
- 外鍵檢查會對被參照的那一列加一個最弱的鎖，避免它被刪掉。

PostgreSQL 的行鎖有四種模式，由弱到強。下表是在範例裡實測的結果，「持有」代表交易 A 已經鎖住那一列，「請求」是交易 B 想用某個模式鎖同一列，`X` 代表 B 會被擋住：

| 持有 \ 請求 | `FOR KEY SHARE` | `FOR SHARE` | `FOR NO KEY UPDATE` | `FOR UPDATE` |
| --- | --- | --- | --- | --- |
| `FOR KEY SHARE` | ok | ok | ok | X |
| `FOR SHARE` | ok | ok | X | X |
| `FOR NO KEY UPDATE` | ok | X | X | X |
| `FOR UPDATE` | X | X | X | X |

先記兩條就夠用：一般的 `UPDATE` 取得的是 `FOR NO KEY UPDATE`（只有改到唯一索引的欄位時才升級成 `FOR UPDATE`）；而兩個交易想同時修改同一列，一定有一方要等。這就是排他。

```go title="probe.go"
// 請求方加上 NOWAIT：搶不到就立刻回錯誤（55P03），不會卡住。
err = b.QueryRowContext(ctx, `SELECT id FROM links WHERE id = $1 `+requester+` NOWAIT`, id).Scan(&x)
if SQLState(err) == CodeLockNotAvail {
	return true, nil
}
```

這段是用 `NOWAIT` 偵測衝突的方式：看的是 SQLSTATE `55P03`，不用真的等下去。

有三個行為值得記住：

- **讀不被擋。** 一般的 `SELECT` 不加行鎖，靠 MVCC 看舊版本，所以不會被寫入者擋住，也不會擋住寫入者。
- **鎖持有到交易結束。** 沒有「用完就放」這回事，`COMMIT` 或 `ROLLBACK` 才釋放。交易越長，鎖越久。
- **等待可以設上限。** `NOWAIT` 搶不到就報錯；`SKIP LOCKED` 直接跳過被鎖的列；連線層級的 `lock_timeout` 則是等太久就放棄。

`SKIP LOCKED` 在任務佇列很好用。實測：id 1 被鎖住時，另一個交易 `SELECT ... FOR UPDATE SKIP LOCKED LIMIT 2`，拿到的是 `[2 3]`，它不等、也不重複領到同一筆。

### 二、悲觀鎖：先鎖再改

悲觀鎖的做法是把「讀」那一步也變成加鎖的動作：

```go title="quota.go"
if err := tx.QueryRowContext(ctx, `SELECT quota FROM links WHERE id = $1 FOR UPDATE`, id).Scan(&q); err != nil {
	return err
}
if q <= 0 {
	return ErrNoQuota
}
time.Sleep(work)
if _, err := tx.ExecContext(ctx, `UPDATE links SET quota = $1, clicks = clicks + 1 WHERE id = $2`, q-1, id); err != nil {
	return err
}
return tx.Commit()
```

`FOR UPDATE` 讓第二個請求的 `SELECT` 停在原地，等第一個提交後才讀到新值。同樣 50 個請求、額度 30：

```text
悲觀鎖（FOR UPDATE）   成功 30，額度用完 20，剩餘額度 0，耗時 242ms
```

成功次數正好等於額度。代價是排隊：30 個成功的請求一個接一個通過，每個持鎖約 5 毫秒的檢查時間，總耗時就被拉長。

### 三、樂觀鎖：不鎖，寫的時候檢查

樂觀鎖在資料列加一個 `version` 欄位。讀的時候記下版本，寫的時候要求版本沒變：

```go title="quota.go"
res, err := db.ExecContext(ctx,
	`UPDATE links SET quota = $1, clicks = clicks + 1, version = version + 1
	 WHERE id = $2 AND version = $3`, q-1, id, ver)
// ...
n, err := res.RowsAffected()
// n == 0 代表版本被別人改過：重讀、重算、重試
```

被擋掉的請求不會卡住，而是得到「0 列受影響」，由應用程式決定要重試還是放棄。這段只示範概念，版本欄位的設計、衝突處理與重試策略，留給下一篇「版本欄位與條件更新」。

實測，同樣的 50 個請求、每個最多重試 3 次：

```text
樂觀鎖（版本，重試 3 次）   成功 7，衝突放棄 43，剩餘額度 23
```

額度沒有超扣，但 50 個請求只有 7 個成功，其他 43 個重試用完還是衝突。原因是這個短網址是「熱點」：50 個請求搶同一列，每一輪只有一個人能贏。樂觀鎖的前提是衝突少，前提不成立時，它就把成本從「等待」變成「白做工與失敗」。

### 四、不用先讀的寫法：原子 UPDATE

這個情境其實還有第四種解法：把判斷和扣減放進同一個語句。

```go title="quota.go"
res, err := db.ExecContext(ctx,
	`UPDATE links SET quota = quota - 1, clicks = clicks + 1 WHERE id = $1 AND quota > 0`, id)
```

```text
原子 UPDATE   成功 30，額度用完 20，剩餘額度 0，耗時 74ms
```

它之所以安全，是因為 `UPDATE` 自己會對命中的列加行鎖。第二個請求撞上被鎖的列時會等；等到第一個提交後，PostgreSQL 在預設的 Read Committed 下會對最新版本的資料列重新檢查 `WHERE` 條件，所以 `quota > 0` 是用最新的值判斷的。結果是又對又快：沒有往返、持鎖時間最短。

限制也很清楚：只有「判斷與修改都能寫成一條 SQL」時才用得上。一旦中間要呼叫外部服務、或要根據讀到的內容做複雜計算，就得回到悲觀鎖或樂觀鎖。

### 五、死結：互相等的環

前面的鎖都只有一個，死結要兩個以上才會出現。情境：「把 A 的額度搬給 B」與「把 B 的額度搬給 A」同時發生，每個交易先扣來源、再加目的地。

```text
交易 A（搬 1→2）                     交易 B（搬 2→1）
UPDATE ... WHERE id = 1  ← 鎖住 1
                                     UPDATE ... WHERE id = 2  ← 鎖住 2
UPDATE ... WHERE id = 2  ← 等 B
                                     UPDATE ... WHERE id = 1  ← 等 A
          A 等 B，B 等 A：形成環
```

這張圖要看的是箭頭構成的環：A 握著 1 等 2，B 握著 2 等 1，沒有任何一方的等待會自己結束。

資料庫不會讓它永遠卡下去。PostgreSQL 在等待超過 `deadlock_timeout`（預設 1 秒）後，會檢查等待關係有沒有環，有就挑一個交易中止：

```text
ERROR:  deadlock detected
DETAIL:  Process 123 waits for ShareLock on transaction 456; blocked by process 789.
```

應用程式看到的是 SQLSTATE `40P01`。實測結果：

```text
相反順序：交易 A 被資料庫回滾（SQLSTATE 40P01）
相反順序：交易 B 成功
相反順序：耗時 1.108s
```

被中止的交易，它做過的修改全部回滾，由應用程式決定要不要重試。注意耗時：光是偵測就花了超過 1 秒，這段時間兩個請求都卡著。

最有效的預防不是偵測，而是讓環不可能形成：**所有交易都以同樣的順序取得鎖**。把要鎖的兩列依 id 由小到大鎖，A 與 B 都先鎖 1 再鎖 2，後到的人只會在第一把鎖前排隊，不會握著一把等另一把。

```go title="deadlock.go"
ids := []int64{from, to}
sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
for _, id := range ids {
	var q int
	if err := tx.QueryRowContext(ctx, `SELECT quota FROM links WHERE id = $1 FOR UPDATE`, id).Scan(&q); err != nil {
		return err
	}
	time.Sleep(pause)
}
```

```text
固定順序：交易 A 成功
固定順序：交易 B 成功
固定順序：耗時 425ms
```

兩個交易都成功，沒有任何回滾。順序固定只是預防，不能保證不會死結：鎖的對象多或程式路徑複雜時，仍可能漏掉。所以應用程式還是要把 `40P01` 當成「可以重試」的錯誤，並設重試上限。

### 回到一開始的問題

兩個請求同時搶最後一點額度時，要讓其中一個等：用 `FOR UPDATE` 悲觀鎖，或乾脆把判斷寫進 `UPDATE ... WHERE quota > 0`；能用原子 `UPDATE` 時優先用它，因為最簡單也最快。

至於 A、B 互相搬額度，兩邊各握一把等對方那一把，就是死結。解法是讓所有交易用同樣的順序取得鎖（例如依 id 排序），並且在程式裡對 `40P01` 做有限次數的重試。

## 取捨與替代方案

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| 原子 `UPDATE ... WHERE` | 最快、最簡單，沒有往返，持鎖時間最短 | 判斷與修改必須能寫成單一語句 | 計數器、額度、庫存這類「條件扣減」 |
| 悲觀鎖（`FOR UPDATE`） | 保證不衝突，不需要重試邏輯 | 持鎖期間別人排隊；順序不一致會 deadlock；交易拖長整體變慢 | 衝突頻繁，或讀完要做較複雜的計算再寫 |
| 樂觀鎖（版本欄位） | 不持鎖，讀取與計算期間不擋人；適合跨請求的編輯流程 | 衝突時要重試；熱點資料大量失敗，白做工 | 衝突很少，例如使用者編輯自己的設定 |
| 提高隔離級別（Serializable） | 把正確性交給資料庫，不用自己想鎖 | 衝突會報序列化失敗，仍要重試；效能成本較高 | 邏輯複雜、難以用單一鎖表達 |
| 表鎖（`LOCK TABLE`） | 簡單粗暴，保證互斥 | 整張表被擋，吞吐量大降 | 維運與批次作業，不是一般請求路徑 |

InnoDB 與 PostgreSQL 在這裡有幾個差異：InnoDB 的行鎖加在索引上，所以沒有索引可用的條件會讓它鎖住掃描過的大片資料；在預設的 Repeatable Read 下，加鎖的讀取還會產生 gap lock，把不存在的間隙也鎖起來，這是 [PostgreSQL vs MySQL](/backend/l2/2-2-database/postgres-vs-mysql/) 那篇實測過的差異。MySQL 的死結錯誤碼是 `1213`，細節可以用 `SHOW ENGINE INNODB STATUS` 查看。

**什麼時候不該用鎖：**

- 一條原子 `UPDATE` 就能表達時，不要先 `SELECT ... FOR UPDATE` 再算再寫，多兩次往返、多一段持鎖時間。
- 熱點資料用樂觀鎖。幾十個請求搶同一列，大部分只會失敗重試。
- 交易裡要呼叫外部服務時，不要一邊持鎖一邊等網路，先做完外部呼叫，再開短交易寫入。
- 需要跨多個請求或機器的互斥（例如「同一個使用者同時只能有一個匯出任務」）時，行鎖不適用：它只活在一個交易裡，交易結束就消失。這類需求會在後面的分散式鎖再談。

## 常見坑與故障案例

### 先讀後寫沒有鎖，額度或庫存悄悄超賣

**症狀**：額度、庫存、餘額對不起來。壓測或活動流量高峰才出現，沒有任何錯誤日誌。

**原因**：預設的 Read Committed 下，兩個交易可以讀到同一個舊值，各自算出新值，後寫的蓋掉先寫的。「先查再算再寫」的程式碼看起來沒問題，單元測試也一定會過，因為測試是單執行緒跑的。

**怎麼避免**：能寫成一條 `UPDATE ... WHERE` 就這麼寫；做不到就在讀的時候用 `FOR UPDATE`，或改用樂觀鎖。並加一個並行測試，像範例裡的 50 個 goroutine 搶同一列。

### 交易裡呼叫外部服務，鎖與連線被占住

**症狀**：平常很快的請求，在某個外部服務變慢時，延遲一起飆高，連線池耗盡，接著 `pg_stat_activity` 裡出現一大排 `idle in transaction` 的連線，以及一堆在等鎖的語句。

**原因**：鎖持有到交易結束。如果交易裡夾了一次 HTTP 呼叫，外部服務慢 3 秒，鎖就多握 3 秒；後面所有想動同一列的請求，全部跟著排隊。這是示意案例，但模式很常見。

**怎麼避免**：交易只包資料庫操作。先做完網路請求，再開交易，用最短的時間完成讀寫。同時設定 `idle_in_transaction_session_timeout`，讓忘記提交的連線被自動清掉。

### 加鎖順序不一致，高峰期出現 deadlock

**症狀**：日誌偶爾出現 `ERROR: deadlock detected`，應用端看到 `40P01`。平常不出現，流量一大就出現，而且總是發生在「同時動兩筆以上資料」的操作。

**原因**：不同程式路徑用不同順序鎖同樣的列。A 路徑先鎖訂單再鎖庫存，B 路徑先鎖庫存再鎖訂單，兩邊剛好同時進來就形成環。

**怎麼避免**：定一條全專案共用的順序規則，例如一律依主鍵由小到大；一次要鎖多列時，先排序再逐一鎖。對 `40P01` 做有上限的重試，並加上隨機延遲，避免兩個被回滾的交易同時重來、再撞一次。

### DDL 排在長交易後面，整張表被卡死

**症狀**：某次上線執行 `ALTER TABLE links ADD COLUMN ...`，原本只該花幾毫秒，結果整個服務對 `links` 的所有讀寫都卡住，連簡單的 `SELECT` 也不例外。

**原因**：`ALTER TABLE` 需要表上最強的鎖。如果當時有一個很久沒結束的交易還握著這張表的鎖，DDL 就得排隊；而在它後面進來的所有請求，也都排在 DDL 後面。一個長交易加一條 DDL，就擋住整張表。這是示意案例。

**怎麼避免**：執行 DDL 前先設 `SET lock_timeout = '2s'`，搶不到鎖就報錯放棄，之後重試，比起讓整個服務卡住便宜得多。上線前先檢查有沒有長交易：

```sql
SELECT pid, now() - xact_start AS age, state, query
FROM pg_stat_activity
WHERE xact_start IS NOT NULL
ORDER BY age DESC LIMIT 5;
```

## 小練習

範例目錄：`playground/backend/l2/2-2-database/database-locks/`。先跑 `make up` 與 `make demo`，看過四個實驗的輸出。

### 動手寫 1：修好額度超扣

`exercise/consume.go` 的 `Consume` 是「先讀再寫」的寫法。請修好它，讓 50 個請求同時扣額度 30 的短網址時，成功次數正好 30，剩餘額度 0。

```bash
make exercise
```

`TestConsume` 通過就完成。起始狀態下，`TestConsume` 與 `TestTransfer` 都會失敗。

<details>
<summary>參考解答</summary>

最小的修法是把 `SELECT` 改成 `SELECT quota FROM links WHERE id = $1 FOR UPDATE`，讓第二個請求在讀的時候就排隊。

更好的做法是整個換掉，用 `UPDATE links SET quota = quota - 1 WHERE id = $1 AND quota > 0`，看影響的列數是否為 0。兩種寫法都能讓測試通過，原子 `UPDATE` 持鎖時間最短。但如果你需要在中間做事（像範例裡的檢查），就只能用 `FOR UPDATE`。

</details>

### 動手寫 2：消滅 deadlock

`exercise/transfer.go` 的 `Transfer` 先扣來源、再加目的地。請修改它，讓 10 個方向相反的呼叫同時執行時，不會有任何一個遇到 `40P01`，而且兩邊額度加起來維持 2000。

`TestTransfer` 通過就完成。

<details>
<summary>參考解答</summary>

在兩個 `UPDATE` 之前，先依 id 由小到大對兩列各執行一次 `SELECT ... FOR UPDATE`。之後不管轉帳方向，所有交易都先鎖小的、再鎖大的，不會再形成環。

```go
first, second := from, to
if first > second {
	first, second = second, first
}
for _, id := range []int64{first, second} {
	if _, err := tx.ExecContext(ctx, `SELECT 1 FROM links WHERE id = $1 FOR UPDATE`, id); err != nil {
		return err
	}
}
```

這和 `deadlock.go` 的 `MoveQuotaOrdered` 是同一個做法。

</details>

### 辨識

下面是 AI 產生的程式，包含「點擊扣額度」與「合併兩個短網址的額度」，資料庫是 PostgreSQL（預設隔離級別）。請做 code review，找出 3 個與本文主題相關的問題。

```go title="handler.go"
// AI 產生：Click 先檢查額度、通過安全檢查，再扣一點
func (s *Server) Click(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	// ...
	var quota int
	var target string
	if err := tx.QueryRowContext(ctx,
		`SELECT quota, url FROM links WHERE id = $1`, id).Scan(&quota, &target); err != nil {
		return err
	}
	if quota <= 0 {
		return ErrNoQuota
	}

	// 向外部的安全瀏覽服務確認網址沒有被列入黑名單。
	resp, err := http.Get("https://safe.example.com/check?u=" + url.QueryEscape(target))
	// ...
	if _, err := tx.ExecContext(ctx,
		`UPDATE links SET quota = $1, clicks = clicks + 1 WHERE id = $2`, quota-1, id); err != nil {
		return err
	}
	return tx.Commit()
}

// 把 src 的額度全部併入 dst，src 歸零。
func (s *Server) Merge(ctx context.Context, src, dst int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	// ...
	var moved int
	if err := tx.QueryRowContext(ctx,
		`SELECT quota FROM links WHERE id = $1 FOR UPDATE`, src).Scan(&moved); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE links SET quota = 0 WHERE id = $1`, src); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE links SET quota = quota + $1 WHERE id = $2`, moved, dst); err != nil {
		return err
	}
	return tx.Commit()
}
```

完整程式碼在 `recognize/handler.go`（AI 產生）。

<details>
<summary>參考解答</summary>

1. **`Click` 的 `SELECT` 沒有加鎖，扣減值是在應用程式算出來的（`quota-1`）**：兩個請求可以同時讀到相同的 `quota`，各自寫回同一個值，造成更新遺失，額度被超扣。改成 `UPDATE links SET quota = quota - 1 ... WHERE id = $1 AND quota > 0` 並看影響的列數；如果真的需要先讀，就用 `FOR UPDATE`。
2. **`Click` 在交易裡呼叫外部 HTTP 服務**：交易與連線在等網路的期間一直被占住。外部服務變慢時，連線池會被耗盡；而且這段等待把讀與寫之間的間隙拉大，競態更容易發生。如果照第 1 點改成 `FOR UPDATE`，這段網路等待就變成鎖持有時間，情況更糟。要先做完外部檢查，再開短交易。
3. **`Merge` 先鎖 `src`、再碰 `dst`，順序取決於呼叫參數**：`Merge(1, 2)` 與 `Merge(2, 1)` 同時執行，就形成環，資料庫會回滾其中一個並回報 `40P01`。應該先把兩個 id 排序，依序用 `FOR UPDATE` 鎖住，再做更新；並且對 `40P01` 做有限次數的重試。

</details>

## 快閃卡

<details>
<summary>PostgreSQL 的 SELECT 一般讀取，會被另一個交易的 UPDATE 擋住嗎？為什麼？</summary>

不會。一般 `SELECT` 不加行鎖，靠 MVCC 讀舊版本，所以讀不擋寫、寫不擋讀。只有寫對寫（以及明確加鎖的讀，例如 `FOR UPDATE`）才會互相等待。

</details>

<details>
<summary>悲觀鎖與樂觀鎖的核心差異是什麼？各自假設了什麼？</summary>

悲觀鎖假設衝突常發生，所以讀的時候就加鎖，別人排隊；樂觀鎖假設衝突少，所以不加鎖，寫入時才檢查資料有沒有被改過，衝突就重試。

</details>

<details>
<summary>交易 A 持有某列的 FOR SHARE，交易 B 想對同一列加 FOR SHARE，會被擋嗎？如果 B 想加 FOR NO KEY UPDATE 呢？</summary>

FOR SHARE 對 FOR SHARE 不衝突，B 能直接取得；FOR NO KEY UPDATE 與 FOR SHARE 衝突，B 要等 A 結束。共享鎖允許多人同時讀，但不允許有人修改。

</details>

<details>
<summary>quota 剩 1，兩個請求先各自 SELECT quota 讀到 1，再各自 UPDATE SET quota = 0，預設的 Read Committed 下會發生什麼事？</summary>

兩個請求都會成功，額度被用了兩次，卻只扣了一次，這是更新遺失。因為一般 SELECT 沒有加鎖，兩個交易能同時讀到同一個舊值。

</details>

<details>
<summary>為什麼 UPDATE links SET quota = quota - 1 WHERE id = 1 AND quota > 0 在高並行下是安全的？</summary>

UPDATE 本身會對命中的列加行鎖，後來的交易要等。等到前一個提交後，Read Committed 下 PostgreSQL 會用最新版本重新檢查 WHERE 條件，所以 quota 是不是大於 0 是用最新的值判斷的。

</details>

<details>
<summary>交易 A 先 UPDATE id=1 再 UPDATE id=2；交易 B 先 UPDATE id=2 再 UPDATE id=1，兩者同時執行。PostgreSQL 會怎麼處理？應用程式會看到什麼？</summary>

兩邊各握一把鎖又等對方的鎖，形成死結。PostgreSQL 在等待超過 deadlock_timeout（預設 1 秒）後偵測到環，中止其中一個交易，應用程式收到 SQLSTATE 40P01，被中止的交易全部回滾。

</details>

<details>
<summary>要怎麼從設計上避免兩個交易搬額度時 deadlock？</summary>

讓所有交易以同樣的順序取得鎖，例如一律依 id 由小到大鎖。這樣不會有人握著一把等另一把，也就不會形成環。仍要對 40P01 做有限次數的重試，因為路徑複雜時順序可能漏掉。

</details>

<details>
<summary>50 個請求搶同一個短網址的額度，用樂觀鎖（重試 3 次）為什麼大部分會失敗？</summary>

樂觀鎖假設衝突少。這裡是熱點，每一輪只有一個請求能成功更新版本，其他的要重讀重試，3 次用完仍衝突就放棄。衝突頻繁時，樂觀鎖把成本從等待變成了白做工與失敗。

</details>

<details>
<summary>在交易裡先 UPDATE，再呼叫一個要 3 秒才回應的外部 API，最後提交。這會造成什麼問題？</summary>

UPDATE 取得的行鎖要持有到交易結束，所以那一列被多鎖 3 秒，其他想改同一列的請求都在排隊，連線也一直被占住。要先做完外部呼叫，再開短交易。

</details>

## 延伸閱讀

- [PostgreSQL：Explicit Locking](https://www.postgresql.org/docs/16/explicit-locking.html)：表鎖、行鎖各模式的衝突表與 deadlock 的官方說明
- [PostgreSQL：SELECT 的 Locking Clause](https://www.postgresql.org/docs/16/sql-select.html#SQL-FOR-UPDATE-SHARE)：`FOR UPDATE`、`NOWAIT`、`SKIP LOCKED` 的語意
- [PostgreSQL：Lock Management 設定](https://www.postgresql.org/docs/16/runtime-config-locks.html)：`deadlock_timeout` 等參數
- [MySQL 8.4：InnoDB Locking](https://dev.mysql.com/doc/refman/8.4/en/innodb-locking.html)：record lock、gap lock 與 next-key lock
- [MySQL 8.4：Deadlocks in InnoDB](https://dev.mysql.com/doc/refman/8.4/en/innodb-deadlocks.html)：InnoDB 怎麼偵測與處理死結，以及如何減少它
