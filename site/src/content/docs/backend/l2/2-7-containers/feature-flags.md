---
title: 功能旗標：部署與發布分離、漸進放量、kill switch
description: 用 Go 寫出穩定分桶的功能旗標，做到先部署、後放量，出事時不重新部署就能關掉，並學會清理過期旗標。
sidebar:
  order: 5
---

## 先想想

短網址服務 v2 把查詢路徑換成新的快取實作，預計週五下午上線。你在預備環境測過，一切正常，但預備環境沒有真實流量。

如果新路徑有 bug，回滾要重跑一次 CI、重建映像、滾動更新，走完整條流水線大約 10 分鐘（示意數字，依你的流水線而定）。這 10 分鐘裡每一位使用者都在踩雷。

有沒有辦法：程式碼先部署上去，但只讓 10% 的使用者走新路徑；出事時 1 秒內關掉，不用重新部署；而且同一位使用者不會這次走新路徑、下次又走舊路徑？

<details>
<summary>提示</summary>

把「程式碼在正式環境裡」和「使用者看得到這個行為」拆成兩件事。第二件事要用什麼東西控制？「同一位使用者」又該用什麼來判斷？

</details>

## 是什麼

**功能旗標（feature flag）** 是程式碼裡的一個條件分支，由執行期可以修改的設定決定走哪一邊，不需要重新部署。

它最接近你在 Twelve-Factor 學到的「設定」，但有兩個差別：旗標通常是暫時的，用完就該刪；旗標的值會依使用者而不同，不是整個環境一個值。

## 為什麼存在

沒有旗標時，「部署」和「發布」是同一個動作：映像一上線，新行為就對所有人生效。這帶來兩個後果。

第一，出錯的影響範圍是 100% 的使用者，退路只有整條流水線重跑。金絲雀部署（上一個項目）能把風險縮到一部分實例，但它切的是 **實例** ：某位使用者這次打到新實例、下次打到舊實例，行為會來回跳，也沒辦法指定「只給內部員工」。

第二，長期分支。新功能做三週，開三週的分支，最後合併一次衝突。主幹式開發（trunk-based development）要求每天合併，未完成的功能就必須能藏在關著的旗標後面。

旗標把「部署」和「發布」拆開：程式碼隨時可以上線，行為由旗標決定何時對誰開啟。

## 怎麼運作

### 部署與發布是兩件事

```text
            部署（deploy）                     發布（release）
  commit ──▶ CI ──▶ 映像 ──▶ 正式環境     旗標 0% ──▶ 10% ──▶ 50% ──▶ 100%
              │                              │         │
      新程式碼在線上，                  只有改旗標設定，
      但旗標關著，沒人走到              不動映像，秒級生效
```

左邊走流水線，慢但可審查；右邊只改設定，快。出事時你要做的是右邊的反方向：把旗標關掉。

### 穩定分桶

放量的核心問題：10% 要怎麼挑？每次請求擲骰子（`rand.Intn(100) < 10`）是錯的，同一位使用者重新整理就可能在兩條路徑之間跳來跳去。

正確做法是對「旗標名稱加使用者 ID」做雜湊，把結果對應到 0–99 的桶，桶號小於放量比例就開啟：

```go title="flags.go"
func (s *Store) Enabled(name, userID string) bool {
	r, ok := s.Snapshot()[name]
	if !ok || !r.Enabled {
		return false
	}
	if slices.Contains(r.AllowUsers, userID) {
		return true
	}
	return bucket(name, userID) < r.Percent
}

func bucket(name, userID string) int {
	h := fnv.New32a()
	h.Write([]byte(name + ":" + userID))
	return int(h.Sum32() % 100)
}
```

這個寫法有三個性質：

- **穩定** ：同一位使用者在同一個旗標下永遠落在同一個桶。
- **單調** ：比例從 10% 調到 20%，原本那 10% 仍然開啟，只會再加入新的人。測試 `TestRampIsMonotonic` 驗證這點。
- **彼此獨立** ：雜湊裡加了旗標名稱當鹽。不加的話，桶號小的那群人會在每個旗標都最先被放量，變成固定的白老鼠。

`Enabled` 還做了兩個保守的選擇：未知的旗標回傳 `false`，因為查不到設定時，走舊行為最安全；`Enabled` 為 `false` 時，連 `allow_users` 也不生效，總開關說關就是關。

