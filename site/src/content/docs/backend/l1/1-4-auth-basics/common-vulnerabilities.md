---
title: 常見漏洞入門：XSS、CSRF、敏感資料外洩
description: 認得三種最常見的 Web 漏洞，知道後端各要做哪一道防禦，並知道深入內容在資安領域的哪裡。
sidebar:
  order: 6
---

## 先想想

短網址服務 v1 有個管理頁：任何訪客都能送一筆「檢舉」，管理員 alice 登入後在 `/admin` 看這些檢舉。API 也有 `GET /me`，前端用它顯示登入者資料。

稽核時發現三件事：

1. 有人在檢舉備註欄填了一段 HTML，alice 打開 `/admin` 時它被瀏覽器當成程式執行。
2. alice 逛了另一個網站，回來發現自己的短網址被刪光了，而她沒按過任何刪除鈕。
3. `GET /me` 的回應裡有 `password_hash`，後端工程師說「前端又不會顯示它」。

這三件事有共同的根源嗎？後端各要在哪一層擋？

<details>
<summary>提示</summary>

三件事都是「信任了不該信任的東西」。想想：瀏覽器怎麼知道一段文字是資料還是程式？瀏覽器自動附上 Cookie 的時候，伺服器怎麼知道請求是不是使用者本意？一個回應裡，誰決定要放哪些欄位？

</details>

## 是什麼

這三個漏洞是 OWASP 長年列在 Web 應用風險清單前段的常見類別：

- 跨站腳本（cross-site scripting，XSS）：攻擊者的 JavaScript 在你網站的頁面裡執行。
- 跨站請求偽造（cross-site request forgery，CSRF）：別的網站讓使用者的瀏覽器，帶著使用者的登入憑證，對你的網站送出非本意的請求。
- 敏感資料外洩（sensitive data exposure）：不該被看到的資料，經由回應、錯誤訊息、日誌等路徑離開了伺服器。

本篇只做入門：認得它們、知道後端那一側的最小防禦。攻擊怎麼重現、各種變形與繞過，在資安領域有完整的一篇，文中會逐一指過去。

## 為什麼存在

三個漏洞的根源都是「信任邊界」畫錯了：

- XSS：程式把使用者輸入的字串，直接拼進 HTML。瀏覽器分不出哪段是你寫的版面、哪段是別人填的資料。
- CSRF：瀏覽器會依「目的地」自動附上 Cookie，不管請求是從哪個頁面發出的。這叫環境權限（ambient authority），伺服器只看到「有登入的 Cookie」，看不出使用者有沒有真的想做這件事。
- 敏感資料外洩：程式把「內部的資料結構」直接當成「對外的回應」，新增一個欄位就自動多洩一個欄位。

沒有防禦的後果很具體。2005 年 MySpace 的 Samy 蠕蟲靠一段儲存型 XSS，在約 20 小時內讓上百萬名使用者自動加了同一個人為好友（見延伸閱讀）。這三類漏洞也都不需要攻擊者碰到你的伺服器：XSS 與 CSRF 發生在受害者的瀏覽器裡。

## 怎麼運作

:::caution
本篇的攻擊示範只針對你在本機自己跑的範例程式（`127.0.0.1`）。只能在自己擁有或獲得授權的系統上測試，不要拿來測別人的網站。
:::

範例放在 `playground/backend/l1/1-4-auth-basics/common-vulnerabilities/`：同一組路由有 `Vulnerable()` 與 `Safe()` 兩個版本，測試同時驗證「攻擊在前者成功、在後者失敗」。

### XSS：把資料當成資料

有漏洞的版本把備註用 `fmt.Fprintf` 拼進 HTML：

```go title="xss.go"
func reportUnsafe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<!doctype html><p>備註：%s</p>", r.URL.Query().Get("note"))
}
```

