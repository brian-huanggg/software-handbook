---
title: 資料與檔案同一個請求
description: 讓一個 multipart 請求同時帶欄位與檔案，做到先驗證再落地、限制大小、失敗不留殘骸。
sidebar:
  order: 3
---

## 先想想

短網址服務 v4 要讓使用者在建立連結時附上一張預覽圖。前端送出一個表單：兩個文字欄位 `target_url`、`title`，加一個檔案 `preview`。檔案存在磁碟上，連結的資料列存在資料庫裡。

現在有三種失敗：使用者把 `target_url` 打成 `not a url`，卻附了一張 8 MB 的圖；檔案寫好了，資料庫的 insert 卻因為連線逾時失敗；或是上傳到一半，使用者關掉分頁。

這三種情況結束之後，磁碟上和資料庫裡各應該剩下什麼？你的 handler 要用什麼順序處理欄位、檔案和資料庫，才能讓答案永遠是「什麼都沒剩下」，或「兩邊都有」？

<details>
<summary>提示</summary>

想想兩個方向的失敗哪一個比較糟：資料庫有紀錄但檔案不存在，還是檔案存在但沒有任何紀錄指向它？

</details>

## 是什麼

**資料與檔案同一個請求** 是指用一個 `multipart/form-data` 請求，同時送出文字欄位與檔案，由伺服器一次處理完。請求本體被切成多個 part，每個 part 有自己的名稱，可能是欄位，也可能是檔案。

它和 `multipart/form-data` 的解析（逐個 part 串流讀取、不整包進記憶體）是兩件事。解析回答「怎麼讀」，本篇回答「讀的過程中，什麼時候驗證、什麼時候落地、失敗時怎麼收拾」。你可以把它看成一個小型的兩階段提交：檔案系統和資料庫是兩個互不知道對方的系統，handler 要自己決定先動哪一個。

本篇不談檔案內容是否安全（副檔名、MIME、檔頭），那是下一篇「檔案驗證」的主題；也不談把檔案直接傳到物件儲存，那是「上傳的三種做法」比較過的另一條路。這裡的範例只把檔案存在本機磁碟。

## 為什麼存在

最直覺的寫法是 `r.ParseMultipartForm(32 << 20)`，之後 `r.FormValue` 拿欄位、`r.FormFile` 拿檔案。它能動，但有三個後果：

- 整個請求在 handler 的第一行就讀完了。`maxMemory` 只決定多少 byte 留在記憶體，超過的部分會寫到暫存檔，不是請求大小的上限。送一個 2 GB 的檔案，磁碟就多 2 GB 的暫存檔。
- 驗證在檔案收完之後才發生。欄位錯誤的請求，伺服器仍然付出了收完整個檔案的頻寬、磁碟和時間。
- 檔案與資料庫沒有順序。寫 handler 的人通常照「欄位、檔案、資料庫」的直覺順序寫，其中任何一步失敗，前面做完的步驟都沒人收拾。

用數字感受一下第二點。假設預覽圖上限是 2 MB，一個壞掉的前端每次都送 `title` 為空的請求，每分鐘 600 次（皆為估算）。整包收完再驗證，每分鐘白收 1.2 GB；先驗證欄位，在檔案開始前就回 422，頻寬幾乎是零。

第三點更隱蔽。檔案系統和資料庫沒有共同的交易，所以失敗的組合有四種：兩邊都成功、兩邊都失敗、只有檔案、只有紀錄。我們的目標是讓「只有紀錄」（懸空紀錄，dangling record）永遠不會發生，「只有檔案」（孤兒檔案，orphan file）即使發生也要能被事後清掉。這就是本篇所有設計的出發點。

## 怎麼運作

:::note
本篇的範例在 `playground/backend/l4/4-1-file-upload/upload-with-data/`。檔案存在本機目錄，資料表用記憶體實作，重點是 handler 的順序。
:::

### 請求本體的形狀決定順序

瀏覽器和多數 HTTP 客戶端依欄位出現的順序，把 part 依序寫進請求本體。伺服器串流讀取時，part 就是照這個順序到達：

```text
POST /links  Content-Type: multipart/form-data; boundary=XYZ

--XYZ
Content-Disposition: ... name="target_url"
https://example.com/a                      <- 小：幾十 byte
--XYZ
Content-Disposition: ... name="title"
hi                                         <- 小
--XYZ
Content-Disposition: ... name="preview"; filename="a.png"
<2 MB 的二進位內容……>                      <- 大：成本在這裡
--XYZ--
```

