---
title: 檔案驗證
description: 用檔頭判定類型、伺服器決定副檔名、檔名只當顯示用，並替解碼與解壓設上限。
sidebar:
  order: 4
---

## 先想想

短網址服務 v4 的建立連結表單可以附一張預覽圖，上一篇已經把欄位、暫存區和資料庫的順序處理好了。現在有三個上傳請求通過了所有大小檢查：

1. 檔名 `avatar.png`，`Content-Type: image/png`，內容其實是一段 HTML，裡面有 `<script>fetch('/api/me')...</script>`。
2. 檔名 `../../etc/cron.d/x`，內容是一行文字。
3. 一個 33 byte 的 PNG，檔頭宣稱寬高都是 50000 像素。

三個請求都不大，也都「看起來」合法。你的 handler 要檢查什麼，才能分別擋下它們？哪一道檢查擋哪一個？如果某個請求真的被存下來，事後會在哪裡出事？

<details>
<summary>提示</summary>

先列出這個請求裡有哪些東西是客戶端自己寫的：檔名、副檔名、`Content-Type` 標頭、檔案內容。再想想伺服器能自己「讀出來」的有哪些。

</details>

## 是什麼

**檔案驗證（file validation）** 是在檔案落地或被處理之前，以伺服器自己讀到的位元組判定三件事：它是什麼類型、該叫什麼名字、展開之後有多大，並拒絕不符合規則的檔案。客戶端給的任何描述都只是參考，不是依據。

它和上一篇〈資料與檔案同一個請求〉的大小上限是兩層不同的檢查。大小上限問的是「送來多少 byte」，檔案驗證問的是「這些 byte 是什麼、會被怎麼用」。你可以把它想成資料欄位的輸入驗證，只是欄位換成了一個二進位的檔案，而且這個檔案之後還會被瀏覽器顯示、被圖片程式庫解碼、被寫到檔案系統的某個路徑。

本篇不處理病毒與惡意內容掃描、也不處理把使用者內容放到獨立網域，那是下一篇〈惡意檔案防護〉的主題。

## 為什麼存在

一個上傳請求裡，客戶端能控制的東西有：檔名（含副檔名）、每個 part 的 `Content-Type`、檔案內容本身。前兩者只是字串，任何人用 `curl` 都能隨便填。只看這兩個就決定怎麼處理檔案，會有三類後果：

- **儲存型 XSS**（stored XSS）：HTML 檔案被命名成 `avatar.png` 存下來，之後由你的網域提供下載。如果回應沒有固定 `Content-Type`，或瀏覽器自行猜測內容類型，它可能被當成 HTML 執行，腳本拿著使用者的 cookie 呼叫你的 API。
- **路徑穿越**（path traversal）：檔名被拿去組路徑，`../../etc/cron.d/x` 就從上傳目錄跳了出去，寫進系統排程目錄或覆蓋別人的檔案。這是 CWE-22。
- **壓縮炸彈**（decompression bomb）：檔案本身很小，展開之後非常大。一張 33 byte 的 PNG 宣稱 50000 × 50000 像素，以 RGBA 解碼要配置約 10 GB 記憶體（50000 × 50000 × 4 byte）；一個 10 KB 的 zip 可以解出 10 MB 的零，巢狀壓縮更能放大到 GB 等級。這是 CWE-409。

這三類問題的共通點是：大小上限擋不住，因為請求本身都很小。它們都利用了「伺服器相信了客戶端的描述」或「伺服器在檢查之前就開始處理」。檔案驗證要把判斷權拿回伺服器，並且在昂貴的處理之前先看檔頭。

## 怎麼運作

:::caution
本篇的攻擊範例（偽裝的 HTML、穿越路徑、像素炸彈、zip slip）都只在 `playground/backend/l4/4-1-file-upload/file-validation/` 的本機測試裡重現。只在你擁有或被授權測試的系統上嘗試。
:::

