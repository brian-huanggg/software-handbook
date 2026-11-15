---
title: 跨系統一致性
description: 在資料庫與物件儲存之間，用不可變 key、同交易待刪除表與冪等清除，換圖與刪檔時不留懸空紀錄。
sidebar:
  order: 8
---

## 先想想

短網址服務 v4 的使用者可以換掉自己的頭像。後端的做法很直覺：把新圖上傳到物件儲存，用新的 key 更新資料庫裡的 `avatar_key`，最後刪掉舊圖。三個步驟，一步接一步。

現在假設程序可能在任何兩步之間當機，而且物件儲存和資料庫是兩個互不知道對方的系統，沒有共同的交易可以包住它們。

三個步驟各自當機時，使用者會看到什麼？有沒有一種順序與做法，讓每一個當機點留下的都是「可以事後收拾」的殘骸，而不是「使用者看得到的錯誤」？

<details>
<summary>提示</summary>

先分辨兩種殘骸：資料庫指向一個不存在的物件，和物件存在但沒有任何紀錄指向它。哪一種使用者會立刻看到？

</details>

## 是什麼

**跨系統一致性（cross-system consistency）** 是指：當一個操作要同時改動兩個沒有共同交易的系統（這裡是資料庫與物件儲存），無論程序在哪一步中斷，兩邊都不會停在「紀錄指向不存在的物件」的狀態，而且多出來的物件最終都會被回收。

它和資料庫內部的 ACID 交易是兩回事。交易靠資料庫自己的日誌讓多個寫入一起生效或一起失效；跨系統沒有這個機制，只能靠 **操作的順序** 與 **可重試的清除** 來逼近同樣的效果。上一篇 [資料與檔案同一個請求](/backend/l4/4-1-file-upload/upload-with-data/) 已經用「先檔案、後紀錄」處理了新增；本篇處理更難的情況：替換與刪除，也就是同時有「新的」與「舊的」要處理。

本篇用物件儲存的最小抽象（`Put`、`Delete`、`List`），範例用本機目錄模擬，資料庫用記憶體模擬交易。真正的 MinIO 與 S3 細節留給「物件儲存實務」。

## 為什麼存在

沒有任何設計時，替換頭像的程式碼通常是：

```text
1. 把舊檔刪掉
2. 把新檔寫進去
3. 更新資料庫
```

或是把 2、3 對調。不管怎麼排，只要少了一個明確的原則，每個當機點都會留下不同的爛攤子：

| 中斷在 | 做法 A（先刪舊、再寫新、再更新） | 做法 B（先更新、再寫新） |
| --- | --- | --- |
| 第 1 步之後 | 舊檔沒了，紀錄還指著它：**破圖** | 紀錄指向還不存在的檔：**破圖** |
| 第 2 步之後 | 紀錄指向舊 key，舊檔已刪：**破圖** | 紀錄對了，舊檔留著：孤兒 |

兩種做法都有「破圖」的當機點。更糟的是，破圖是 **資料錯誤**，使用者立刻看到，而且沒有任何掃描能靠檔案系統發現它：檔案根本不存在，沒有東西可以掃。

另一種殘骸，孤兒檔案，只是多花一點儲存費。以常見的物件儲存價格估算（約每 GB 每月 0.02 到 0.03 美元，各家與各區不同），一萬個被遺棄的 2 MB 頭像大約 20 GB，一個月不到一美元。懸空紀錄與孤兒檔案的成本差了好幾個數量級，所以設計原則很清楚：**寧可留垃圾，不可留懸空指標。** 本篇所有做法都服務於這一句。

## 怎麼運作

### 規則一：key 不可變，新的就是新的

每次上傳都產生一個全新的 object key，包含隨機值，絕不覆蓋既有的 key。範例的 `DirStore.Put` 甚至用 `O_EXCL` 強制這件事：

```go title="service.go"
// newKey returns a key nobody has used before, so an upload never
// overwrites an object some record still points at.
func newKey(user string) (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("avatars/%s/%s", user, hex.EncodeToString(b[:])), nil
}
```

這一條讓「新舊兩份物件同時存在」成為正常狀態。舊的紀錄指向舊 key，新的指向新 key，兩者互不干擾；上傳失敗時，舊物件從頭到尾沒被碰過，所以舊紀錄永遠有效。覆蓋同一個 key 會同時破壞這兩個性質，後面的「常見坑」會再回來。