看箭頭的註記：文字欄位很小，檔案很大。如果 `preview` 排在最前面，伺服器要先讀完整個檔案才看得到欄位，無法「先驗證再落地」。所以第一條規則是：**契約要求文字欄位在檔案之前**，違反就回 400。這是 API 的規格，要寫進文件，前端用 `FormData` 時 `append` 的順序就是送出順序。

### 先給整個請求一個上限

串流讀取不代表可以無限讀。在做任何事之前，先替請求本體加上上限：

```go title="upload.go"
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Cap the whole body first: every later read, even a field, is bounded.
	r.Body = http.MaxBytesReader(w, r.Body, h.MaxBody)

	mr, err := r.MultipartReader() // streams parts, buffers no file
	if err != nil {
		http.Error(w, "expected multipart/form-data", http.StatusBadRequest)
		return
	}
	// ...
```

`http.MaxBytesReader` 包住 `r.Body`，讀超過上限時回傳 `*http.MaxBytesError`，並且通知 server 讓連線在回應後關閉。它是底線：不論 `Content-Length` 寫多少，甚至根本沒寫（chunked），超過上限的 byte 都讀不到。看 `Content-Length` 提早回 413 可以更快，但它是客戶端自己宣告的，不能取代 `MaxBytesReader`。

`MultipartReader` 則一次只前進一個 part，不會把檔案讀進記憶體，也不會自己建暫存檔。

上限有三層，各管不同的東西：

| 上限 | 範例值 | 擋住什麼 |
| --- | --- | --- |
| `MaxBody` | 3 MB | 整個請求，含所有欄位與 multipart 邊界 |
| `MaxFile` | 2 MB | 單一檔案 part，給使用者一個明確的檔案大小規則 |
| `MaxField` | 4 KB | 單一文字欄位，防止有人把 `title` 當成上傳通道 |

`MaxBody` 要比 `MaxFile` 大一點，留給欄位和邊界；兩者都需要，因為單看檔案上限擋不住一個請求塞十個欄位。

### 欄位先收，檔案抵達前先驗證

主迴圈逐個 part 往前走。文字欄位讀進一個小 map；輪到檔案時，欄位都已經在手上，**先驗證，通過才讀檔案的第一個 byte**：

```go title="upload.go"
		if name != "preview" {
			b, err := io.ReadAll(io.LimitReader(part, h.MaxField+1))
			// ... size check, then:
			fields[name] = string(b)
			continue
		}

		// The file arrives: validate every field we already have *before*
		// reading a single byte of the file.
		if err := validate(fields); err != nil {
			h.fail(w, err, tmp)
			return
		}
		tmp, err = h.Store.Stage(part, h.MaxFile)
```

`io.LimitReader(part, h.MaxField+1)` 多讀一個 byte，是為了分辨「剛好等於上限」和「超過上限」。驗證失敗時直接回 422，檔案那個 part 根本沒被讀。測試用一個計數的 reader 證明了這點：2 MB 的請求，伺服器在回 422 之前讀到的 byte 不到十分之一。

```go title="upload_test.go"
	if cr.n > total/10 {
		t.Fatalf("server read %d of %d bytes before rejecting", cr.n, total)
	}
```

同一個迴圈也處理順序規則：檔案已經到了，後面又來一個 part，就回 400，並丟掉剛寫好的暫存檔。

### 檔案先進暫存區，不直接進正式位置

通過驗證後，檔案以串流方式寫進暫存區（staging area，即暫存目錄），同時檢查大小：

```go title="store.go"
func (s DirStore) Stage(r io.Reader, max int64) (tmpPath string, err error) {
	f, err := os.CreateTemp(s.tmpDir(), "upload-*")
	// ...
	// Read one byte past the limit so "exactly max" and "too big" differ.
	n, err := io.Copy(f, io.LimitReader(r, max+1))
	if err != nil {
		_ = f.Close()
		return "", err
	}
	if n > max {
		_ = f.Close()
		return "", errTooLarge
	}
	// ...
```