### 三種訊號，信任程度不同

伺服器手上有三種關於「這是什麼檔案」的訊號：

| 訊號 | 誰決定 | 能被偽造嗎 | 用途 |
| --- | --- | --- | --- |
| 副檔名（`.png`） | 客戶端，就是檔名的一部分 | 任意改 | 幾乎不用；最多當顯示用 |
| part 的 `Content-Type` 標頭 | 客戶端，瀏覽器通常依副檔名填 | 任意改 | 不當作判斷依據 |
| 檔頭（magic bytes） | 檔案內容本身 | 能，但必須讓內容真的長得像那個格式 | 判斷類型的起點 |

**檔頭（magic bytes）** 是多數檔案格式開頭固定的幾個 byte，例如 PNG 一定以 `89 50 4E 47 0D 0A 1A 0A` 開頭，GIF 以 `GIF89a` 或 `GIF87a` 開頭。Go 的 `http.DetectContentType` 實作了 WHATWG 的 MIME 嗅探（MIME sniffing）演算法，只看前 512 byte，回傳一個 MIME 類型；認不出來就回 `application/octet-stream`。

表格的重點在第三欄：前兩種訊號完全由客戶端決定，第三種至少要求內容配合。但它也只證明「開頭像」，後面會看到它的限制。

### 白名單，副檔名由伺服器決定

拿到檔頭之後，用 **白名單（allowlist）** 判斷：只接受明確列出的類型，其他一律拒絕。同一張表同時決定副檔名，客戶端給的副檔名完全不參與：

```go title="sniff.go"
// allowed maps a sniffed type to the extension the server will use.
// The client's extension is never consulted.
var allowed = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/gif":  ".gif",
}

// DetectType sniffs the first bytes of a file and checks the allowlist.
func DetectType(head []byte) (mime, ext string, err error) {
	mime = http.DetectContentType(head) // looks at no more than 512 bytes
	ext, ok := allowed[mime]
	if !ok {
		return mime, "", fmt.Errorf("%w: %s", ErrType, mime)
	}
	return mime, ext, nil
}
```

為什麼是白名單而不是黑名單？黑名單要列出所有危險類型（`.html`、`.svg`、`.js`、`.php`、`.exe`……），漏一個就破；白名單只列你真的需要的幾種，新格式出現時預設是拒絕。注意 SVG 不在名單上：它是 XML，可以內嵌 `<script>`，對瀏覽器來說它比較接近 HTML 而不是圖片。

偵測到的 MIME 類型要跟著檔案一起存進資料庫，之後提供下載時用它當 `Content-Type`，不要再用客戶端當初送來的那個值。

### 先讀檔頭，再決定要不要解碼

上一篇的檔案已經串流寫進暫存區，所以這裡可以開檔、讀前 512 byte、倒回開頭再讀圖片標頭。`Inspect` 把兩步串起來：

```go title="sniff.go"
func Inspect(f io.ReadSeeker, maxPixels int64) (Info, error) {
	head := make([]byte, 512)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return Info{}, err
	}
	mime, ext, err := DetectType(head[:n])
	if err != nil {
		return Info{}, err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return Info{}, err
	}
	// DecodeConfig parses only the header: a 40 KB file can claim
	// 50000x50000 pixels, and we want to know before allocating 10 GB.
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return Info{}, fmt.Errorf("%w: unreadable image header: %v", ErrType, err)
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxPixels {
		return Info{}, fmt.Errorf("%w: %dx%d", ErrPixels, cfg.Width, cfg.Height)
	}
	// ...
```

有三個細節。第一，`io.ReadFull` 遇到不足 512 byte 的檔案會回 `io.ErrUnexpectedEOF`，這不是錯誤，小檔案照樣要嗅探。第二，`image.DecodeConfig` 只解析標頭，回傳寬、高和色彩模型，不配置像素緩衝區；`image.Decode` 才會。所以順序必須是「先 `DecodeConfig` 檢查像素數，通過了才 `Decode`」（例如之後要做縮圖時）。第三，寬乘高用 `int64` 計算，避免兩個大數相乘溢位成一個小數而通過檢查。