### 規則二：指標與待刪除在同一個交易

物件寫好之後，只有 **一個** 資料庫交易：把使用者指向新 key，同時把舊 key 寫進待刪除表 `pending_deletions`。

```go title="service.go"
	err = s.DB.Update(func(tx *Tx) error {
		if old := tx.Avatar(user); old != "" {
			tx.AddPendingDeletion(old) // same transaction as the pointer change
		}
		tx.SetAvatar(user, key)
		return nil
	})
```

換成真實資料庫，這個交易大概是：

```sql
BEGIN;
UPDATE users SET avatar_key = $new WHERE id = $user;
INSERT INTO pending_deletions (object_key, created_at) VALUES ($old, now());
COMMIT;
```

交易保證兩件事要嘛一起發生、要嘛都沒發生。不會有「指標換了，但沒人記得舊檔要刪」，也不會有「記了要刪舊檔，指標卻沒換」，後者會讓清除程式刪掉仍在使用的檔案。

### 規則三：刪除在交易之外，非同步、可重試

背景的清除程式（sweeper）掃描待刪除表，對每個 key 刪物件，**刪除成功後** 才刪掉那一列：

```go title="sweeper.go"
	for _, key := range s.DB.PendingDeletions() {
		if err := s.Store.Delete(key); err != nil {
			errs = append(errs, fmt.Errorf("delete %s: %w", key, err))
			continue // leave the row; the next run retries
		}
		err := s.DB.Update(func(tx *Tx) error {
			tx.RemovePendingDeletion(key)
			return nil
		})
		// ...
```

這裡有兩個設計。第一，失敗的 key 保留在表裡，下一輪再試，失敗不會讓別的 key 卡住。第二，`Delete` 必須 **冪等（idempotent）**：物件已經不存在時視為成功。因為清除程式可能在「物件刪掉了、資料列還沒刪」之間當機，重跑時會再刪一次同一個物件；若此時回錯誤，這一列就永遠清不掉。

### 每個當機點留下什麼

把三條規則組合起來，替換頭像的流程和各個當機點如下：

```text
步驟                         此刻當機，留下什麼                   使用者看到
────────────────────────────────────────────────────────────────────────────
1. Put 新物件（新 key）       寫到一半：Put 自己清掉半個檔案        舊頭像
                              或 key 沒人引用                      
2. BEGIN … COMMIT 之前        新物件無人引用（孤兒）；             舊頭像
                              資料庫完全沒變
3. COMMIT 之後、sweeper 之前  指標已換；舊物件在待刪除表          新頭像
4. sweeper 刪物件之後、        物件已刪，待刪除列還在；             新頭像
   刪待刪除列之前              重跑時冪等刪除視為成功
5. 全部完成                   舊物件與待刪除列都消失               新頭像
```

看右欄：沒有任何一列會出現「破圖」。每個當機點留下的殘骸只有兩種：孤兒檔案（第 2 步），和等待處理的待刪除列（第 3、4 步）。待刪除列有明確的清除路徑；孤兒檔案則需要下一節的對帳。

### 對帳掃描：回收第 2 步的孤兒

第 1 步成功、第 2 步之前當機，新物件沒有任何紀錄、也不在待刪除表，沒有任何東西記得它的存在。唯一的辦法是反過來從物件儲存找：列出所有物件，找出「沒有任何紀錄引用，也不在待刪除表」的。

```go title="sweeper.go"
	for _, o := range objs {
		if !known[o.Key] && now.Sub(o.ModTime) > grace {
			out = append(out, o.Key)
		}
	}
```

`grace`（寬限期）不能省略。一個正在上傳的請求，物件已經寫好、交易還沒提交，從掃描的角度看和孤兒一模一樣。若沒有寬限期，掃描會把「即將被引用」的新物件刪掉，造成真正的懸空紀錄。寬限期要遠大於一次上傳加提交的最長耗時，常見做法是數小時到一天。

:::caution
對帳掃描是整套設計裡唯一會「主動刪使用者可能還要用的東西」的環節。先在日誌中只列出候選，確認幾輪之後，再打開真的刪除。
:::

### 回到一開始的問題

回到一開始的問題：三步各自當機時，使用者看到什麼？照「刪舊、寫新、更新」的直覺順序，任何一個當機點都可能破圖。本篇的做法把順序改成「寫新 → 同交易換指標並記下舊的 → 非同步刪舊」，並且讓每一步都可以重跑。每個當機點之後，使用者看到的不是舊頭像就是新頭像，留下的殘骸只是孤兒檔案或待刪除列，都能被回收。