這裡有兩個重點。第一，檔案是邊收邊寫，記憶體只用到固定大小的複製緩衝區，不管檔案多大。第二，寫到一半被判定太大或連線中斷，殘骸只會在 `tmp/`，不會出現在正式目錄，沒有任何東西會去引用它。`defer` 裡的清理讓錯誤路徑自己收乾淨。

暫存目錄與正式目錄要在同一個檔案系統，這樣最後的 `os.Rename` 是原子的：觀察者只會看到「沒有這個檔案」或「完整的檔案」，不會看到寫到一半的。

### 先動檔案，再動資料庫

欄位通過、檔案完整收進暫存區之後，剩下的是檔案系統和資料庫的先後。範例的順序是：

1. 產生一個全新的隨機 key（由伺服器產生，絕不用客戶端給的檔名）。
2. 把暫存檔 rename 成這個 key。
3. 對資料庫 insert 一筆紀錄，內含這個 key。
4. 如果第 3 步失敗，刪掉第 2 步的檔案。

```go title="upload.go"
	if err := h.Store.Commit(tmp, key); err != nil {
		h.fail(w, err, tmp)
		return
	}
	l := Link{Code: key[:7], TargetURL: fields["target_url"], Title: fields["title"], PreviewKey: key}
	if err := h.Repo.Create(r.Context(), l); err != nil {
		if rmErr := h.Store.Remove(key); rmErr != nil {
			log.Printf("orphan object %s: %v", key, rmErr) // left for the sweeper
		}
		h.fail(w, err, "")
		return
	}
```

為什麼是這個順序？看每一步之後如果程序死掉（當機、被 kill），留下什麼：

```text
步驟            此刻若中斷，留下什麼                 後果
─────────────────────────────────────────────────────────────
rename 之前     tmp/ 裡一個暫存檔                    無人引用，定期清掉
rename 之後     objects/ 裡一個沒人引用的檔案        孤兒檔案，可事後掃描清除
insert 之後     檔案和紀錄都在                       完成
```

反過來先 insert 再寫檔，中斷的後果是「有紀錄、沒有檔案」：使用者的連結頁面會顯示一張壞掉的圖，而且沒有任何掃描能靠檔案系統發現它。孤兒檔案多花一點空間，可以靠定期掃描「沒有被任何紀錄引用的檔案」來清；懸空紀錄是資料錯誤，使用者看得到。**寧可多一個孤兒檔案，也不要留下一筆指向不存在檔案的紀錄。**

這不是魔法：第 4 步的刪除本身也可能失敗，所以範例把它記在日誌裡「留給清掃程式」。孤兒檔案的清除、以及更完整的跨系統一致性（把舊檔記入待刪除表、與資料庫同一個交易）在後面的「中繼資料與物件分離」和「跨系統一致性」兩篇處理。本篇只要求：失敗時盡力回收，回收不了的，留下的是無害的那一種殘骸。

### 把失敗對應成狀態碼

所有錯誤匯集到 `fail`，它做兩件事：刪掉還沒提交的暫存檔，並把錯誤對應成狀態碼：

```go title="upload.go"
	var tooBig *http.MaxBytesError
	switch {
	case errors.As(err, &tooBig), errors.Is(err, errTooLarge), errors.Is(err, errFieldTooLarge):
		http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
	case errors.Is(err, errFieldAfterFile):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, errBadURL), errors.Is(err, errNoFile):
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
	default:
		log.Printf("upload: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
```

三類狀態碼的意思不同：413 是大小超過規則，400 是請求形狀不符合契約（不是 multipart、欄位順序錯），422 是格式正確但內容不合法。內部錯誤（資料庫掛了、磁碟滿了）只回 500，細節進日誌，不回給客戶端。

最後，別忘了伺服器本身的逾時。`main.go` 設了 `ReadTimeout`，它限制讀完整個請求的時間，擋住用極慢速度上傳、一直佔住連線的客戶端。

### 回到一開始的問題

回到一開始的問題：三種失敗之後，磁碟與資料庫各剩什麼？

- 欄位錯誤卻附 8 MB 圖：欄位在檔案之前被驗證，回 422，檔案沒被讀，**兩邊都沒有東西**。
- 檔案寫好但 insert 失敗：rename 後的檔案被刪掉，回 500，**兩邊都沒有東西**。
- 上傳到一半關掉分頁：`NextPart` 或 `Stage` 讀到錯誤，暫存檔被清掉，**兩邊都沒有東西**。