### 設定與熱重載

旗標存在一個 JSON 檔：

```json title="flags.json"
{
  "new-lookup-path": { "enabled": true, "percent": 10, "allow_users": ["alice"], "expires": "2026-12-31" }
}
```

程式每秒檢查檔案修改時間，變了就重新載入。重點在 **載入失敗時怎麼辦** ：

```go title="watch.go"
if err := s.Load(data); err != nil {
	slog.Error("flags: invalid config, keep last good config", "err", err)
	return
}
```

`Load` 先解析、驗證（比例在 0–100、日期格式正確），全部通過才用 `atomic.Pointer` 整份換上，讀取端不需要鎖。有任何錯誤就保留上一份可用的設定。

實際跑一次：

```bash
go run ./cmd/demo &
curl -H 'X-User-ID: alice' localhost:8080/abc
curl -H 'X-User-ID: bob' localhost:8080/abc
```

```text
new path: abc
old path: abc
```

`alice` 在 `allow_users` 裡，所以走新路徑；`bob` 要看他的桶號是否小於 10。用 200 個不同使用者實測，有 20 個走新路徑，符合 10%。把 `enabled` 改成 `false` 存檔，約 2 秒內所有人，包含 `alice`，都回到舊路徑。

### 旗標的四種用途

| 類型 | 壽命 | 例子 | 由誰改 |
| --- | --- | --- | --- |
| 發布旗標（release） | 天到週 | 新查詢路徑放量 | 工程師 |
| 實驗旗標（experiment） | 週 | A/B 測試兩種跳轉頁 | 產品與資料 |
| 營運旗標（ops / kill switch） | 長期 | 出事時關掉點擊統計 | 值班工程師 |
| 權限旗標（permission） | 長期 | 付費方案才有自訂網址 | 產品，通常更適合放進權限系統 |

不同類型的壽命差很多，所以要分開管理。本篇主要處理發布旗標與 kill switch。

回到一開始的問題：程式碼先部署，旗標關著；用旗標名稱加使用者 ID 雜湊分桶，從 10% 開始放量，同一位使用者永遠落在同一邊；出事時把 `enabled` 改成 `false`，熱重載讓它在秒級生效，不必重跑流水線。

## 取捨與替代方案

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| 設定檔加熱重載（本篇） | 零依賴、容易理解、可進版本控制 | 多實例要同步檔案；沒有介面、沒有稽核紀錄 | 單一服務、旗標少 |
| 資料庫或 Redis 存旗標 | 即時生效、可做管理介面 | 多一個執行期依賴，讀取要快取 | 多實例、需要營運人員自己改 |
| 旗標服務（Unleash、LaunchDarkly 等）搭配 OpenFeature SDK | 有介面、稽核、分群、實驗統計 | 要維運或付費；SDK 失效時要有預設值 | 團隊大、旗標多 |
| 金絲雀部署 | 不改程式碼；能擋掉啟動失敗與資源問題 | 以實例為單位，無法依使用者分流，無法秒級關掉 | 基礎設施層的風險 |
| 長期分支 | 不用改程式碼結構 | 合併痛苦、整合問題很晚才暴露 | 幾乎沒有好理由 |

旗標增加的是 **程式碼路徑** ：N 個獨立旗標理論上有 2 的 N 次方種組合，你不可能全部測過。這是旗標最大的長期成本。

**什麼時候不該用：**

- 改動很小而且可以輕易回滾，例如修錯字、調整文案。加旗標比改動本身還複雜。
- 會改資料庫結構或資料格式的變更。旗標只能切換程式行為，切不回已經寫進資料庫的新格式，這類變更要用向前向後相容的分階段遷移。
- 安全修補。你要的是所有人立刻套用，不是漸進放量。
- 團隊沒有清理旗標的習慣。這種情況下，旗標只會越積越多。

## 常見坑與故障案例

### 每次請求擲骰子

**症狀** ：使用者回報「我重新整理一下，頁面就變了」，客服無法重現，因為每次結果不同。A/B 實驗的數據也對不起來。

**原因** ：用 `math/rand` 決定要不要開，沒有跟使用者綁定。

**修正** ：像上面的 `bucket` 一樣，用穩定的 ID 雜湊。匿名使用者用 cookie 裡的隨機 ID，不要用 IP。

### 旗標永遠不清理