測試裡用 33 byte 手工組出一個 PNG 標頭，宣稱 50000 × 50000，`Inspect` 在配置任何像素之前就回 `ErrPixels`。

### 檔名只是顯示用的中繼資料

上一篇已經定下規則：儲存用的 key 由伺服器產生隨機值，絕不用客戶端的檔名。那客戶端的檔名還有用嗎？有，使用者想在下載時看到 `報告.png` 而不是 `3f9a...`。所以檔名降級為 **顯示用的中繼資料**，存進資料庫的一個欄位，永遠不拿來組路徑。

即使只是顯示，也要先消毒（sanitize）：

```go title="filename.go"
func DisplayName(raw string) string {
	// Windows clients may send "C:\Users\a\b.png"; on Linux "\" is not a
	// separator, so filepath.Base would keep the whole thing.
	raw = strings.ReplaceAll(raw, `\`, "/")
	if i := strings.LastIndex(raw, "/"); i >= 0 {
		raw = raw[i+1:]
	}

	var b strings.Builder
	for _, r := range strings.ToValidUTF8(raw, "") {
		// Drop control characters and bidi overrides such as U+202E,
		// which can make "gnp.exe" display as "exe.png".
		if unicode.IsControl(r) || unicode.Is(unicode.Bidi_Control, r) {
			continue
		}
		b.WriteRune(r)
	}
	name := strings.Trim(b.String(), " .")
	// ... cut to 255 bytes on a rune boundary, "" becomes "file"
```

每一步各擋一種問題：只留最後一段，去掉目錄部分（包括 Windows 的反斜線）；去掉控制字元，避免 `\n` 混進日誌或 `Content-Disposition` 標頭；去掉雙向文字控制字元，避免 `report\u202Egnp.exe` 在畫面上顯示成 `reportexe.png`；修掉頭尾的空白和點；長度限制在 255 byte，並且在 rune 邊界切，不切出半個中文字。

輸出到 `Content-Disposition` 時，檔名還要做編碼，那是〈後端串流下載〉的主題。

### 路徑穿越：`Clean` 和 `Join` 不是防線

如果檔名或任何客戶端字串真的要參與路徑，常見的直覺寫法是 `filepath.Join(dir, filepath.Clean(name))`。它擋不住穿越：

```text
filepath.Join("/srv/uploads", "../../etc/cron.d/x")
          => "/etc/cron.d/x"
```

看結果的開頭：`Join` 會順便做 `Clean`，而 `Clean` 的工作是「把 `..` 算掉」，不是「禁止 `..`」，算完就真的跳出去了。`filepath.Base` 在 Linux 上又不認反斜線，`..\..\x` 對它來說是一個合法的檔名。

Go 提供兩個正確的工具。`filepath.IsLocal(name)`（Go 1.20）檢查一個路徑是否「留在目前目錄之內」：不是絕對路徑、清理後不以 `..` 開頭、不是空字串，在 Windows 上也不是 `NUL` 這類保留名稱。`os.Root`（Go 1.24）更進一步，它把一個目錄開成根，之後所有透過它的開檔、建檔都不能逃出去，包括透過符號連結逃出去：

```go title="filecheck_test.go"
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err := root.Create("../../etc/x"); err == nil {
		t.Fatal("os.Root allowed an escaping path")
	}
```

我們的預覽圖用伺服器產生的 key，本來就沒有穿越問題；`os.Root` 是縱深防禦。它真正必要的地方是下一節：檔名來自你無法控制的封存檔。

### 壓縮檔：所有上限都算實際寫出的 byte

