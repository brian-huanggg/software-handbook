---
title: MariaDB 標準版與 Galera 叢集
description: 比較 MariaDB 非同步主從與 Galera 多主叢集：資料何時算寫入成功、會不會掉資料、誰負責故障轉移，以及各自的代價。
sidebar:
  order: 6
---

## 先想想

短網址服務 v3 要服務全球使用者，資料庫是一台 MariaDB primary，加一台非同步的 replica。某天凌晨 primary 的主機掛了，值班的人把 replica 提升成新的 primary，服務在五分鐘後恢復。

隔天客服收到回報：使用者在事故前一分鐘建立的幾條短網址，現在點開是 404。資料庫在那一刻明明回了「寫入成功」。

同事提議改成三節點的 Galera 叢集，說「這樣就不會掉資料，也不用有人半夜手動切換」。這些連結為什麼會不見？Galera 真的解掉這個問題，而且沒有任何代價嗎？

<details>
<summary>提示</summary>

想想「資料庫回覆成功」的那一刻，資料已經在幾台機器上了。然後想想：如果每筆寫入都要等其他機器，哪個數字會變差。

</details>

## 是什麼

**Galera 叢集** 是 MariaDB 的多主叢集：每個節點都能寫入，交易在 commit 時把 write-set 送給所有節點驗證，全部通過才算成功。**標準版** 則是單一 `mariadbd`，需要備援時再加上非同步的 binlog 複寫。

這裡有一個容易誤會的地方：兩者不是兩種不同的 image。同一個 `mariadb` image 可以跑成單機、主從，也可以跑成 Galera 節點，差別在設定檔有沒有 `wsrep_on=ON`、`wsrep_provider`、`wsrep_cluster_address` 這幾項，而不在 image 本身。官方 image 通常已附帶 Galera 函式庫，實際以你使用的 tag 為準。

所以比較的其實是兩種複寫方式。RPO（Recovery Point Objective）是故障後最多可以接受丟失多少資料，RPO = 0 代表不能丟任何一筆已確認的寫入。

## 為什麼存在

只有一台資料庫時，那台機器就是單點故障。標準版的解法是加一台 replica：primary 把變更寫進 binlog，replica 在背後追。

這個設計讓 primary 的寫入很快，因為它不等任何人。代價是 primary 回覆成功時，replica 可能還沒收到。以每秒 500 筆寫入、replica 落後 2 秒估算（示意數字），primary 掛掉的瞬間大約有 1,000 筆「已確認」的寫入只存在於那台死掉的機器上。這就是開頭的 404。而且提升 replica 這件事需要有人或有工具去做：偵測、挑選、提升、改連線。

Galera 想解決的就是這兩件事：已確認的寫入不能只存在一台機器上，節點掛掉時也不需要「提升」誰，因為每個節點本來就能寫。它的代價是每次 commit 都要和其他節點溝通一次。

## 怎麼運作

:::note
本篇的範例在 `playground/backend/l3/3-1-distributed-basics/mariadb-galera/`。它是用記憶體模擬資料移動的小程式，不連真的 MariaDB，也不模擬網路與磁碟。設定檔在 `conf/`，沒有實際啟動叢集驗證過。
:::

### 標準版：非同步複寫

```text
client ──► primary ──(1) write + binlog ──► 回覆成功
              │
              └─(2) 稍後 ──► replica  (IO thread ► relay log ► SQL thread)
```

先看 (1) 與 (2) 的順序：回覆成功發生在複寫之前，兩者之間的空隙就是 replica 落後的量。

MariaDB 也有半同步（semi-synchronous）複寫：primary 等到至少一個 replica 收到 binlog 才回覆。但逾時（`rpl_semi_sync_master_timeout`）之後它會退回非同步，所以半同步縮小了丟資料的視窗，並沒有保證 RPO = 0。

故障轉移不屬於 MariaDB 本身，要靠外部：MaxScale 之類的 proxy、Orchestrator、Pacemaker，或人工。下面是範例裡最核心的動作，提升 replica 時，還沒送出的寫入就消失了：

```go title="async.go"
func (c *Cluster) FailOver() (lost int) {
	lost = len(c.pending)
	c.primary = c.replica
	c.replica = map[string]string{}
	c.pending = nil
	return lost
}
```

提升之後還有一個風險：舊 primary 如果復活又接受寫入，就會出現兩個 primary 各寫各的（split-brain）。外部工具要負責把舊的隔離掉（fencing），例如把它設成 `read_only`。

### Galera：certification-based 複寫

Galera 的交易先在本地樂觀執行，不鎖其他節點。到 commit 時才發生複寫：