備註若是 `<img src=x onerror=alert(1)>`，瀏覽器會建立一個圖片元素、載入失敗、執行 `onerror` 裡的程式。這段程式跑在你網站的 origin 裡，能讀頁面、能用 alice 的身分呼叫 API。

修法是輸出編碼（output encoding）：資料放進 HTML 時，把 `<`、`>`、`&`、引號換成實體，讓它只能是文字。Go 的 `html/template` 會依位置自動做：

```go title="xss.go"
var reportTmpl = template.Must(template.New("report").
	Parse("<!doctype html><p>備註：{{.}}</p>"))

// ...
w.Header().Set("Content-Security-Policy", "default-src 'none'")
```

同一個 payload 現在輸出 `&lt;img src=x onerror=alert(1)&gt;`，顯示成文字。下面那行 CSP 標頭是第二道防線：就算哪裡漏了編碼，瀏覽器也不會執行這頁上的腳本。

兩個重點：編碼要在「輸出的當下」做，而且要依位置（HTML 本文、屬性、URL、JavaScript）選對方法；`html/template` 的好處是你不必自己選。`HttpOnly` Cookie 只能防腳本讀走 session ID，擋不住腳本直接替使用者發請求，所以不能代替編碼。完整的三種 XSS 與 CSP nonce 見資安領域的 [XSS](/cyber/p2-web/2-3-cross-site/xss/)。

### CSRF：問瀏覽器請求從哪來

攻擊者在 `evil.test` 放一個自動送出的表單，目標是你的 `POST /links/delete`。alice 的瀏覽器照規矩附上她的 Cookie，後端看到「有效的 session」就刪了。

```text
alice 的瀏覽器                evil.test            短網址服務
      │  開啟惡意頁面            │                      │
      ├─────────────────────────▶│                      │
      │  ◀── 自動送出的表單 ─────┤                      │
      │  POST /links/delete  + Cookie: sid=...          │
      ├────────────────────────────────────────────────▶│
      │                          │        Cookie 有效 → 刪除
```

圖中的重點：請求的 Cookie 是真的，但「想刪除」不是 alice 的意思，伺服器只靠 Cookie 分不出來。

後端的最小防禦有兩層。第一層是 Cookie 的 `SameSite=Lax`（見 [Session 與 Cookie](/backend/l1/1-4-auth-basics/session-cookie/)），讓跨站的 POST 與 `fetch` 不附帶 Cookie。第二層是在伺服器上檢查請求來源。Go 1.25 起標準函式庫內建 `http.CrossOriginProtection`：

```go title="app.go"
// Rejects cross-site POST/PUT/DELETE; GET is never checked.
return http.NewCrossOriginProtection().Handler(mux)
```

它依瀏覽器送的 `Sec-Fetch-Site` 標頭（沒有時改比對 `Origin` 與 `Host`）擋下跨站的不安全方法。測試 `TestCSRFDelete` 送出帶 `Origin: http://evil.test`、`Sec-Fetch-Site: cross-site` 的 POST：有漏洞的版本回 204、連結被刪；`Safe()` 回 403、連結還在。同源的請求照常通過。

注意註解那句：GET 完全不檢查，所以 **會改變狀態的操作絕不能用 GET**。CSRF token、盲點與繞過見資安領域的 [CSRF](/cyber/p2-web/2-3-cross-site/csrf/)。

### 敏感資料外洩：回應只放你列出的欄位

有漏洞的 `/me` 直接把資料庫模型編成 JSON：

```go title="exposure.go"
s.writeJSON(w, s.user)
```

`User` 有 `PasswordHash`，所以它也出去了。修法是為對外回應另外定義一個型別，明確列出可以給的欄位：

```go title="exposure.go"
type meView struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
}
```

這是 **白名單** 的思路：新增欄位預設不會洩漏，要放進回應得有人主動改 `meView`。反過來，用 `json:"-"` 一個個遮欄位是黑名單，漏掉一個就洩一個。