## 取捨與替代方案

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| 本篇：不可變 key、同交易待刪除表、非同步清除 | 沒有破圖；每步可重試；不需要特殊基礎設施 | 要多一張表、一個 sweeper、一個對帳掃描；舊物件會多活一小段時間 | 資料庫與物件儲存各自獨立，大多數產品 |
| 分散式交易（2PC / XA） | 理論上兩邊一起提交 | 物件儲存（S3、MinIO）沒有提供參與者介面；協調者當機會讓資源被鎖住；可用性降低 | 兩邊都支援 XA 的傳統企業資料庫 |
| 先刪舊再寫新 | 沒有殘骸 | 中途失敗就永久失去舊資料，還可能破圖 | 幾乎沒有 |
| 覆蓋同一個 key | 不需要待刪除表 | 無法回滾；CDN 與瀏覽器快取舊內容；並發上傳互相覆蓋 | 內容本來就不重要、可隨時重建的快取物件 |
| 啟用物件儲存的版本控制 | 舊版本由儲存層保留，可還原 | 儲存成本增加；仍要決定何時清掉舊版本；不解決「資料庫指向哪個版本」 | 需要稽核或誤刪還原的資料 |
| 與 Outbox 合用 | 刪除事件經佇列送出，多個消費者可以處理 | 多一套佇列；待刪除表其實就是最小的 Outbox | 刪除之後還要通知其他系統（搜尋、快取、CDN） |

最後一列值得多說一句。**Outbox 模式** 是指把「之後要做的事」和業務資料寫在同一個交易裡的表，再由背景程序送出去。本篇的待刪除表就是 Outbox 的最簡單版本：事件只有一種（刪物件），消費者只有一個（sweeper）。

為什麼不用 2PC：兩階段提交要求每個參與者在「準備」階段承諾能提交，並且鎖住資源直到協調者決定。物件儲存的 API 沒有這種承諾，幾乎所有雲端物件儲存都沒有提供；就算有，協調者當機時所有參與者都得等，可用性比本篇的「失敗就留殘骸、之後再收」差。

### 什麼時候不該用

- **物件和紀錄永遠一起生滅，且物件很小**：例如幾百 byte 的設定。直接存進資料庫欄位，一個交易就解決，根本不需要跨系統。
- **整個系統只有一個地方會寫物件，且物件不可變、也從不刪除**：沒有「舊的」要處理，規則一就夠了，不需要待刪除表。
- **儲存成本完全不在意，也不需要回收**：可以只留孤兒，不寫 sweeper 與對帳。但只有在刪除要求（例如個資刪除）不存在時成立；一旦有法規要求真的刪掉，非同步清除就不能省。

## 常見坑與故障案例

### 覆蓋同一個 key

**症狀**：使用者換了頭像，自己看到新圖，但別人看到的還是舊圖，或在不同裝置之間時新時舊；有時新圖上傳失敗後，舊頭像也跟著壞掉。

**原因**：新圖寫到和舊圖相同的 key（例如 `avatars/<user>.png`）。CDN 與瀏覽器依 URL 快取，舊內容活到過期為止；而且覆蓋寫入失敗時，原本完整的舊物件已經被破壞，沒有任何東西可以回滾。這是示意案例。

**避免**：每次上傳都用新 key，讓 URL 本身隨內容而變，快取問題隨之消失。

### 在交易提交之前就刪舊物件

**症狀**：偶爾有使用者回報頭像變成破圖，重新整理也不會好；資料庫的 `avatar_key` 指向一個已被刪除的物件。

**原因**：程式碼為了「一次做完」，在資料庫交易內（或交易之前）就呼叫 `Delete`。交易隨後因為死鎖、逾時或提交失敗而回滾，指標仍指向舊 key，物件卻已經不在。刪除不能被回滾，所以它不能放在可能回滾的交易裡面。這是示意案例。

**避免**：交易內只寫資料，永遠不呼叫外部系統；刪除一律在交易 **提交之後**，由 sweeper 做。

### 待刪除列寫在交易之外

**症狀**：資料庫裡指標換了，物件儲存裡舊物件卻一直留著，儲存用量緩慢上升，而待刪除表是空的。