```text
client ──► db1: 本地執行，commit
           db1 ──廣播 write-set──┬─► db1  certify
                                 ├─► db2  certify ─► 排進佇列 ─► 稍後 apply
                                 └─► db3  certify ─► 排進佇列 ─► 稍後 apply
           全部通過 ──► db1 apply，回覆成功
```

看箭頭：回覆成功之前，write-set 已經到達每個節點並通過驗證；但其他節點「套用到資料表」是之後的事。

驗證（certification）是每個節點用同樣順序、同樣規則做的決定性檢查：這筆交易開始之後，有沒有別的交易已經 commit 過同一個 key？有就失敗，所有節點的結論一致，不需要再投票。

```go title="galera.go"
func Certify(index map[string]uint64, snapshot uint64, writes map[string]string) error {
	for key := range writes {
		if index[key] > snapshot {
			return ErrConflict
		}
	}
	return nil
}
```

`index` 記錄每個 key 最後一次被哪個全域序號（seqno）的交易寫過，`snapshot` 是這筆交易開始時本節點已套用到的 seqno。在 MariaDB 裡，驗證失敗的交易會在 commit 時收到 `ERROR 1213 (40001): Deadlock found when trying to get lock`。

通過之後的流程是：

```go title="galera.go"
	c.seqno++
	e := entry{seqno: c.seqno, writes: tx.writes}
	for key := range tx.writes {
		c.index[key] = e.seqno
	}
	c.log = append(c.log, e)
	// Every live node receives the write-set before the client sees success.
	for _, n := range c.nodes {
		if n.up {
			n.queue = append(n.queue, e)
		}
	}
	// Only the origin applies it now. The others apply asynchronously.
	origin.Apply()
```

這就是 virtually synchronous（虛擬同步）的意思：保證已 commit 的交易一定會被每個節點套用（所以 origin 節點立刻掛掉也不丟，RPO = 0），但不保證每個節點「此刻」都套用完了。因此在別的節點立刻讀，可能讀到舊資料。需要讀到最新時，對該連線設定 `wsrep_sync_wait`（例如 `SET SESSION wsrep_sync_wait=1`），節點會先套用完已驗證的 write-set 再讀。

:::tip
`wsrep_sync_wait` 不用全域開。只有「剛寫完馬上要讀」的路徑需要，其餘讀取維持預設，才不會把延遲加到每個查詢上。
:::

```bash
go run .
```

```text
== standard: async replication ==
replica lag before crash: 3 writes
lost after failover: 3 writes the clients were told succeeded

== galera: certification-based replication ==
plain read on db2 right after commit: found=false
read on db2 with wsrep_sync_wait:     found=true
first writer:  <nil>
second writer: deadlock found when trying to get lock; try restarting transaction | retry it: true
only db1 left: WSREP has not yet prepared node for application use
```

最後一行是 quorum 的效果。

### Quorum 與節點掛掉

**Quorum** 是「超過一半的節點彼此連得上」。只有持有 quorum 的那一邊可以繼續寫入；少數那邊會拒絕服務（`ERROR 1047`），避免兩邊各自寫出不同的歷史。三節點容忍掛 1 台；掛 2 台時剩下的那台只佔 1/3，不能寫。兩節點的叢集掛 1 台就剩 1/2，不過半，所以兩台並不比一台更耐故障。要用兩台機器時，加一個只投票不存資料的 `garbd`（Galera Arbitrator）。

節點掛掉時沒有「提升」的動作，其餘節點繼續服務。但 client 仍要避開死掉的節點，所以前面還是需要 HAProxy、ProxySQL 或 MaxScale 這類 proxy 做健康檢查。節點修好回來時，如果落後的量還在其他節點的 gcache 裡，只補差異（IST，incremental state transfer）；否則整份資料複製（SST，state snapshot transfer，常用 `mariabackup`）。

回到一開始的問題：404 的連結消失，是因為標準版在 replica 收到之前就回覆成功，提升 replica 時那段落後的寫入就沒了。Galera 在回覆成功前，write-set 已到達每個節點，所以不會因為單一節點掛掉而丟資料，也不需要提升誰。但它不是沒有代價：每次 commit 要等所有節點，寫入延遲、衝突處理與節點數量都要你自己負責，下一節量化這些代價。

## 取捨與替代方案

Galera 的 commit 延遲大約等於到最慢節點的來回時間（RTT）。同一個機房約 0.5 ms，跨區若是 70 ms，則單一連線循序提交的上限約 1,000 / 70 ≈ 14 筆交易每秒（估算，忽略處理時間）。標準版的 commit 不受 RTT 影響。

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| 單機 | 最簡單，延遲最低 | 單點故障，沒有備援 | 開發、可容忍停機的內部工具 |
| 標準版 + 非同步 replica | 寫入快；replica 可分擔讀取 | RPO > 0；故障轉移靠外部工具；讀 replica 可能落後 | 讀多寫少，能接受少量資料丟失 |
| 標準版 + 半同步 | 丟資料視窗小 | 仍可能退回非同步；primary 要等 replica | 想降低風險，但不想換架構 |
| Galera 叢集 | 已確認寫入不丟；任一節點可寫；不需提升 | 寫入延遲含 RTT；多寫會衝突；至少 3 節點 | 同機房或低延遲網路，要求 RPO = 0 |

