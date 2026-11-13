---
title: Testcontainers for Go：真資料庫測試
description: 在 Go 測試裡啟動真的 PostgreSQL 與 Redis，測出 fake 與 mock 永遠測不到的並行、約束與逾時行為。
sidebar:
  order: 2
---

## 先想想

短網址服務 v2 的「建立短網址」原本這樣寫：先 `SELECT` 看這個 code 有沒有人用，沒有再 `INSERT`。你為它寫了單元測試，依賴的儲存層換成記憶體裡的 fake（一個 `map` 加一把鎖），測試全綠，覆蓋率也很漂亮。

上線第一週，客服收到回報：兩個人幾乎同時建立同一個自訂 code，兩邊都顯示成功，後建立的那筆蓋掉了前一筆。

為什麼全綠的單元測試沒擋下這個 bug？要抓到它，測試至少得具備什麼，是 fake 給不了的？

<details>
<summary>提示</summary>

想想「兩個請求同時進來」時，誰來決定誰先誰後：是你寫的 fake，還是資料庫本身？

</details>

## 是什麼

**Testcontainers** 是一套函式庫，讓測試程式在執行時透過 Docker 啟動一次性的真實服務（PostgreSQL、Redis、Kafka……），測完自動銷毀。

它屬於整合測試的工具，不是測試框架：斷言、子測試、平行執行仍然由 Go 的 `testing` 負責。Testcontainers 只回答一個問題：「測試需要的真實依賴，從哪裡來？」答案是：從你的測試程式碼裡來，不是從某台共用的測試伺服器來。

## 為什麼存在

整合測試要接真的資料庫，但「真的資料庫」該放哪裡，傳統做法都有痛點：

- **共用測試資料庫**：兩個人同時跑測試就互相踩資料；誰改了 schema 沒通知，別人的測試就紅。紅燈的原因和你的程式碼無關。
- **開發者各自在本機裝**：版本不同（你是 PostgreSQL 15，CI 是 18），行為就可能不同。新人第一天要花半天裝環境。
- **用 SQLite 或 fake 代替**：快，但行為不是同一套。`ON CONFLICT`、`SERIAL`、交易隔離層級、錯誤碼，每一項都可能不一樣。你測的是 fake，不是服務真正會碰到的資料庫。
- **docker compose 事先起好**：比前面好，但「先記得起環境」是人工步驟，CI 與本機設定又各寫一份。

Testcontainers 把「起環境」變成測試程式碼的一部分：`go test` 一個指令，映像版本寫死在程式碼裡，本機與 CI 跑的是同一個東西。成本是時間：在本機實測（OrbStack、映像已在本機），PostgreSQL 與 Redis 各約 2 秒就緒；第一次要下載映像，整個套件跑了約 11 秒。這些數字因機器與網路而異，量一次自己的環境再決定要不要共用容器。

## 怎麼運作

範例在 `playground/testing/l2/2-2-integration-testing/testcontainers-go/`。受測的是短網址服務 v2 的一小塊：連結存在 PostgreSQL，熱門連結的查詢結果快取在 Redis。你不需要讀過後端 L2：只要知道「先查 Redis，沒有再查 PostgreSQL，並把結果寫回 Redis」這個流程就夠了。

```text
go test
  │
  ├─ TestMain ── 啟動 ──▶ [ryuk]      監看測試程序，程序結束就清掉容器
  │             ── 啟動 ──▶ [postgres:18]  隨機 host port ─┐
  │             ── 啟動 ──▶ [redis:8]      隨機 host port ─┤
  │                                                        │
  │             等待就緒 ◀─ 等待策略（log、port、SQL 連線）┘
  │
  ├─ 測試 A ── 清空資料 ─▶ 真的 PostgreSQL / Redis
  ├─ 測試 B ── 清空資料 ─▶ 同一組容器
  └─ m.Run() 結束 ── Terminate 兩個容器
```