**原因**：換指標的交易提交後，程式才用另一個語句插入待刪除列。兩個語句之間當機，舊 key 就沒有人記得了，這等於回到沒有待刪除表的狀態，只剩對帳掃描能救。

**避免**：換指標與寫待刪除列要在同一個交易，這正是規則二。

### sweeper 的刪除不是冪等的

**症狀**：待刪除表裡有幾列永遠清不掉，日誌每分鐘都重複同一行錯誤，例如「object not found」。

**原因**：sweeper 在「物件刪掉了、資料列還沒刪」之間當機，重跑時對一個已不存在的物件呼叫刪除，把「不存在」當成錯誤，於是那一列永遠卡住。

**避免**：把「物件不存在」視為刪除成功，範例的 `DirStore.Delete` 就是這麼做。

### 對帳掃描刪掉剛上傳、還沒提交的物件

**症狀**：尖峰時段偶爾有使用者換頭像後立即破圖；事後查，物件在被引用之前就被刪了。

**原因**：對帳掃描沒有寬限期，或寬限期比最慢的上傳加提交還短。物件剛寫入、交易尚未提交的那一刻，它沒有任何紀錄引用，看起來像孤兒，被掃描刪掉；交易提交後，指標就指向一個不存在的物件。

**避免**：以物件的建立時間判斷，只處理超過寬限期的物件；寬限期取保守的數小時以上，並先用「只記日誌」的模式觀察。

## 小練習

### 動手寫

範例的 `Sweep` 綁在 `Service` 上，這裡把它抽成可單獨測試的函式。請實作 `Sweep(s Store, q Queue) error`：依序對 `q.Pending()` 裡的每個 key 呼叫 `s.Delete`；刪除成功才呼叫 `q.Remove`；某個 key 失敗不能中止其他 key，失敗的列要留在佇列裡；最後把所有錯誤合併回傳。

起始程式碼在 `playground/backend/l4/4-1-file-upload/cross-system-consistency/exercise/`。做對時，下面的指令會通過：

```bash
go test -tags exercise ./exercise/
```

<details>
<summary>參考解答</summary>

```go title="sweep.go"
func Sweep(s Store, q Queue) error {
	var errs []error
	for _, k := range q.Pending() {
		if err := s.Delete(k); err != nil {
			errs = append(errs, fmt.Errorf("delete %s: %w", k, err))
			continue // keep the row so the next run retries it
		}
		if err := q.Remove(k); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
```

關鍵是「先刪物件、成功後才刪列」。反過來先刪列，刪物件失敗時就沒有任何東西記得它，變成孤兒。`continue` 讓一個壞 key 不拖累其他 key。

</details>

### 辨識

下面是 AI 產生的替換頭像程式。它能編譯，在沒有任何失敗的測試裡會通過。請做 code review，找出會在線上出問題的地方（至少三個）。

```go title="replace.go"
func (s *Service) ReplaceAvatar(user string, content io.Reader) error {
	key := "avatars/" + user + ".png"
	old := s.DB.Avatar(user)

	if old != "" {
		if err := s.Store.Delete(old); err != nil {
			return err
		}
	}
	if err := s.Store.Put(key, content); err != nil {
		return err
	}
	return s.DB.Update(func(tx *consistency.Tx) error {
		tx.SetAvatar(user, key)
		return nil
	})
}

// Cleanup removes objects that no avatar refers to.
func (s *Service) Cleanup() error {
	// ...
	for _, o := range objs {
		if !used[o.Key] && time.Since(o.ModTime) > 0 {
			s.Store.Delete(o.Key)
		}
	}
	return nil
}
```

完整檔案在 `recognize/replace.go`，`go test ./...` 不會掃到它。

<details>
<summary>參考解答</summary>