什麼時候不該用 Galera：

- **節點分在高延遲的區域**：每次 commit 都付一次跨區 RTT，吞吐量會被它綁住。跨區備援用非同步 replica 加明確的 RPO 說明，比較誠實。
- **工作負載有大量寫入同一列的熱點**：例如每次點擊都 `UPDATE ... hits = hits + 1`。多節點同時寫會一直衝突，先改成寫入單一節點，或先在別處聚合。
- **只需要擴充讀取**：非同步 replica 就夠，不必為了一致性付寫入延遲。
- **資料表不是 InnoDB，或沒有 primary key**：Galera 只複寫 InnoDB，也建議每張表都有 primary key。

## 常見坑與故障案例

以下案例是示意案例，沒有對應到特定公開事故。

### 兩節點叢集斷線後全部拒寫

症狀：兩台機器間的網路閃斷，兩邊的應用都開始收到 `WSREP has not yet prepared node for application use`。原因：兩節點時，任何一邊都只佔一半，沒有人過半，所以兩邊都停寫。避免：用三個（或奇數個）節點，或加 `garbd` 當仲裁。

### 多寫同一列造成 deadlock

症狀：流量上來後，應用 log 零星出現 `ERROR 1213`，但沒有人手動鎖過任何東西。原因：兩個節點同時修改同一列，其中一個在 certification 失敗；錯誤是在 commit 時才回來。避免：應用層要檢查 commit 的錯誤並整筆交易重試；熱點列改成單一寫入節點，或先聚合再寫。

### 剛寫完立刻讀不到

症狀：使用者建立短網址後，導向頁偶爾 404，重新整理就好了。原因：寫入落在 db1，proxy 把下一個讀取送到 db2，db2 還沒套用。避免：需要讀到自己剛寫的資料時，對該連線設 `wsrep_sync_wait`，或讓同一個 session 固定走同一個節點。

### 整個叢集關機後起不來

症狀：三台全部停機後重啟，節點拒絕啟動，訊息指向 `grastate.dat` 的 `safe_to_bootstrap: 0`。原因：Galera 不知道哪一台有最新的資料，不讓你隨便挑一台當起點。避免：先查各節點最後的 seqno（`mysqld --wsrep-recover`），從最新的那台引導（`galera_new_cluster`），其餘再依序加入。

:::danger
從落後的節點強制引導叢集，會讓較新的交易永遠消失。引導前先確認每一台的 seqno。
:::

## 小練習

### 動手寫

範例裡的 `galera.Certify` 是 Galera 防止「讀到舊資料還寫」的核心。請補完 `exercise/certify.go` 的 `Certify`：當 `writes` 中任何一個 key 在 `snapshot` 之後被別的交易 commit 過，回傳 `ErrConflict`。

起始程式碼在 `playground/backend/l3/3-1-distributed-basics/mariadb-galera/exercise/`。做對時，下面的指令會通過：

```bash
go test -tags exercise ./exercise/
```

<details>
<summary>參考解答</summary>

```go title="galera.go"
func Certify(index map[string]uint64, snapshot uint64, writes map[string]string) error {
	for key := range writes {
		if index[key] > snapshot {
			return ErrConflict
		}
	}
	return nil
}
```

關鍵是用「大於」比較：`index[key] == snapshot` 代表那次寫入已經在這筆交易的快照裡，不算衝突。沒出現過的 key 在 `index` 裡是 0，永遠不大於 `snapshot`，所以不會誤判。

</details>

### 辨識

下面是 AI 產生的短網址 `Store`，連線指向三個 Galera 節點前面的 load balancer。它能編譯、單機測試會過。請做 code review，找出會在叢集上出問題的地方（至少兩個）。

```go title="store.go"
func (s *Store) Create(ctx context.Context, code, url string) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO links (code, url) VALUES (?, ?)", code, url)
	return err
}

// Resolve is called right after Create to render the "your link" page.
func (s *Store) Resolve(ctx context.Context, code string) (string, error) {
	var url string
	err := s.db.QueryRowContext(ctx,
		"SELECT url FROM links WHERE code = ?", code).Scan(&url)
	return url, err
}

func (s *Store) Hit(ctx context.Context, code string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		"UPDATE links SET hits = hits + 1 WHERE code = ?", code); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO click_log (code, at) VALUES (?, NOW())", code); err != nil {
		return err
	}
	tx.Commit()
	return nil
}
```