圖的重點：容器只在 `TestMain` 啟動一次，所有測試共用；每個測試開始前自己清資料。

### 啟動容器：用 module 而不是自己拼

Testcontainers 為常見服務提供現成的 module，已經包好預設環境變數與等待條件：

```go title="store_test.go"
pg, err := tcpostgres.Run(ctx, "postgres:18",
	tcpostgres.WithDatabase("shortener"),
	tcpostgres.WithUsername("test"),
	tcpostgres.WithPassword("test"),
	tcpostgres.BasicWaitStrategies(),
)
if err != nil {
	log.Printf("start postgres: %v", err)
	return 1
}
defer pg.Terminate(ctx) //nolint:errcheck // best effort; Ryuk reaps leftovers
```

映像標籤一定寫明版本（`postgres:18`），不用 `latest`：測試環境要和正式環境的主版本一致，也要讓昨天綠的測試今天不會因為映像更新而變紅。

容器的 port 不要自己指定。Testcontainers 預設把容器 port 映射到 host 上的隨機 port，再由 `ConnectionString` 這類方法告訴你實際位置：

```go title="store_test.go"
dsn, err := pg.ConnectionString(ctx)
// ...
testDB, err = sql.Open("pgx", dsn)
```

### 等待就緒：等條件，不是等時間

「容器已啟動」不等於「服務可以連線」。PostgreSQL 初始化資料庫時會短暫重啟，這期間連線會被拒絕。Testcontainers 用**等待策略（wait strategy）** 處理：持續檢查某個條件，成立才讓 `Run` 回傳。

`BasicWaitStrategies()` 等 PostgreSQL 的 log 出現「ready to accept connections」兩次，並確認 port 可連。沒有現成 module 設定時，自己指定：

```go title="store_test.go"
rd, err := tcredis.Run(ctx, "redis:8",
	testcontainers.WithWaitStrategy(wait.ForLog("Ready to accept connections")),
)
```

常用的策略是 `wait.ForLog`、`wait.ForListeningPort`、`wait.ForSQL`、`wait.ForHTTP`。選的標準是：這個條件成立時，你的測試真的能連上。

### 共用容器，隔離資料

每個測試各起一組容器最乾淨，但每組約 2 秒；二十個測試就是 40 秒。常見做法是 `TestMain` 起一組、所有測試共用，用「每個測試開始前清資料」換回隔離：

```go title="store_test.go"
func newStore(t *testing.T, ttl time.Duration) *store.Store {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test: needs Docker")
	}
	ctx := context.Background()
	if _, err := testDB.ExecContext(ctx, `TRUNCATE links`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if err := testRDB.FlushAll(ctx).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}
	return store.New(testDB, testRDB, ttl)
}
```

清資料放在測試**開始前**而不是結束後：上一個測試如果中途失敗，結束後的清理可能根本沒跑到。

這個做法的前提是同一個套件內的測試不平行執行（不呼叫 `t.Parallel()`）。要平行，就改成每個測試用自己的 schema 或資料庫，細節在本領域 3.1 測試資料管理。

`testing.Short()` 是 Go 內建的慢測試開關：`go test -short ./...` 會跳過這些測試，讓開發時的快速迴圈不依賴 Docker；完整的 `go test ./...` 則包含它們。

### 測到 fake 測不到的東西

有了真的資料庫，「先想想」那個 bug 就能寫成測試：二十個 goroutine 同時建立同一個 code，只能有一個成功。

```go title="store_test.go"
const writers = 20
var wins, taken int
// ...
for range writers {
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := s.Create(ctx, "race", "https://example.com/r")
		// ... count nil as a win, ErrCodeTaken as taken
	}()
}
wg.Wait()
if wins != 1 || taken != writers-1 {
	t.Fatalf("wins=%d taken=%d, want 1 and %d", wins, taken, writers-1)
}
```