唯一可能殘留的是「刪除失敗」或「rename 與 insert 之間程序死掉」留下的孤兒檔案，它沒有任何紀錄指向，不影響使用者，之後能被掃描清除。

## 取捨與替代方案

單一 multipart 請求不是唯一做法。同樣的需求還有兩個常見替代：先建資料再傳檔的兩步驟，以及讓客戶端直接傳到物件儲存。

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| 單一 multipart 請求（本篇） | 一次呼叫，客戶端簡單；伺服器能在同一處驗證欄位與檔案 | 檔案流經應用伺服器，吃頻寬與連線；失敗要整份重傳；欄位順序成為契約 | 檔案小（數 MB 以內）、表單與檔案必須一起成功 |
| 兩步驟：先 `POST /links` 建資料，再 `PUT /links/{id}/preview` 傳檔 | 檔案失敗可單獨重試；每個請求單純，容易搭配冪等 | 中間狀態（有連結、還沒有圖）要處理；多一次往返 | 檔案較大、需要重試、或檔案可有可無 |
| 預簽名 URL 直傳物件儲存 | 檔案不經過應用伺服器；大檔省頻寬 | 要在上傳完成後確認物件存在並綁定紀錄；驗證要移到事後 | 大檔、高併發、已經有物件儲存 |

單一請求的優勢在於「資料與檔案要嘛一起存在、要嘛都不存在」這個語意最容易講清楚；兩步驟把這個保證拆成兩個狀態，換來可重試。選擇時看三件事：檔案多大、失敗後使用者願不願意重傳、中間狀態在產品上是否可接受。

### 什麼時候不該用

- **檔案大到重傳不可接受**（幾十 MB 以上、行動網路）：單一請求一斷就全部重來，改用分片續傳或直傳物件儲存。
- **檔案是可選的附屬品**：不要為了一張可有可無的圖，讓整個「建立連結」請求承擔檔案失敗的風險，拆成兩步驟。
- **應用伺服器的連線很珍貴**：每個上傳佔住一條連線與一個 goroutine 的時間和檔案大小、使用者網速成正比；大量慢速上傳會擠掉正常的請求，該把上傳分流到別的服務或直傳。
- **請求需要冪等重試**：multipart 本體很難放進冪等金鑰的比對流程，重試前要先確認前一次是否已經成功。

## 常見坑與故障案例

### 把 `ParseMultipartForm` 的參數當成大小上限

**症狀**：磁碟用量在某個下午暴增，`/tmp` 滿了，其他服務連帶寫入失敗。

**原因**：`ParseMultipartForm(32 << 20)` 的參數是記憶體上限，超過的部分寫到暫存檔，沒有任何東西限制總量。有人（或腳本）送一個很大的檔案，每個請求都在磁碟上留下同樣大的暫存檔，而且在 handler 讀完前就已發生，欄位錯誤也來不及擋。這是示意案例。

**避免**：先用 `http.MaxBytesReader` 限制整個本體，再用 `MultipartReader` 串流處理，別呼叫 `ParseMultipartForm`。

### 檔案排在欄位前面，驗證變成事後

**症狀**：測試時前端用的是自己手寫的表單，順序剛好是欄位在前，一切正常；換成另一個客戶端（行動 App、第三方）後，欄位驗證失敗的請求仍然把整個檔案收完才回 422，日誌顯示頻寬異常。

**原因**：handler 假設欄位一定在檔案之前，卻沒有在遇到順序錯誤時拒絕，而是默默地繼續讀，或者 `validate` 只在檔案寫完後才呼叫。

**避免**：把順序寫進 API 文件；handler 在遇到檔案 part 時，對「必要欄位尚未出現」直接回 4xx，不嘗試補救。

### 先寫資料庫，再寫檔案

**症狀**：使用者回報某些連結的預覽圖是破圖；資料庫裡 `preview_key` 有值，物件儲存裡找不到對應檔案。

**原因**：程式碼先 insert 再存檔。磁碟滿、權限錯誤、程序被重啟，任何一個發生在 insert 與存檔之間，都會留下懸空的紀錄。因為沒有任何其他機制記得「這筆紀錄的檔案還沒寫成功」，這類錯誤只能靠使用者回報才被發現。這也是示意案例。