完整檔案（含資料表定義）在 `recognize/store.go`，`go test ./...` 不會掃到它。

<details>
<summary>參考解答</summary>

1. **`Create` 之後馬上 `Resolve`，沒有處理讀取一致性**：連線經過 load balancer，`Resolve` 可能落在另一個還沒套用 write-set 的節點，偶爾回 404。正確寫法：取一條 `db.Conn`，先 `SET SESSION wsrep_sync_wait=1` 再查，或讓這兩步走同一個節點。
2. **`Hit` 忽略 `tx.Commit()` 的回傳值**：Galera 的 certification 衝突是在 commit 時回報（`ERROR 1213`），這裡直接丟掉，點擊數悄悄少算。另外 `UPDATE links SET hits = hits + 1` 是多節點寫同一列的熱點，衝突機率高，卻沒有重試。正確寫法：檢查 `Commit` 的錯誤，遇到 1213 就整筆交易重試（加退避），熱點計數改成單一寫入節點或先聚合。
3. **`click_log` 用 MyISAM 且沒有 primary key**：MyISAM 預設不會被 Galera 複寫（`wsrep_replicate_myisam` 預設關閉），資料只留在處理該次點擊的那個節點，其他節點看不到。沒有 primary key 的表也不建議用在 Galera。正確寫法：`ENGINE=InnoDB` 並加上 primary key（例如自增的 `id`）。

</details>

## 快閃卡

<details>
<summary>virtually synchronous 保證什麼、不保證什麼？</summary>

保證已 commit 的交易一定會被每個節點套用，所以不丟資料。不保證每個節點此刻都已套用完，因此在其他節點立刻讀可能是舊資料。

</details>

<details>
<summary>標準版與 Galera 各在什麼時間點回覆「寫入成功」？</summary>

標準版在 primary 自己寫完就回，replica 稍後才追。Galera 在 write-set 已送達所有節點並通過 certification 後才回，其他節點的套用則在之後。

</details>

<details>
<summary>replica 落後 3 筆時 primary 掛了，你提升 replica。那 3 筆寫入呢？</summary>

消失。這 3 筆只存在於已死的 primary 上，客戶端卻已收到成功。這就是非同步複寫 RPO &gt; 0 的來源。

</details>

<details>
<summary>三節點 Galera 同時掛掉 db2、db3，剩下的 db1 還能接受寫入嗎？</summary>

不能。db1 只佔 1/3，沒有過半，不持有 quorum，會回 <code>ERROR 1047</code> 拒絕寫入，避免和另一邊產生不同的歷史。

</details>

<details>
<summary>在 db1 commit 成功後，立刻在 db2 執行 SELECT，一定看得到嗎？</summary>

不一定。db2 可能還沒套用那筆 write-set。需要時對該連線設 <code>wsrep_sync_wait</code>，db2 會先套用完再讀。

</details>

<details>
<summary>為什麼兩節點的 Galera 叢集掛 1 台就不能寫？</summary>

剩下 1 台只佔一半，不是過半，沒有 quorum。兩個節點斷線時兩邊都這樣，所以兩邊都停寫，沒有比單機更耐故障。

</details>

<details>
<summary>應用 log 偶爾出現 ERROR 1213，但沒人手動鎖表。可能的原因是什麼？</summary>

多個節點同時寫同一列，其中一筆在 commit 時 certification 失敗。處理方式是整筆交易重試，或把熱點寫入集中到單一節點。

</details>

<details>
<summary>半同步複寫為什麼仍然不是 RPO = 0？</summary>

等不到 replica 確認超過 <code>rpl_semi_sync_master_timeout</code> 時，primary 會退回非同步繼續回覆成功，這段時間的寫入就沒有第二份。

</details>

<details>
<summary>節點重新加入叢集時，IST 與 SST 差在哪？</summary>

IST 只補節點缺少的 write-set，前提是其他節點的 gcache 還留著；SST 是整份資料複製，較慢，gcache 不夠時才用。

</details>

## 延伸閱讀

- [What is MariaDB Galera Cluster?](https://mariadb.com/kb/en/what-is-mariadb-galera-cluster/)：官方對 Galera 的總覽與限制
- [Galera Cluster Documentation](https://galeracluster.com/library/documentation/)：Galera 的完整手冊，含 certification 與 quorum 的細節
- [Replication Overview](https://mariadb.com/kb/en/replication-overview/)：MariaDB 標準版主從複寫的總覽
- [Semisynchronous Replication](https://mariadb.com/kb/en/semisynchronous-replication/)：半同步的設定與逾時退回行為
- [Galera Cluster System Variables](https://mariadb.com/kb/en/galera-cluster-system-variables/)：`wsrep_sync_wait` 等變數的說明