`Create` 依賴的是資料庫主鍵：`INSERT` 違反唯一約束時，PostgreSQL 回傳 SQLSTATE `23505`，程式把它轉成 `ErrCodeTaken`。這整條路徑（約束、錯誤碼、驅動怎麼包裝錯誤）都在資料庫與驅動裡，fake 不會有。

同樣的道理，快取的 TTL 也用真的 Redis 驗證：把 TTL 設成 200 毫秒，等 400 毫秒後確認快取真的過期，不需要自己假造時鐘。

回到一開始的問題：fake 全綠，是因為 fake 的行為由寫它的人決定：一把鎖讓寫入排隊，根本不會出現「兩個連線同時 `INSERT`」。要抓到這個 bug，測試必須讓多個連線同時打真的資料庫，由資料庫的約束來裁判。Testcontainers 讓這種測試能在幾秒內、不依賴任何共用環境就跑起來。

### CI 與清理：Ryuk

Testcontainers 啟動容器時，順帶會起一個叫 **Ryuk** 的小容器（`testcontainers/ryuk`），它守著你的測試程序：程序結束，不論是正常結束還是被 `kill -9`，Ryuk 都會把這次啟動的容器清掉。程式裡的 `Terminate` 是正常路徑上的整潔，Ryuk 是異常路徑上的保險。

在 CI 上，要確認 runner 能存取 Docker daemon：GitHub Actions 的 Linux runner 內建 Docker；自架 runner 需要自己確認 socket 可用。

## 取捨與替代方案

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| fake／記憶體實作 | 微秒級、零依賴 | 行為由你定義，測不到約束、並行、交易、錯誤碼 | 單元測試，驗證業務邏輯 |
| SQLite 或內嵌資料庫 | 快、不用 Docker | SQL 方言與行為和正式資料庫不同 | 正式環境本身就是 SQLite |
| 共用測試資料庫 | 沒有啟動成本 | 互相踩資料、schema 漂移、紅燈原因不明 | 預備環境（staging）的端對端驗證 |
| docker compose 先起好 | 與正式拓樸接近、可以起很多服務 | 要記得先起；本機與 CI 各維護一份 | 多服務一起測、手動探索 |
| Testcontainers | 測試自己帶環境、版本寫在程式碼裡、本機與 CI 一致 | 啟動數秒；需要 Docker；平行與清理要自己設計 | 驗證資料庫、快取、訊息佇列的真實行為 |

什麼時候不該用：

- **測的是純業務邏輯。** 計算短網址的過期時間、驗證 URL 格式，這些用單元測試更快、更穩。為了它起容器，只是把 2 秒的成本浪費在沒有依賴真實服務的事情上。
- **執行環境沒有 Docker，也不打算有。** 某些受限的 CI、沙箱環境不允許存取 Docker daemon。這時要嘛改用 compose 在別處起服務，要嘛接受這層測試跑在另一條流水線。
- **要驗證的是正式環境獨有的行為。** 叢集的故障轉移、雲端託管資料庫的擴充套件，容器裡的單機版本不等於它。這類風險要在預備環境或正式環境驗證。

## 常見坑與故障案例

### 固定 host port：單獨跑會過，一起跑就紅

**症狀**：`go test ./store` 通過，`go test ./...` 隨機失敗，訊息像「port is already allocated」或連到別人的資料庫。

**原因**：用 `"5432:5432"` 把容器 port 綁到 host 的固定 port。第二個測試或第二個套件啟動時，5432 已被前一個佔用；更糟的是 host 上本來就有一個 PostgreSQL 在 5432，測試默默連到它，把你本機的資料清空。

**怎麼避免**：不指定 host port，讓 Testcontainers 隨機映射，再用 `ConnectionString` 或 `MappedPort` 取得實際位置。

（示意案例：這類「本機跑得過、CI 一起跑才紅」的失敗，常出現在平行跑多個套件的流水線上。）

### 用 `time.Sleep` 當等待策略