短網址服務 v4 的批次匯入接受一個 zip，裡面是多個 CSV。zip 帶來兩個新問題。第一個是 **zip slip**：zip 裡每個項目的名稱由打包的人決定，可以是 `../../evil.sh`，解壓程式照名字建檔就寫到目錄外面。第二個是壓縮炸彈：zip 標頭裡的「解壓後大小」也是打包的人寫的，可以宣稱 1 KB，實際解出 1 GB。

```go title="archive.go"
func Extract(zr *zip.Reader, dir string, lim Limits) error {
	if len(zr.File) > lim.MaxEntries {
		return fmt.Errorf("%w: %d", ErrTooMany, len(zr.File))
	}
	root, err := os.OpenRoot(dir) // all opens below stay inside dir
	if err != nil {
		return err
	}
	defer root.Close()

	budget := lim.MaxTotal
	for _, zf := range zr.File {
		if !filepath.IsLocal(zf.Name) {
			return fmt.Errorf("%w: %q", ErrUnsafePath, zf.Name)
		}
		// ... skip directories
		n, err := extractOne(root, zf, budget)
		if err != nil {
			return err
		}
		budget -= n
	}
	return nil
}
```

三道上限各擋一種攻擊：項目數上限擋「一百萬個空檔案」耗盡 inode 與迴圈時間；`filepath.IsLocal` 加上 `os.Root` 擋 zip slip，前者給出清楚的錯誤，後者保證即使漏檢也逃不出去；總量預算擋壓縮炸彈。`extractOne` 用 `io.LimitReader(src, budget+1)` 複製，數的是實際寫出的 byte，完全不看 `UncompressedSize64`。測試裡 10 MB 的零壓成不到 64 KB，在 1 MB 預算處就被停下。

`archive/zip` 自己也知道 zip slip：設定 `GODEBUG=zipinsecurepath=0` 時，`zip.NewReader` 遇到非本地名稱會回 `zip.ErrInsecurePath`。但這不是預設行為，所以不要依賴它，自己檢查。

### 下載時把決定說清楚

驗證不只發生在上傳。提供下載時，用上傳當下偵測並存起來的類型當 `Content-Type`，並加上 `X-Content-Type-Options: nosniff`，叫瀏覽器不要自己猜：

```go title="serve.go"
func ServeStored(w http.ResponseWriter, info Info, body io.Reader) error {
	w.Header().Set("Content-Type", info.Type)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, err := io.Copy(w, body)
	return err
}
```

`nosniff` 讓瀏覽器信任你給的類型。這就是為什麼類型必須由伺服器在上傳時判定：你給的類型若來自客戶端，`nosniff` 反而保證了錯的類型被採用。

### 整條流程

```text
 暫存區裡的檔案
      │
      ▼
 讀前 512 byte ──► DetectContentType ──► 不在白名單？──► 415／422，刪暫存檔
      │
      ▼
 DecodeConfig ──► 寬 × 高 > 上限？──► 422，刪暫存檔
      │
      ▼
 伺服器決定：類型、副檔名、隨機 key
 客戶端檔名 ──► DisplayName ──► 只存進資料庫
      │
      ▼
 rename 進正式位置，insert 紀錄（上一篇的順序）
      │
      ▼
 下載：Content-Type = 存下的類型 + nosniff
```

看左邊那一條主線：每一個判斷都只用伺服器自己讀出來的資料，客戶端的檔名從右邊分出去，只走到資料庫的顯示欄位，從頭到尾沒有接近路徑。

### 回到一開始的問題

回到一開始的問題：三個請求分別被哪一道檢查擋下？