第二條外洩路徑是錯誤訊息。範例的 `lookup` 模擬資料庫失敗，錯誤文字帶著查詢語句和內部位址 `10.0.3.7:5432`。有漏洞的版本把 `err.Error()` 原樣回給客戶端；安全的版本把細節寫進日誌，只回「lookup failed」：

```go title="exposure.go"
s.log.Error("lookup", "err", err)
http.Error(w, errLookup.Error(), http.StatusInternalServerError)
```

第三條是日誌本身：密碼、token、完整的 `Authorization` 標頭都不該寫進日誌。這部分在 L2 的日誌與審計相關章節再展開。

### 回答先想想的問題

回到一開始的問題：三件事的共同根源是邊界畫錯了。XSS 是把「資料」當成「程式」，後端要在輸出時編碼；CSRF 是把「瀏覽器自動附上的憑證」當成「使用者的意圖」，後端要檢查請求來源並讓 GET 保持唯讀；`password_hash` 是把「內部模型」當成「對外回應」，後端要用明確列出欄位的回應型別。「前端不會顯示它」不是防禦，任何人都能直接看網路回應。

## 取捨與替代方案

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| `html/template` 自動編碼 | 依位置自動選對編碼，幾乎不用想 | 只管 Go 輸出的 HTML；前端用 `innerHTML` 寫入的不在內 | 伺服器端渲染頁面 |
| CSP | 編碼漏掉時仍能擋腳本 | 要調整既有的行內腳本；設太鬆等於沒有 | 當第二道防線，不單獨依賴 |
| `SameSite` Cookie | 一行設定，瀏覽器代勞 | 只擋跨站，擋不了同站的其他子網域；舊瀏覽器行為不一 | 所有用 Cookie 的網站 |
| `CrossOriginProtection` | 標準庫，不用管理 token | 不檢查 GET；非瀏覽器客戶端不被它保護 | 瀏覽器為主的 API |
| CSRF token | 不依賴瀏覽器標頭 | 要發放、綁定 session、驗證 | 需要相容舊瀏覽器，或要多一層防禦 |
| 對外 DTO（回應專用型別） | 新欄位預設不外洩 | 多寫一個型別與轉換 | 所有對外 API |

什麼時候這些做法不夠：

- 輸出到 HTML 以外的地方（JavaScript 字串、URL、CSS）時，HTML 編碼不適用，要用對應位置的方法；這已超出入門，見資安領域。
- 使用 `Authorization: Bearer` 標頭而非 Cookie 的 API，不受 CSRF 影響（瀏覽器不會自動附上它），但 token 存在 `localStorage` 時就更怕 XSS。
- 這三項只是入門清單，不是完整的安全審查：SQL injection（見 1.3）、授權檢查缺失、依賴套件的漏洞都不在內。

## 常見坑與故障案例

### 坑 1：以為用了 `html/template` 就一定安全

症狀：程式碼審查通過了「有用模板」，頁面仍被注入腳本。

原因：把使用者輸入轉成 `template.HTML`（或 `template.JS`、`template.URL`）等於告訴模板「這段已經安全，不要編碼」，自動編碼因此被關掉。

避免：不要把來自使用者的字串轉成這些型別。真的需要允許部分 HTML 時，先用專門的 sanitizer 處理，並把它的結果限制在單一位置。

### 坑 2：用 GET 改變狀態

症狀：Safe 版本已加上 `CrossOriginProtection`，攻擊者仍能放一個連結或轉址到 `https://你的網站/links/delete?slug=go`，alice 一點就刪掉連結。

原因：標準函式庫與大多數防禦只檢查不安全的方法；GET 被視為唯讀所以放行。把刪除做成 GET，就等於繞過整套防禦；`SameSite=Lax` 也放行這種頂層導覽的 GET。

避免：所有會改變狀態的路由用 POST、PUT、DELETE；GET 只讀取。

### 坑 3：把資料庫模型直接編成 JSON