1. **先 `Delete(old)` 再 `Put`**：舊檔刪除後，只要 `Put` 或資料庫更新失敗，資料庫仍指向剛被刪的舊物件，使用者破圖，而且刪除無法復原。正確做法：先寫新物件，更新成功後才處理舊的。
2. **`key` 固定為 `avatars/<user>.png`，每次覆蓋同一個 key**：`Put` 在新 key 和舊 key 相同時，會直接破壞舊物件；CDN 與瀏覽器還會繼續回舊內容。正確做法：每次上傳產生含隨機值的新 key。
3. **刪舊物件沒有和指標更新放在同一個交易**：就算順序改對，沒有待刪除列，當機後就沒人記得舊物件，變成永遠的孤兒。正確做法：同一個交易內換指標並寫待刪除列，交由 sweeper 非同步刪。
4. **`Cleanup` 的 `time.Since(o.ModTime) > 0` 等於沒有寬限期**：幾乎所有物件都符合條件，包含剛寫入、交易尚未提交的新物件，這些會被當成孤兒刪掉，造成懸空紀錄。正確做法：設一個保守的寬限期，例如數小時。
5. **`Cleanup` 忽略 `Delete` 的錯誤，且沒有排除待刪除表中的 key**：失敗沒有被記錄，也沒有重試機制；若另有待刪除表，兩邊會互相爭著刪同一批物件。正確做法：記錄並回傳錯誤，把待刪除表中的 key 視為已知。

</details>

## 快閃卡

<details>
<summary>為什麼資料庫與物件儲存之間，設計目標是「寧可留孤兒，不可留懸空紀錄」？</summary>

懸空紀錄是使用者看得到的資料錯誤，也沒有東西能掃描發現；孤兒檔案只是多花一點儲存費，並且可以靠對帳掃描回收。兩者成本差距很大，所以順序要讓中斷只留下孤兒。

</details>

<details>
<summary>為什麼每次上傳都要用全新的 object key，而不是覆蓋同一個？</summary>

覆蓋失敗時舊物件已被破壞、無法回滾；CDN 與瀏覽器依 URL 快取舊內容；並發上傳會互相覆蓋。新 key 讓新舊物件並存，上傳失敗時舊紀錄仍然有效。

</details>

<details>
<summary>為什麼「更新指標」與「把舊 key 寫進待刪除表」要在同一個資料庫交易？</summary>

兩件事要嘛一起生效、要嘛都不生效。分開寫時，兩者之間當機會讓舊 key 沒有人記得，變成永遠的孤兒；也可能記了要刪卻沒換指標，讓清除程式刪掉仍在使用的檔案。

</details>

<details>
<summary>程式在資料庫交易內先呼叫 <code>Store.Delete(old)</code>，之後交易因死鎖而回滾。會發生什麼事？</summary>

資料庫的指標沒有換，仍指向舊 key，但舊物件已被刪除，變成懸空紀錄，使用者破圖。刪除無法回滾，所以不能放在可能回滾的交易裡，要等提交後由 sweeper 非同步做。

</details>

<details>
<summary>sweeper 的 <code>Delete</code> 為什麼必須冪等？物件不存在時該怎麼處理？</summary>

sweeper 可能在「物件已刪、待刪除列還在」之間當機，重跑時會再刪一次。把「不存在」當成錯誤，那一列就永遠清不掉，所以不存在要視為成功。

</details>

<details>
<summary>sweeper 對某個 key 的 <code>Delete</code> 失敗，應該怎麼處理那一列？</summary>

保留那一列、繼續處理其他 key，下一輪再試。只有物件確定刪除之後才刪列，否則失敗的物件就沒有任何東西記得它。

</details>

<details>
<summary>上傳一張頭像時，物件已寫入、資料庫交易尚未提交，此時對帳掃描發現這個物件沒被引用。若沒有寬限期會發生什麼事？</summary>

掃描把它當成孤兒刪掉；交易隨後提交，指標指向一個不存在的物件，造成懸空紀錄。寬限期讓「剛寫入、還沒被引用」的物件不會被誤刪。

</details>

<details>
<summary>為什麼不用 2PC 讓資料庫與物件儲存一起提交？</summary>

物件儲存（如 S3、MinIO）沒有提供兩階段提交的參與者介面；即使有，協調者當機時參與者要等待、鎖住資源，可用性比「失敗就留殘骸、之後再收」差。

</details>

## 延伸閱讀

- [Amazon S3 data consistency model](https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html#ConsistencyModel)：說明 S3 對新增、覆蓋與刪除物件的一致性保證
- [Amazon S3 Versioning](https://docs.aws.amazon.com/AmazonS3/latest/userguide/Versioning.html)：版本控制如何保留舊版本，以及它不解決什麼
- [Pattern: Transactional outbox](https://microservices.io/patterns/data/transactional-outbox.html)：Outbox 模式的原始描述，待刪除表是它的最小版本
- [Pattern: Saga](https://microservices.io/patterns/data/saga.html)：跨服務沒有共同交易時，另一種用補償動作換一致性的做法