- **偽裝成 PNG 的 HTML**：檔名和 `Content-Type` 都說 PNG，但 `DetectContentType` 讀到的是 `text/html; charset=utf-8`，不在白名單，回 415 或 422。就算它過了，下載時固定的 `Content-Type: image/png` 加 `nosniff` 也讓瀏覽器不會把它當 HTML 執行。
- **`../../etc/cron.d/x`**：檔名從頭到尾不參與路徑，儲存 key 是伺服器產生的；顯示名稱經過 `DisplayName` 變成 `x`。如果是 zip 裡的項目名稱，`filepath.IsLocal` 拒絕它，`os.Root` 再保底。
- **33 byte 的像素炸彈**：大小檢查擋不住，`DecodeConfig` 讀出 50000 × 50000，超過像素上限，在配置任何像素緩衝區之前回 422。

三個請求都沒有寫出任何正式檔案，暫存檔照上一篇的規則在失敗路徑被清掉。

## 取捨與替代方案

「驗證到多深」是一個光譜，越深越安全，也越貴：

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| 只看副檔名 | 零成本 | 客戶端任意改，形同沒有驗證 | 只用來決定 UI 圖示，不做任何安全判斷 |
| 只看 part 的 `Content-Type` | 零成本 | 同樣由客戶端決定 | 不建議單獨使用 |
| 檔頭嗅探 + 白名單（本篇） | 便宜（讀 512 byte）；擋住絕大多數偽裝 | 只證明開頭像；擋不住 polyglot | 大多數上傳的基本盤 |
| 讀標頭（`DecodeConfig`）+ 上限 | 擋像素炸彈；確認格式能被解析 | 每種格式要有解碼器；仍沒有看完整個內容 | 會被解碼、縮圖的圖片 |
| 完整解碼後重新編碼 | 輸出是你自己產生的乾淨檔案，夾帶內容與中繼資料（EXIF）被丟掉 | CPU 與記憶體成本高；有損格式會再損失畫質；解碼器本身的漏洞仍是攻擊面 | 頭像、預覽圖這類會被大量公開顯示的圖片 |
| 交給外部掃描服務或沙箱 | 涵蓋病毒、惡意巨集等內容層攻擊 | 延遲、費用、要處理非同步結果 | 文件類上傳、企業客戶要求（下一篇） |

這幾層是疊加，不是擇一。短網址的預覽圖合理的組合是：嗅探 + 白名單 + `DecodeConfig` 上限，若之後要做縮圖，就在縮圖時順便重新編碼，對外只提供縮圖。

重新編碼值得多說一句：它是唯一真正「消毒內容」的做法，因為輸出的每個 byte 都是你的編碼器寫的，GIF 後面夾帶的腳本、JPEG 裡的 GPS 座標都不會留下。代價是你必須真的解碼它，所以 `DecodeConfig` 的上限檢查要在它之前。

### 什麼時候不該用

- **伺服器從不解析、也不對外提供的檔案**：例如使用者上傳自己的加密備份，只有本人用簽名 URL 下載回去。做深度內容驗證沒有意義（你也解不開），重點變成大小上限、隨機 key 和存取控制。
- **重新編碼在需要原檔的場景**：攝影作品、醫療影像、法律證據要保留原始 byte 與中繼資料，重新編碼會破壞它們；改用嗅探 + 獨立網域提供下載。
- **想用嗅探取代存取控制**：驗證回答「這是什麼檔案」，不回答「誰能看」。一張完全合法的 PNG 被錯的人下載，仍然是資料外洩。

## 常見坑與故障案例

### 相信 part 的 `Content-Type`

**症狀**：安全測試回報，上傳一個 `.html` 改名成 `.png`、標頭填 `image/png`，之後用檔案網址開啟，頁面跑出了測試腳本。

**原因**：白名單檢查的是 `hdr.Header.Get("Content-Type")`，那是客戶端自己填的；下載時又把同一個值回給瀏覽器，或根本沒設，讓瀏覽器自己猜。這是示意案例，但 OWASP 的檔案上傳指南把「不要信任 `Content-Type`」列為基本要求。

**避免**：類型用 `DetectContentType` 判定，存進資料庫；下載時用存下的類型並加 `nosniff`。

### 以為 `filepath.Join` 會擋 `..`