**避免**：檔案先、紀錄後。讓可能留下的殘骸是「沒有人引用的檔案」，並對它寫一個清掃程式。

### 失敗路徑沒有清暫存檔，也沒有檢查複製的錯誤

**症狀**：`tmp/` 目錄越來越大；或使用者收到 201，下載預覽圖卻發現只有一半。

**原因**：成功路徑寫得很仔細，但每個 `return` 前都要記得 `Remove` 暫存檔，漏掉一處就多一個殘骸；`io.Copy` 的錯誤被忽略時，連線中斷造成的半個檔案會被當成完整檔案提交。

**避免**：把「清理」集中到一個地方（範例的 `Stage` 用 `defer`，handler 用 `fail`），並且永遠檢查 `io.Copy` 的錯誤。再加一個定期清掃，刪除 `tmp/` 裡超過一定時間的檔案，作為最後一道防線。

## 小練習

### 動手寫

範例的 `commit` 把「搬檔案、寫紀錄、失敗時回收」寫在 handler 裡。請把這個順序抽成一個可單獨測試的 `Save`：依本文的規則，先搬檔案，再 insert 紀錄；搬檔案失敗時直接回傳錯誤，不碰資料庫；insert 失敗時刪掉剛搬的檔案，並回傳 insert 的錯誤。

起始程式碼在 `playground/backend/l4/4-1-file-upload/upload-with-data/exercise/`。做對時，下面的指令會通過：

```bash
go test -tags exercise ./exercise/
```

<details>
<summary>參考解答</summary>

```go title="save.go"
func Save(ctx context.Context, s Store, r Repo, tmpPath, key string) error {
	if err := s.Commit(tmpPath, key); err != nil {
		return err
	}
	if err := r.Create(ctx, key); err != nil {
		_ = s.Remove(key)
		return err
	}
	return nil
}
```

順序是關鍵：搬檔案在前，所以 `Commit` 失敗時資料庫根本沒被呼叫，不會有懸空紀錄。insert 失敗時的 `Remove` 是盡力而為，這裡不處理它的錯誤；正式的程式碼會像範例那樣記日誌，留給清掃程式。

</details>

### 辨識

下面是 AI 產生的建立連結 handler，目的是接收欄位與預覽圖並儲存。它能編譯，用一般表單和小圖片測試都會通過。請做 code review，找出會在線上出問題的地方（至少三個）。

```go title="handler.go"
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("preview")
	if err != nil {
		http.Error(w, "missing preview", http.StatusUnprocessableEntity)
		return
	}
	defer file.Close()

	target := r.FormValue("target_url")
	if _, err := url.ParseRequestURI(target); err != nil {
		http.Error(w, "bad target_url", http.StatusUnprocessableEntity)
		return
	}

	key := target[len(target)-6:] + ".img"
	if err := h.Repo.Create(r.Context(), target, key); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	dst, err := os.Create(filepath.Join(h.Dir, key))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	io.Copy(dst, file)
	dst.Close()
	w.WriteHeader(http.StatusCreated)
}
```

完整檔案在 `recognize/handler.go`，`go test ./...` 不會掃到它。

<details>
<summary>參考解答</summary>

1. **`ParseMultipartForm(32 << 20)` 沒有搭配 `MaxBytesReader`**：32 MB 只是記憶體上限，超過的部分寫到暫存檔，請求總量沒有限制。後果是一個大請求就能填滿磁碟。正確做法：先用 `http.MaxBytesReader` 限制本體，並用 `MultipartReader` 串流處理。
2. **`target_url` 在檔案已被完整收下之後才驗證**：`ParseMultipartForm` 在 handler 第一行就讀完了整個請求，之後的驗證擋不住頻寬與磁碟的浪費。正確做法：欄位排在檔案前面，檔案抵達前先驗證，失敗就不讀檔案。
3. **先 `Repo.Create`，再 `os.Create` 與複製檔案**：任何一步在 insert 之後失敗（磁碟滿、權限、程序重啟），都留下指向不存在檔案的紀錄，而且 handler 已經回 500 或 201 了。正確做法：先把檔案放到新 key，再 insert，insert 失敗就刪檔案。
4. **`io.Copy` 的錯誤被忽略，且直接寫進正式位置**：連線中斷時，寫到一半的檔案留在正式目錄，handler 仍然回 201。正確做法：寫進暫存區，檢查 `io.Copy` 的錯誤，成功後才 rename。
5. **`key` 由 `target_url` 的最後六個字元組成**：URL 少於 6 個字元會 panic；兩個結尾相同的連結得到同一個 key，後上傳的檔案蓋掉前一個，兩筆紀錄指向同一個檔案。正確做法：key 由伺服器產生的隨機值決定，每次上傳都是全新的。