**症狀** ：程式碼裡有一堆 `if flagEnabled(...)`，沒人記得哪些還在用。某天有人重新利用舊旗標名稱，意外喚醒了一段早就該刪的程式。

**案例** ：2012 年 8 月 1 日，Knight Capital 在部署新的交易程式時，把一個舊旗標拿來重新利用。八台伺服器中有一台沒有拿到新程式，旗標一開，那台伺服器執行了多年前就該淘汰的舊邏輯。SEC 的調查報告指出，公司在約 45 分鐘內損失約 4.6 億美元。部署流程有缺口是直接原因，但「舊程式碼和舊旗標沒刪」讓缺口變成災難。見 [SEC 公告 34-70694](https://www.sec.gov/files/litigation/admin/2013/34-70694.pdf)。

**修正** ：每個旗標登記到期日（本篇的 `expires` 欄位），在 CI 或定期檢查裡列出過期的旗標；放量到 100% 並穩定後，同一個 PR 刪掉旗標與舊路徑。

### 旗標服務掛了就全開

**症狀** ：旗標設定檔壞掉或旗標服務連不上，結果新功能對所有人開啟，把問題放大。

**原因** ：錯誤處理寫成「讀不到就當作 100%」，或每次失敗都重設成預設值。

**修正** ：兩層預設。第一層是載入失敗時保留上一份可用設定（見 `watch.go`）；第二層是完全沒有設定時，未知旗標回傳 `false`，也就是舊行為。預設值要選「出事時最安全」的那邊，而不是「最方便」的那邊。

### 並行讀寫旗標表

**症狀** ：偶爾出現 `fatal error: concurrent map read and map write`，或讀到寫到一半的設定。

**原因** ：用全域 `map` 存旗標，背景 goroutine 重載時直接改它，請求處理則同時讀取。

**修正** ：載入時建一份全新的 `Set`，驗證通過後用 `atomic.Pointer` 整份換上，不要原地修改。測試時加上 `-race` 檢查。

### 示意案例：旗標組合沒測過

**症狀** ：三個旗標各自單獨測試都通過，同時開啟時結帳頁壞掉。

**原因** ：旗標 A 改了價格計算、旗標 B 改了優惠券邏輯，兩者共用同一個欄位。

**修正** ：有相依的旗標合併成一個，或在程式碼裡明確寫出不能同時開啟，並在測試中至少覆蓋「全關」與「預計上線時的組合」。

## 小練習

### 動手寫：找出該清理的旗標

範例目錄：`playground/backend/l2/2-7-containers/feature-flags/`。

先跑 `go test ./...`，確認分桶、放量與 kill switch 的測試通過。接著補完 `exercise/stale.go` 的 `Stale`：給定一個 `flags.Set` 與現在時間，回傳該清理的旗標名稱，依字母排序。到期日早於今天的旗標要列出，沒有 `expires` 的旗標也要列出。到期日剛好是今天的旗標不列。

用下面這行驗證，測試全過就完成了：

```bash
go test -tags exercise ./exercise
```

<details>
<summary>參考解答</summary>

遍歷 `Set`：`Expires` 為空就加入結果；否則用 `time.Parse(time.DateOnly, ...)` 解析，並與 `now.Truncate(24*time.Hour)` 比較，早於才加入。最後用 `slices.Sort` 排序，因為 Go 的 `map` 遍歷順序不固定，不排序的話測試會時過時不過。

</details>

### 辨識

下面是 AI 產生的旗標程式，目標是把新查詢路徑放量給 20% 的使用者，設定檔每秒重新讀取一次。請做 code review，找出 3 個與本文主題相關的問題。

```go title="recognize/flags.go"
// AI 產生：短網址服務 v2 的功能旗標
var flags = map[string]Flag{}

func reload(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("read flags: %v", err)
		flags = map[string]Flag{"new-lookup-path": {Percent: 100}}
		return
	}
	var next map[string]Flag
	if err := json.Unmarshal(data, &next); err != nil {
		log.Printf("parse flags: %v", err)
		return
	}
	flags = next
}

func enabled(name string) bool {
	f, ok := flags[name]
	if !ok {
		return false
	}
	return rand.Intn(100) < f.Percent
}
```

完整程式碼在 `recognize/` 目錄（AI 產生）。

<details>
<summary>參考解答</summary>

1. **`enabled` 用 `rand.Intn(100) < f.Percent`** ：沒有跟使用者綁定，同一個人每次請求可能走不同路徑，而且 `enabled` 連使用者 ID 都沒有接收。改成對旗標名稱加使用者 ID 雜湊分桶。
2. **`reload` 讀檔失敗時把 `new-lookup-path` 設成 `Percent: 100`** ：設定檔暫時讀不到（例如部署時被替換）就把新功能對所有人打開，把一個小問題放大成全面事故。應該保留上一份可用設定，或回到最安全的預設值（關閉）。
3. **`flags` 是全域 `map`，`reload` 在背景 goroutine 直接整個賦值，請求處理同時讀它** ：沒有同步，是資料競爭，`go test -race` 會報。改用 `atomic.Pointer` 持有一份建好、不再修改的 `Set`。

另外，`reload` 沒有驗證 `Percent` 是否在 0–100，也沒有到期日，旗標永遠不會被提醒清理。

</details>

## 快閃卡

<details>
<summary>功能旗標解決的是「部署」還是「發布」的問題？兩者差在哪？</summary>

解決的是把兩者拆開。部署是把程式碼放上正式環境，發布是讓使用者看得到新行為。有旗標後，程式碼可以先上線但關著，之後只改設定就能發布或收回。

</details>

<details>
<summary>金絲雀部署和功能旗標，哪一個可以「只給內部員工」先用新功能？為什麼？</summary>

功能旗標。金絲雀切的是實例，請求打到哪個實例由負載平衡決定，和使用者身分無關；旗標的判斷發生在程式碼裡，可以用使用者 ID 或 allow list 決定。

</details>

<details>
<summary>為什麼放量不能用 rand.Intn(100) &lt; 10？</summary>

因為每次請求都重新擲骰子，同一位使用者會在新舊路徑之間來回跳。要用「旗標名稱加使用者 ID」雜湊出固定的桶號，桶號小於 10 才開啟，這樣同一個人永遠落在同一邊。

</details>

<details>
<summary>旗標從 10% 調到 20%，原本在那 10% 裡的使用者會怎樣？</summary>

仍然開啟。桶號固定，條件從「小於 10」放寬成「小於 20」，原本符合的一定還符合，只會再加入新的人。如果行為會變成「掉出來」，表示分桶不穩定。

</details>

<details>
<summary>旗標設定檔 {"f":{"enabled":true,"percent":150}} 被載入時，程式應該怎麼處理？</summary>

拒絕這份設定並保留上一份可用的設定，同時記錄錯誤。比例超出 0–100，表示設定有誤，不應該猜它的意思，更不該因此全開或全關。

</details>

<details>
<summary>旗標的 enabled 是 false，但 allow_users 裡有 "boss"，boss 看得到新功能嗎？</summary>

看不到。enabled 是 kill switch，為 false 時其他欄位都不生效。否則出事時關掉旗標，名單上的人仍會踩雷，總開關就失去意義。

</details>

<details>
<summary>旗標服務連不上，程式查不到某個旗標，應該回傳 true 還是 false？為什麼？</summary>

回傳 false，也就是走舊行為。舊路徑是已經在正式環境跑過的，出事時最安全；預設成新行為，等於旗標系統一出問題，新功能就對所有人打開。

</details>

<details>
<summary>放量到 100% 並穩定之後，旗標還應該留著嗎？</summary>

不該。留著的旗標是死掉的分支，沒人敢刪，還可能被誤用（Knight Capital 事件就有舊旗標被重複利用）。應該在同一個 PR 刪掉旗標與舊路徑，並用到期日提醒。

</details>

## 延伸閱讀

- [Feature Toggles (aka Feature Flags)](https://martinfowler.com/articles/feature-toggles.html)：Pete Hodgson 在 martinfowler.com 的文章，旗標分類與管理成本講得最完整
- [OpenFeature](https://openfeature.dev/)：旗標評估的廠商中立規格，有 Go SDK，換旗標服務時不用改業務程式碼
- [Unleash 文件](https://docs.getunleash.io/)：開源旗標服務，可以看它怎麼設計放量策略與分群
- [SEC 公告 34-70694](https://www.sec.gov/files/litigation/admin/2013/34-70694.pdf)：Knight Capital 事件的調查報告，部署流程與舊程式碼殘留的細節都在裡面