**症狀**：上傳目錄外出現了不認識的檔案，或某個設定檔被覆蓋。

**原因**：程式寫了 `filepath.Join(dir, filepath.Clean(name))`，以為清理過就安全。`Clean` 會把 `..` 算掉，結果正是目錄外的路徑；`filepath.Base` 在 Linux 上又不處理反斜線。

**避免**：儲存位置一律用伺服器產生的 key；不得不用外部名稱時，先 `filepath.IsLocal` 再透過 `os.Root` 開檔。

### Zip Slip：解壓縮時照單全收項目名稱

**症狀**：解壓使用者上傳的封存檔後，伺服器上的任意位置多了檔案，嚴重時是可執行的腳本或被覆寫的程式檔，演變成遠端程式碼執行。

**原因**：解壓程式用 `filepath.Join(dest, entry.Name)` 建檔，沒有檢查名稱是否包含 `../`。這是真實的漏洞類別：Snyk 在 2018 年 6 月公開 [Zip Slip](https://github.com/snyk/zip-slip-vulnerability)，影響上千個專案，包括 HP、Amazon、Apache、Pivotal 的專案，多個語言生態系的熱門函式庫都因此發了 CVE。

**避免**：每個項目先 `filepath.IsLocal`，所有建檔透過 `os.Root`；同時對項目數與解壓後總量設上限，上限算實際寫出的 byte。

### 以為過了檔頭檢查就是乾淨的圖片

**症狀**：上傳的 GIF 通過了嗅探和白名單，但同一個檔案在別的情境下被當成 JavaScript 或 Java 封存檔載入。

**原因**：嗅探只看開頭。一個以 `GIF89a` 開頭、後面接 `/*<script>...*/` 的檔案同時是合法的 GIF 開頭與合法的 JavaScript，這類檔案叫 **多重格式檔案**（polyglot）。2008 年公開的 GIFAR（GIF 與 JAR 合一）就是這類技巧的知名例子。範例的 `TestDetectTypeGIFPolyglotPasses` 證明了嗅探確實會放行它。

**避免**：嗅探是第一道，不是最後一道。對外顯示的圖片重新編碼；提供下載時固定 `Content-Type` 並加 `nosniff`；使用者內容放在獨立網域（下一篇）。

### 下載時忘了 `nosniff`

**症狀**：上傳端的驗證都做了，掃描工具仍然標出「MIME 嗅探」風險；某些舊瀏覽器或某些檔案會被顯示成 HTML。

**原因**：伺服器回應的 `Content-Type` 是 `application/octet-stream` 或空的，瀏覽器依 WHATWG 演算法自行判斷內容，可能判成可執行的類型。

**避免**：所有使用者上傳內容的回應都加 `X-Content-Type-Options: nosniff`，`Content-Type` 用上傳時判定的值；需要強制下載時再加 `Content-Disposition: attachment`。

## 小練習

### 動手寫

不看本文的 `Inspect`，在練習目錄自己完成一個。它要做到：嗅探前 512 byte（檔案可能更短）；只接受 PNG、JPEG、GIF，並由伺服器決定副檔名；用 `image.DecodeConfig` 讀尺寸，寬乘高超過上限就回 `ErrPixels`，而且不能溢位；成功時讀取位置要回到開頭，讓呼叫者可以接著存檔。

起始程式碼在 `playground/backend/l4/4-1-file-upload/file-validation/exercise/`。做對時，下面的指令會通過：

```bash
go test -tags exercise ./exercise/
```

<details>
<summary>參考解答</summary>

```go title="inspect.go"
func Inspect(f io.ReadSeeker, maxPixels int64) (Info, error) {
	head := make([]byte, 512)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return Info{}, err
	}
	mime := http.DetectContentType(head[:n])
	ext, ok := allowed[mime]
	if !ok {
		return Info{}, fmt.Errorf("%w: %s", ErrType, mime)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return Info{}, err
	}
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return Info{}, fmt.Errorf("%w: %v", ErrType, err)
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxPixels {
		return Info{}, fmt.Errorf("%w: %dx%d", ErrPixels, cfg.Width, cfg.Height)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return Info{}, err
	}
	return Info{Type: mime, Ext: ext, Width: cfg.Width, Height: cfg.Height}, nil
}
```

`allowed` 是本文那張 MIME 對副檔名的表。容易漏的是三個地方：短檔案的 `io.ErrUnexpectedEOF` 不是錯誤；嗅探之後要先 `Seek` 回開頭，`DecodeConfig` 才讀得到完整標頭；最後一次 `Seek` 是給呼叫者的，少了它，存下來的檔案會缺掉開頭。

</details>

### 辨識

下面是 AI 產生的頭像上傳與下載 handler。它能編譯，用一般的 PNG 測試會通過，也有大小上限。請做 code review，找出會在線上出問題的地方（至少三個）。

```go title="avatar.go"
var allowed = map[string]bool{"image/png": true, "image/jpeg": true}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	file, hdr, err := r.FormFile("avatar")
	if err != nil {
		http.Error(w, "missing avatar", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if !allowed[hdr.Header.Get("Content-Type")] {
		http.Error(w, "only png or jpeg", http.StatusUnsupportedMediaType)
		return
	}
	if _, _, err := image.Decode(file); err != nil {
		http.Error(w, "not an image", http.StatusUnprocessableEntity)
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	name := filepath.Clean(strings.TrimSpace(hdr.Filename))
	dst, err := os.Create(filepath.Join(h.Dir, name))
	// ... io.Copy(dst, file), 201
}

func (h *Handler) Serve(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(r.PathValue("name"))
	http.ServeFile(w, r, filepath.Join(h.Dir, name))
}
```

完整檔案在 `recognize/avatar.go`，`go test ./...` 不會掃到它。

<details>
<summary>參考解答</summary>

1. **白名單檢查的是 part 的 `Content-Type`**：這個值由客戶端填寫，HTML 檔案標成 `image/png` 就通過了。正確做法：用 `http.DetectContentType` 讀檔頭判定類型。（這段程式後面的 `image.Decode` 會擋掉純 HTML，但它擋不住 polyglot，而且第 2 點讓它本身變成問題。）
2. **直接 `image.Decode`，沒有先 `DecodeConfig`**：2 MB 上限擋不住像素炸彈，幾十 byte 的 PNG 宣稱 50000 × 50000，`Decode` 會嘗試配置約 10 GB，幾個請求就能讓程序被 OOM kill。正確做法：先 `image.DecodeConfig` 檢查寬乘高（用 `int64`），通過才解碼。
3. **用客戶端檔名組路徑**：`filepath.Clean` 不擋 `..`，`../../etc/cron.d/x` 經過 `Join` 就是 `/etc/cron.d/x`；兩個使用者都上傳 `avatar.png` 時後者蓋掉前者。正確做法：儲存 key 由伺服器隨機產生，客戶端檔名經 `DisplayName` 消毒後只存進資料庫。
4. **下載時沒有固定類型，也沒有 `nosniff`**：`http.ServeFile` 依副檔名或內容猜 `Content-Type`，存下來的檔案又是客戶端命名的，`x.html` 就會被當 HTML 提供。正確做法：用上傳時偵測並存下的類型設定 `Content-Type`，加 `X-Content-Type-Options: nosniff`。
5. **副檔名由客戶端決定**：檔名原樣保留，副檔名與實際內容可以不一致。正確做法：副檔名由白名單表依偵測到的類型決定。

</details>

## 快閃卡

<details>
<summary>上傳請求裡，副檔名、part 的 Content-Type、檔頭三者，哪些是客戶端可以任意決定的？</summary>

副檔名和 part 的 Content-Type 都是客戶端寫的字串，可以任意填。檔頭是內容本身，雖然也能偽造，但必須讓內容真的以該格式的固定 byte 開頭，所以是判斷類型的起點。

</details>

<details>
<summary>為什麼檔案類型用白名單而不是黑名單？</summary>

黑名單要列出所有危險類型，漏一個（例如 SVG 能內嵌腳本）就破；白名單只列需要的幾種，未知格式預設被拒絕。同一張表還能決定伺服器使用的副檔名。

</details>

<details>
<summary>一個 33 byte 的 PNG 通過了 2 MB 的大小上限，為什麼還可能讓服務 OOM？要怎麼擋？</summary>

PNG 標頭可以宣稱 50000 × 50000 像素，<code>image.Decode</code> 會依此配置約 10 GB 記憶體。先用 <code>image.DecodeConfig</code> 只讀標頭，檢查寬乘高是否超過上限，通過才解碼。

</details>

<details>
<summary><code>filepath.Join("/srv/uploads", "../../etc/x")</code> 的結果是什麼？為什麼？</summary>

是 <code>/etc/x</code>。<code>Join</code> 會做 <code>Clean</code>，而 <code>Clean</code> 把 <code>..</code> 算掉，不是禁止它，所以結果跳出了上傳目錄。要用 <code>filepath.IsLocal</code> 檢查，或透過 <code>os.Root</code> 開檔。

</details>

<details>
<summary>客戶端送來的檔名在本文的設計裡扮演什麼角色？</summary>

只是顯示用的中繼資料：消毒後存進資料庫，給使用者看。它從不參與儲存路徑，儲存 key 由伺服器隨機產生，副檔名由偵測到的類型決定。

</details>

<details>
<summary>解壓使用者上傳的 zip 時，用標頭裡的 UncompressedSize64 判斷是否超過上限，有什麼問題？</summary>

標頭裡的大小是打包的人寫的，可以宣稱 1 KB 實際解出 1 GB。上限要算實際寫出的 byte，例如用 <code>io.LimitReader</code> 包住每個項目，並維護一個總量預算。

</details>

<details>
<summary>一個以 GIF89a 開頭、後面接著 JavaScript 的檔案，會通過 DetectContentType 加圖片白名單嗎？之後要靠什麼保護？</summary>

會，嗅探只看開頭，它被判成 <code>image/gif</code>，這種同時符合兩種格式的檔案叫 polyglot。之後要靠下載時固定 <code>Content-Type</code> 加 <code>nosniff</code>、重新編碼對外顯示的圖片，以及獨立網域。

</details>

<details>
<summary>下載回應加了 X-Content-Type-Options: nosniff，但 Content-Type 用的是上傳時客戶端送的值，安全嗎？</summary>

不安全。<code>nosniff</code> 叫瀏覽器相信伺服器給的類型，如果這個類型來自客戶端，攻擊者填 <code>text/html</code> 就會被瀏覽器照單全收。類型必須是伺服器上傳時偵測並存下的值。

</details>

## 延伸閱讀

- [OWASP File Upload Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html)：檔案上傳防護的完整清單，包括副檔名、類型、檔名與儲存位置
- [WHATWG MIME Sniffing Standard](https://mimesniff.spec.whatwg.org/)：`http.DetectContentType` 與瀏覽器共同依據的嗅探演算法
- [Go blog: Traversal-resistant file APIs](https://go.dev/blog/osroot)：`filepath.IsLocal` 與 `os.Root` 的設計動機與用法
- [Snyk: Zip Slip Vulnerability](https://github.com/snyk/zip-slip-vulnerability)：2018 年公開的 zip slip 漏洞說明與受影響專案清單
- [MDN: X-Content-Type-Options](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/X-Content-Type-Options)：`nosniff` 對瀏覽器行為的確切影響
- [CWE-409: Improper Handling of Highly Compressed Data](https://cwe.mitre.org/data/definitions/409.html)：壓縮炸彈的弱點分類與緩解建議