症狀：新增一個欄位（例如 `APIToken`）之後，所有回傳該模型的 API 同時多洩一個欄位，沒有人注意到。

原因：回應的欄位由資料模型決定，而不是由「該給誰看什麼」決定。

避免：每個對外回應用獨立的型別，欄位逐一列出。測試時對回應內容做斷言，確認敏感欄位不在其中。

### 坑 4：錯誤訊息回傳內部細節

症狀：攻擊者故意送壞請求，從回應拿到資料表名稱、查詢語句、內部 IP，用來規劃下一步攻擊。

原因：偷懶用 `http.Error(w, err.Error(), 500)`，把給開發者看的訊息給了客戶端。

避免：客戶端只拿到泛用訊息（可附一個請求 ID 方便對照），詳細錯誤寫進日誌。

### 坑 5：前端隱藏欄位當作防禦（示意案例）

症狀：管理員專用的欄位只是在前端 `if (isAdmin)` 才顯示，一般使用者打開瀏覽器開發者工具，在網路分頁就看到完整回應。

原因：回應已經離開伺服器，前端的任何隱藏都只是外觀。

避免：由後端決定回應內容，一般使用者的請求就不要放進管理員欄位。

## 小練習

範例放在 `playground/backend/l1/1-4-auth-basics/common-vulnerabilities/`，執行 `go test ./...` 可以驗證。想看它動起來，執行 `go run .`（安全版）或 `go run . -vulnerable`（有漏洞版），再用 `curl` 試：

```bash
curl --cookie 'sid=alice-session' http://127.0.0.1:8080/me
```

### 動手寫

在 `app/app.go` 的 `User` 新增 `APIToken string` 欄位（`json:"api_token"`），並在 `NewServer` 給 alice 一個值。先不要改 `meView`。寫一個測試 `TestTokenNeverLeaks`：對 `Safe()` 的 `/me` 送請求，斷言回應不含這個 token。再寫另一個斷言，確認 `Vulnerable()` 的 `/me` 現在多洩了這個 token。

<details>
<summary>參考解答</summary>

測試會通過，不需要改任何處理程式：`meView` 是白名單，新欄位沒被列進去，所以不會出現在安全版的回應。有漏洞的版本直接編碼 `User`，token 自動出現。

這正是白名單的價值：新增欄位時預設是安全的。如果 `meView` 改成黑名單（例如在 `User` 上為 `PasswordHash` 加 `json:"-"`，其他欄位照舊），新增 `APIToken` 就會自動外洩，除非有人記得也替它加標記。

</details>

### 辨識

下面是 AI 產生的路由設定。正常請求下全部行為正確，也有 `CrossOriginProtection` 保護。請做 code review，找出 3 個與本文相關的問題。

```go title="handlers.go"
mux.HandleFunc("GET /report", func(w http.ResponseWriter, r *http.Request) {
	note := r.URL.Query().Get("note")
	page.Execute(w, template.HTML(note))
})
mux.HandleFunc("GET /links/delete", func(w http.ResponseWriter, r *http.Request) {
	s.deleteLink(r.URL.Query().Get("slug"))
	w.WriteHeader(http.StatusNoContent)
})
mux.HandleFunc("GET /links/lookup", func(w http.ResponseWriter, r *http.Request) {
	if err := s.lookup(r.URL.Query().Get("slug")); err != nil {
		http.Error(w, "lookup failed: "+err.Error(), http.StatusInternalServerError)
	}
})
return http.NewCrossOriginProtection().Handler(mux)
```

完整檔案在 `recognize/handlers.go`（AI 產生，不參與測試）。

<details>
<summary>參考解答</summary>