**症狀**：本機通過，CI 偶爾出現 `connection refused` 或 `the database system is starting up`；加長 sleep 後好一陣子，換個較慢的 runner 又發作。

**原因**：sleep 是猜一個時間。機器忙、映像剛下載、磁碟慢，都會讓就緒時間變長；猜得保守，每次測試又白等。

**怎麼避免**：用等待策略等明確條件（log、port、SQL 連線）。要等的是「可以連上」這件事，不是三秒。

### 容器洩漏

**症狀**：跑了幾天測試後，`docker ps` 一堆 `postgres:18` 容器，Docker Desktop 吃掉十幾 GB 記憶體，新測試開始起不來。

**原因**：啟動後沒有 `Terminate`，測試失敗時 `t.Fatal` 又提早離開，沒有任何清理。Ryuk 能在程序結束後補刀，但如果環境停用了它（設了 `TESTCONTAINERS_RYUK_DISABLED=true`），就沒有人收尾。

**怎麼避免**：啟動成功後立刻 `defer Terminate`（或在測試內用 `testcontainers.CleanupContainer(t, c)`）；除非 CI 環境明確不支援，不要關閉 Ryuk。

### `TestMain` 裡先用 `testing.Short()`，直接 panic

**症狀**：

```text
panic: testing: Short called before Parse
```

**原因**：在 `TestMain` 裡，測試旗標還沒解析。`testing.Short()` 讀的是 `-short` 這個旗標，必須先呼叫 `flag.Parse()`。這是我們寫這份範例時實際撞到的錯誤。

**怎麼避免**：`TestMain` 開頭先 `flag.Parse()`：

```go title="store_test.go"
func TestMain(m *testing.M) {
	flag.Parse() // testing.Short() panics if called before the flags are parsed
	os.Exit(run(m))
}
```

另外把實際邏輯放進 `run(m) int`，讓 `defer Terminate` 能在 `os.Exit` 之前執行：`os.Exit` 會直接結束程序，不執行 deferred 函式。

### 本機通過，CI 找不到 Docker

**症狀**：CI 上所有整合測試失敗，訊息像 `Cannot connect to the Docker daemon`。

**原因**：CI runner 沒有 Docker，或測試本身跑在容器裡，卻沒把 Docker socket 交給它。本機的 Docker 是你自己裝好的，CI 不是。

**怎麼避免**：在流水線設計階段就確認 runner 能存取 Docker daemon；在 macOS 上用 OrbStack、Colima 等替代品時，可能要設定 `DOCKER_HOST` 讓 Testcontainers 找到 socket。把這件事寫進專案 README，新人與 CI 都照做。

（示意案例：這個失敗模式很常見，但這裡不是引自特定的公開事故報告。）

## 小練習

### 動手寫：刪除連結時，快取也要失效

在範例的 `Store` 加一個 `Delete(ctx, code)`：刪除資料庫的列，同時讓 Redis 裡的快取失效。

起始程式碼在 `playground/testing/l2/2-2-integration-testing/testcontainers-go/`，直接在 `store/store.go` 與 `store/store_test.go` 上加。目標：

1. 寫一個 `TestDeleteInvalidatesCache`：建立連結、`Resolve` 一次讓快取暖起來、`Delete`，再 `Resolve` 應該得到 `ErrNotFound`。
2. 驗證做對了：`go test ./...` 通過；然後把 `Delete` 裡清快取的那一行註解掉，測試應該紅燈。如果拿掉那行測試還是綠的，代表測試沒有真的測到快取。

<details>
<summary>參考解答</summary>

`Delete` 要兩件事都做：