</details>

## 快閃卡

<details>
<summary><code>ParseMultipartForm(32 &lt;&lt; 20)</code> 的 32 MB 限制了什麼？</summary>

它限制多少 byte 留在記憶體，超過的部分寫到暫存檔，不是請求大小的上限。要限制請求總量，要在它之前用 <code>http.MaxBytesReader</code> 包住 <code>r.Body</code>。

</details>

<details>
<summary>為什麼要求文字欄位排在檔案 part 之前？</summary>

multipart 的 part 依序到達，伺服器串流處理時，只有欄位先到，才能在讀取檔案的第一個 byte 之前驗證，驗證失敗就不付出收檔案的成本。這個順序是 API 契約。

</details>

<details>
<summary>用 <code>io.LimitReader(part, max)</code> 讀欄位，剛好等於上限的內容和超過上限的內容分得出來嗎？</summary>

分不出來，兩者都讀到 max 個 byte 就停。改成 <code>max+1</code>，讀到的長度大於 max 就代表超過；檔案寫入也是同樣的做法。

</details>

<details>
<summary>檔案已用新 key 存好，但資料庫 insert 因逾時失敗，handler 應該怎麼做？</summary>

刪掉剛存的檔案並回 500；刪除失敗就記日誌，留給清掃程式。目標是不留下指向不存在檔案的紀錄，留下無人引用的檔案是可以接受的殘骸。

</details>

<details>
<summary>為什麼是「先動檔案、再 insert 紀錄」，而不是反過來？</summary>

兩個系統沒有共同交易，中斷是無法避免的。先檔案後紀錄，中斷時留下孤兒檔案，沒有人引用，可以事後掃描清除；反過來會留下指向不存在檔案的紀錄，使用者看得到，也很難發現。

</details>

<details>
<summary>上傳的檔案為什麼先寫進暫存目錄，再 <code>os.Rename</code> 到正式位置？</summary>

寫到一半的失敗（太大、連線中斷）只會留在暫存區，沒有任何紀錄引用它。暫存與正式目錄在同一個檔案系統時，rename 是原子的，不會有人讀到不完整的檔案。

</details>

<details>
<summary>有人送出的請求是 <code>Content-Length: 1000</code>，實際卻一直傳資料，沒有設任何 <code>MaxBytesReader</code>。伺服器會發生什麼事？</summary>

伺服器會繼續讀、繼續寫進暫存檔，直到連線關閉或磁碟滿，因為 <code>Content-Length</code> 是客戶端自己宣告的，不會自動限制讀取量。要用 <code>MaxBytesReader</code> 在讀取端強制上限。

</details>

<details>
<summary>請求的 <code>target_url</code> 是 <code>not a url</code>，後面附一個 2 MB 的檔案，用本文的 handler 處理會回什麼？伺服器讀了多少檔案內容？</summary>

回 422。檔案 part 抵達時，handler 先驗證已收到的欄位，驗證失敗就直接回應，沒有讀檔案內容，所以伺服器只讀到檔案 part 的標頭，也沒有暫存檔留下。

</details>

## 延伸閱讀

- [net/http: MaxBytesReader](https://pkg.go.dev/net/http#MaxBytesReader)：說明如何限制請求本體大小，以及超過時回傳的 `*MaxBytesError`
- [net/http: Request.MultipartReader](https://pkg.go.dev/net/http#Request.MultipartReader)：串流讀取 multipart 的方法，與 `ParseMultipartForm` 的差別
- [net/http: Request.ParseMultipartForm](https://pkg.go.dev/net/http#Request.ParseMultipartForm)：`maxMemory` 參數的確切意義
- [mime/multipart](https://pkg.go.dev/mime/multipart)：`Reader`、`Part` 與 `FormName` 的文件
- [RFC 7578: Returning Values from Forms: multipart/form-data](https://www.rfc-editor.org/rfc/rfc7578)：multipart/form-data 格式的規格