1. **`page.Execute(w, template.HTML(note))`**：把使用者輸入轉成 `template.HTML`，模板就不再編碼，備註裡的 `<img onerror=...>` 原樣輸出，等於沒用模板。正確寫法：直接傳字串 `note`，讓模板編碼。
2. **`GET /links/delete`**：刪除用了 GET。`CrossOriginProtection` 不檢查 GET，攻擊者只要讓 alice 點一個連結或被轉址到 `/links/delete?slug=go`，頂層導覽的 GET 就會帶著 `SameSite=Lax` 的 Cookie 送出。正確寫法：改成 `POST /links/delete`。
3. **`"lookup failed: "+err.Error()`**：把內部錯誤回給客戶端，洩漏查詢語句與內部位址。正確寫法：回泛用訊息，細節用 `slog` 寫進日誌。

這份程式碼對正常請求都表現正確，只測「正常流程」的測試看不出問題。

</details>

## 快閃卡

<details>
<summary>XSS 發生的根本原因是什麼？</summary>

程式把使用者輸入的字串直接拼進 HTML，瀏覽器分不出哪段是版面、哪段是資料，就把注入的內容當成程式執行。

</details>

<details>
<summary>CSRF 為什麼能成功？伺服器看到的是什麼？</summary>

瀏覽器依目的地自動附上 Cookie，不管請求從哪個頁面發出。伺服器只看到一個有效的登入 Cookie，分不出使用者是不是真的想做這件事。

</details>

<details>
<summary>回應裡的欄位由資料模型決定與由專用回應型別決定，差在哪？</summary>

由資料模型決定時，新增欄位會自動出現在回應裡；由專用型別決定是白名單，新欄位要有人主動加進型別才會出去，預設是安全的。

</details>

<details>
<summary>頁面用 html/template 輸出 {{.}}，備註是 &lt;script&gt;alert(1)&lt;/script&gt;，瀏覽器會執行它嗎？</summary>

不會。html/template 依位置自動編碼，把 &lt; 與 &gt; 換成實體，瀏覽器把它當作文字顯示。

</details>

<details>
<summary>Session Cookie 已設 HttpOnly，還需要輸出編碼嗎？</summary>

需要。HttpOnly 只讓腳本讀不到 Cookie，不能阻止腳本在使用者頁面裡直接發請求，所以不能取代輸出編碼。

</details>

<details>
<summary>伺服器掛了 CrossOriginProtection。請求 POST /links/delete 帶 Sec-Fetch-Site: cross-site，結果是什麼？GET /links/delete?slug=go 呢？</summary>

POST 會被擋下，回 403。GET 屬於安全方法，不會被檢查，所以若刪除放在 GET 上就不受保護。

</details>

<details>
<summary>為什麼改狀態的操作不能用 GET？</summary>

CSRF 的標準防禦只檢查 POST、PUT、DELETE 等不安全方法，GET 預設放行。刪除做成 GET，攻擊者只要讓使用者點一個連結就能觸發，繞過整套防禦。

</details>

<details>
<summary>資料庫查詢失敗，handler 該把 err.Error() 回給客戶端嗎？</summary>

不該。錯誤文字常含查詢語句與內部位址。客戶端只回泛用訊息，細節寫進日誌給維運人員。

</details>

## 延伸閱讀

- [XSS：反射、儲存、DOM 型與對應修補](/cyber/p2-web/2-3-cross-site/xss/)：資安領域的完整一篇，重現三種 XSS 並用編碼、CSP、安全的 DOM 寫法修補。
- [CSRF：原理與 token／SameSite 修補](/cyber/p2-web/2-3-cross-site/csrf/)：兩種偽造方式與三道防線各自的盲點。
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)：Web 應用最常見風險的分類，本篇三項都在其中。
- [net/http 的 CrossOriginProtection](https://pkg.go.dev/net/http#CrossOriginProtection)：Go 標準庫的 CSRF 防禦，說明檢查規則與例外設定。
- [Samy（電腦蠕蟲）（Wikipedia）](https://en.wikipedia.org/wiki/Samy_(computer_worm))：2005 年 MySpace 儲存型 XSS 蠕蟲的經過。