```go title="store.go"
func (s *Store) Delete(ctx context.Context, code string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM links WHERE code = $1`, code)
	if err != nil {
		return fmt.Errorf("delete link: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("rows affected: %w", err)
	} else if n == 0 {
		return ErrNotFound
	}
	if err := s.rdb.Del(ctx, cacheKey(code)).Err(); err != nil {
		return fmt.Errorf("cache del: %w", err)
	}
	return nil
}
```

測試關鍵是「先暖快取」：

```go title="store_test.go"
func TestDeleteInvalidatesCache(t *testing.T) {
	s := newStore(t, time.Minute)
	ctx := context.Background()
	if err := s.Create(ctx, "gone", "https://example.com/gone"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Resolve(ctx, "gone"); err != nil { // warm the cache
		t.Fatal(err)
	}

	if err := s.Delete(ctx, "gone"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Resolve(ctx, "gone"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Resolve after Delete = %v, want ErrNotFound", err)
	}
}
```

少了「暖快取」這一步，測試在沒清快取的 `Delete` 上也會通過：快取本來就是空的。拿掉 `rdb.Del` 後，`Resolve` 會從 Redis 回傳已刪除的連結，測試因此紅燈（`Resolve after Delete = <nil>, want ErrNotFound`）。

</details>

### 辨識

以下是 AI 產生的整合測試輔助函式與兩個測試。兩個測試單獨跑都會通過。請做 code review，找出會在實際使用時造成問題的地方。

```go title="links_test.go"
func setupDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:18",
			ExposedPorts: []string{"5432:5432"},
			Env: map[string]string{
				"POSTGRES_USER":     "test",
				"POSTGRES_PASSWORD": "test",
				"POSTGRES_DB":       "shortener",
			},
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start container: %v", err)
	}

	time.Sleep(3 * time.Second)

	host, err := c.Host(ctx)
	if err != nil {
		t.Fatalf("host: %v", err)
	}
	db, err := sql.Open("pgx", "postgres://test:test@"+host+":5432/shortener?sslmode=disable")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
```

完整檔案在 `recognize/links_test.go`（用 `-tags recognize` 才會編譯，不影響 `go test ./...`），裡面的 `TestInsertLink` 與 `TestInsertDuplicateLink` 各自呼叫一次 `setupDB(t)`。

<details>
<summary>參考解答</summary>

有三個問題：

1. **固定 host port（`ExposedPorts: []string{"5432:5432"}`）。** 兩個測試各呼叫一次 `setupDB`，第二次啟動時 5432 還被第一個容器佔著，所以單獨跑都過，一起跑第二個測試就失敗（port 已被配置）。host 上本來有 PostgreSQL 時，更會連到錯的資料庫。改法：只寫容器 port（`"5432/tcp"`），啟動後用 `c.MappedPort(ctx, "5432/tcp")` 取得 host 上的實際 port；或改用 `tcpostgres.Run` 搭配 `ConnectionString`。
2. **用 `time.Sleep(3 * time.Second)` 等就緒。** 在慢的機器上 3 秒不夠，會間歇性出現連線被拒；在快的機器上則每個測試白等 3 秒。改法：加等待策略，例如 `WaitingFor: wait.ForListeningPort("5432/tcp")`，或直接用 `tcpostgres.BasicWaitStrategies()`。
3. **沒有 `Terminate` 容器。** 只有 `db.Close()` 被清理，容器沒有。每次跑測試都會留下一個容器，直到程序結束後 Ryuk 才補刀（若 Ryuk 被停用就永遠留著）。改法：啟動成功後立刻 `testcontainers.CleanupContainer(t, c)` 或 `t.Cleanup(func() { c.Terminate(ctx) })`。

另外值得一提的是，這份程式為每個測試各起一組容器；測試變多時總時間會線性成長，可以改成 `TestMain` 起一組、測試之間清資料。但這是效能取捨，不是錯誤。

</details>

## 快閃卡

<details>
<summary>Testcontainers 解決的是整合測試的哪個問題？它取代 Go 的 <code>testing</code> 嗎？</summary>

它解決「測試需要的真實依賴從哪裡來」：在測試程式裡用 Docker 起一次性的服務。它不取代 `testing`，斷言與執行仍由 `testing` 負責。

</details>

<details>
<summary>為什麼用 map 加鎖寫的 fake，測不出兩個請求同時建立同一個 code 的 bug？</summary>

因為 fake 的行為由寫它的人決定：鎖讓寫入排隊，不會出現兩個連線同時 `INSERT`，也沒有資料庫的唯一約束與 SQLSTATE `23505`。這條路徑只存在於真的資料庫與驅動裡。

</details>

<details>
<summary>測試程式碼寫 <code>postgres:latest</code>，今天綠、明天無故變紅，最可能的原因是什麼？</summary>

`latest` 會隨映像更新而指向新版本，資料庫行為或預設設定改變就會讓測試變紅。標籤應該寫明版本（如 <code>postgres:18</code>），並與正式環境的主版本一致。

</details>

<details>
<summary>測試裡寫 <code>ExposedPorts: []string{"5432:5432"}</code>，單獨跑過、一起跑第二個失敗，為什麼？</summary>

這把容器綁到 host 的固定 port 5432。第一個容器還佔著 5432，第二個就會啟動失敗。改成只寫容器 port，讓 Testcontainers 隨機映射 host port。

</details>

<details>
<summary>容器顯示「已啟動」，為什麼連線仍可能被拒絕？該怎麼處理？</summary>

「已啟動」只代表程序在跑，服務可能還在初始化（PostgreSQL 初始化時會短暫重啟）。用等待策略等明確條件（log、port、SQL 連線），不要用 <code>time.Sleep</code> 猜時間。

</details>

<details>
<summary>20 個整合測試，每個測試各起一組 PostgreSQL（約 2 秒），總共大約多久？怎麼縮短？</summary>

約 40 秒（20 × 2 秒）。改成在 <code>TestMain</code> 起一組容器共用，並在每個測試開始前用 <code>TRUNCATE</code> 清資料來維持隔離，啟動成本只付一次。

</details>

<details>
<summary>清資料為什麼放在測試開始前，而不是結束後？</summary>

上一個測試若中途失敗或 panic，結束後的清理可能沒跑到，髒資料會留給下一個測試。在開始前清，每個測試都保證從乾淨狀態出發。

</details>

<details>
<summary>測試程式被 <code>kill -9</code> 殺掉，容器會一直留著嗎？</summary>

通常不會。Testcontainers 啟動時順帶起了 Ryuk 容器，監看測試程序，程序結束（含被強制終止）後會清掉這次啟動的容器。例外是環境停用了 Ryuk（<code>TESTCONTAINERS_RYUK_DISABLED=true</code>）。

</details>

<details>
<summary>在 <code>TestMain</code> 一開頭呼叫 <code>testing.Short()</code>，出現 <code>Short called before Parse</code>，原因與修法是什麼？</summary>

`testing.Short()` 讀 `-short` 旗標，但 `TestMain` 執行時旗標還沒解析。在呼叫前先 `flag.Parse()`。

</details>

## 延伸閱讀

- [Testcontainers for Go 官方文件](https://golang.testcontainers.org/)：啟動容器、等待策略、各種 module 的用法都在這裡
- [Testcontainers for Go：PostgreSQL module](https://golang.testcontainers.org/modules/postgres/)：本文用的 `tcpostgres.Run` 與可用的選項
- [testcontainers-go 原始碼](https://github.com/testcontainers/testcontainers-go)：想知道行為細節時，讀 module 的實作最準
- [moby-ryuk 專案](https://github.com/testcontainers/moby-ryuk)：Ryuk 清理容器的運作方式
- [Go `testing` 套件：Main](https://pkg.go.dev/testing#hdr-Main)：`TestMain` 的官方說明與 `os.Exit` 的注意事項
- [PostgreSQL 錯誤碼附錄](https://www.postgresql.org/docs/current/errcodes-appendix.html)：`23505 unique_violation` 等 SQLSTATE 的完整清單
