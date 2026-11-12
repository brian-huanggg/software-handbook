---
title: CORS：原理、預檢請求與在 Go 中正確設定
description: 看懂 CORS 錯誤從哪來，並用白名單在 Go 裡安全地開放跨源讀取。
sidebar:
  order: 6
---

## 先想想

短網址服務 v1 的 API 在 `https://api.sho.rt`，管理後台另外部署在 `https://app.sho.rt`。後台登入後用 `fetch` 加上 `credentials: "include"` 帶著 Cookie 呼叫 API，主控台卻出現「blocked by CORS policy」。

同事在 Go 裡加了一行 `w.Header().Set("Access-Control-Allow-Origin", "*")`。讀取清單還是失敗，刪除短網址更奇怪：伺服器的 log 裡出現一筆 `OPTIONS /api/links/todo`，回應是 405，而你根本沒寫過 `OPTIONS` 的 handler。

為什麼 `*` 沒有用？那筆 `OPTIONS` 請求是誰送的？

<details>
<summary>提示</summary>

想想 [同源政策](/cs/n-network/n-4-application/same-origin-policy/) 擋的是什麼，再想想帶著 Cookie 的請求如果對任何網站都放行，會發生什麼事。

</details>

## 是什麼

**跨源資源共用（Cross-Origin Resource Sharing，CORS）** 是一組 HTTP header，讓伺服器告訴瀏覽器：「這些 origin 的網頁可以讀我的回應。」

它是同源政策的 **放寬** ，不是一道防線。沒有任何 CORS header 時，瀏覽器維持同源政策的預設行為；伺服器加上 CORS header，等於替特定 origin 開一扇門。

## 為什麼存在

同源政策讓瀏覽器預設不交出跨源回應。這保護了使用者，但前後端分開部署、API 給合作夥伴的網頁使用，都是很正常的需求。

在 CORS 之前，大家只能繞路：用 JSONP 讓對方的回應以 `<script>` 的形式執行，或在自己的網域架一個代理轉發請求。前者等於讓對方在你的頁面執行任意程式碼，後者每個前端都要多維護一層。

CORS 把「誰可以讀」的決定交給擁有資料的伺服器，用標準的 header 表達，瀏覽器負責執行。現在的規則寫在 WHATWG 的 Fetch Standard 裡，所有主流瀏覽器都照它實作。

## 怎麼運作

### 簡單請求：先送出，再決定給不給看

瀏覽器把跨源請求分成兩種。符合以下全部條件的叫 **簡單請求（simple request）** ：

- method 是 `GET`、`HEAD` 或 `POST`
- 只帶瀏覽器允許的 header（例如 `Accept`、`Content-Language`）
- `Content-Type` 只能是 `application/x-www-form-urlencoded`、`multipart/form-data` 或 `text/plain`

簡單請求就是一般表單本來就能送出的請求，所以瀏覽器直接送，在 `Origin` header 寫上發出請求的網頁 origin。伺服器在回應裡加上 `Access-Control-Allow-Origin`，瀏覽器比對它和網頁的 origin，相符才把回應交給 JavaScript。

這表示伺服器的 handler **一定會執行** 。CORS 決定的只是網頁讀不讀得到結果。

### 預檢請求：先問，再送

送 JSON（`Content-Type: application/json`）、用 `DELETE` 或 `PUT`、帶 `Authorization` header，都不符合簡單請求的條件。這些請求是表單做不到的，舊伺服器可能假設它們只會來自同源網頁，所以瀏覽器會先發一個 **預檢請求（preflight request）** 問清楚：

```text
app.sho.rt 的網頁         瀏覽器                         api.sho.rt
      │                     │                                 │
      │ fetch DELETE        │ OPTIONS /api/links/todo         │
      ├────────────────────►│ Origin: https://app.sho.rt      │
      │                     │ Access-Control-Request-Method:  │
      │                     │   DELETE                        │
      │                     │（不帶 Cookie）                   │
      │                     ├────────────────────────────────►│
      │                     │ 204                             │
      │                     │ Access-Control-Allow-Origin     │
      │                     │ Access-Control-Allow-Methods    │
      │                     │◄────────────────────────────────┤
      │                     │ 檢查通過，才送出真正的請求        │
      │                     │ DELETE /api/links/todo + Cookie │
      │                     ├────────────────────────────────►│
```

這張圖要看的是中間那條分界：預檢沒通過，真正的 `DELETE` 根本不會送出。這和簡單請求不同，簡單請求不管伺服器怎麼回，都已經到了伺服器。

預檢的回應可以用 `Access-Control-Max-Age` 告訴瀏覽器快取多久，不必每次都先問。沒有這個 header 時，MDN 記載的預設值是 5 秒。

### 帶 Cookie 的請求

`fetch` 預設不在跨源請求上帶 Cookie。前端寫了 `credentials: "include"` 後，伺服器要同時滿足兩件事，瀏覽器才交出回應：

- `Access-Control-Allow-Origin` 寫出明確的 origin，**不能是 `*`**
- 加上 `Access-Control-Allow-Credentials: true`

`*` 的意思是「任何網頁都能讀」。如果它也能搭配 Cookie，任何網站都能用你的登入身分讀你的資料，同源政策就白費了。所以規格直接禁止這個組合。

### 在 Go 裡實作白名單

做法是準備一份允許的 origin 清單，請求的 `Origin` 在清單裡，就把 **同一個值** 寫回 `Access-Control-Allow-Origin`：

```go title="cors.go"
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The answer depends on Origin, so a cache must not reuse it
			// for another origin, even when this one is refused.
			w.Header().Add("Vary", "Origin")
			origin := r.Header.Get("Origin")
			preflight := r.Method == http.MethodOptions &&
				r.Header.Get("Access-Control-Request-Method") != ""
			if !allowed[origin] {
				if preflight {
					// No CORS headers: the browser will not send the real request.
					w.WriteHeader(http.StatusForbidden)
					return
				}
				// Not our call to block: same-origin pages and curl send no
				// Origin, and a simple request has already arrived anyway.
				next.ServeHTTP(w, r)
				return
			}
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			if p.Credentials {
				h.Set("Access-Control-Allow-Credentials", "true")
			}
			if !preflight {
				next.ServeHTTP(w, r)
				return
			}
			// Preflights carry no cookies, so they are answered here,
			// before the auth middleware and the mux can reject them.
			h.Set("Access-Control-Allow-Methods", methods)
			h.Set("Access-Control-Allow-Headers", headers)
			h.Set("Access-Control-Max-Age", maxAge)
			w.WriteHeader(http.StatusNoContent)
		})
```

`allowed` 是一個 map，比對是整串字串完全相等。瀏覽器送出的 `Origin` 一定是正規化後的形式（小寫、沒有路徑、省略預設 port），所以設定檔也必須寫成這個形式。範例在啟動時用 `checkOrigin` 檢查，`https://app.sho.rt/`（多一個斜線）、`*`、`null` 都會直接報錯，不會等到上線才發現永遠比對不到。

`Vary: Origin` 告訴快取（瀏覽器、CDN）：這個回應的內容會隨 `Origin` 不同而改變，不能拿 A 網站的回應給 B 網站用。

### middleware 的順序

```go title="server.go"
	api := withCORS(requireSession(newMux(store, logger)))
```

CORS 必須包在最外層。預檢請求不帶 Cookie，如果先經過 `requireSession`，它會拿到 401；如果直接進 `ServeMux`，`DELETE /api/links/{code}` 這個路徑沒有註冊 `OPTIONS`，會拿到 405。範例的 `TestMuxAloneRejectsPreflight` 重現了後者。

回到一開始的問題：`*` 沒有用，是因為前端帶了 Cookie，瀏覽器規定這時必須寫出明確的 origin 並加上 `Access-Control-Allow-Credentials: true`。那筆 `OPTIONS` 是瀏覽器替 `DELETE` 發的預檢請求，它沒有被任何 CORS 程式處理，直接掉進 mux 拿到 405，瀏覽器就放棄送出真正的 `DELETE`。修法是把白名單 middleware 放在最外層，讓它回答預檢。

## 取捨與替代方案

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| 讓前端與 API 同源（反向代理把 `/api` 轉給後端） | 完全不需要 CORS，Cookie 行為最單純 | 部署要多一層代理 | 自己的前端呼叫自己的 API |
| 自己寫白名單 middleware | 行為完全看得懂，沒有依賴 | 要自己顧好規格細節與測試 | 需要的規則不多，例如本書範例 |
| 使用成熟的 CORS 套件（例如 `github.com/rs/cors`） | 規格細節處理得較完整 | 多一個依賴，設定錯誤一樣會開出漏洞 | 規則多、多個服務共用設定 |
| `Access-Control-Allow-Origin: *` 且不帶 Cookie | 設定最簡單 | 任何網頁都讀得到 | 真正公開、不需登入的資料，例如公開的短網址統計 |

什麼時候不該用 CORS：

- **前端和 API 能同源時。** 用反向代理讓它們同源，就沒有預檢的延遲，也沒有設定出錯的機會。
- **想拿它當存取控制時。** CORS 只影響瀏覽器交不交出回應，`curl` 完全不受限，簡單請求也照樣會執行。誰能讀、誰能刪，仍然要靠認證與授權。
- **API 沒有用 Cookie 時，不要打開 `Credentials`。** 用不到的權限就不要給。

## 常見坑與故障案例

### 把收到的 Origin 原封不動寫回去

**症狀**：為了「讓所有前端都能用」，伺服器把 `r.Header.Get("Origin")` 直接寫進 `Access-Control-Allow-Origin`，再加上 `Access-Control-Allow-Credentials: true`。一切都能跑，也沒有錯誤。

**原因**：這等於對所有網站說「你可以帶著使用者的 Cookie 讀我的資料」。PortSwigger 的研究者在 2016 年發表的文章中指出，這是最常見的 CORS 漏洞，並用它從一個比特幣交易所讀出使用者的私人 API 金鑰。

**對策**：只回寫白名單裡的 origin。攻擊的完整過程與驗證，會在資安領域 P2 2.3 示範。

### 用前綴或後綴比對 origin

**症狀**：程式用 `strings.HasSuffix(origin, "sho.rt")` 判斷可信來源，`https://evilsho.rt` 也通過了；改用 `HasPrefix(origin, "https://sho.rt")`，`https://sho.rt.evil.example` 又通過了。

**原因**：同一篇 PortSwigger 研究記錄了真實網站的這兩種錯誤。origin 要整串相等，不能部分比對。

**對策**：用 map 或 `slices.Contains` 整串比對。需要支援很多子網域時，先解析出 host，再比對 `host == "sho.rt" || strings.HasSuffix(host, ".sho.rt")`，注意前面的點。

### 把 `null` 放進白名單

**症狀**：本機開發時從 `file:` 打開頁面，`Origin` 是 `null`，為了方便就把 `null` 加進白名單。

**原因**：任何網站都能用帶 `sandbox` 屬性的 iframe 讓請求的 `Origin` 變成 `null`，信任 `null` 等於信任所有人。同一篇研究提到 `docs.google.com` 的一個端點就曾回傳 `Access-Control-Allow-Origin: null` 並允許帶 Cookie。

**對策**：本機開發改用 `http://localhost:<port>` 起一個伺服器。範例的 `checkOrigin` 遇到 `null` 會直接拒絕啟動。

### 忘了 `Vary: Origin`，快取把回應給錯人

**症狀**：API 前面有 CDN。後台和合作夥伴的頁面輪流出現 CORS 錯誤，重新整理幾次又好了。

**原因**：伺服器依 `Origin` 回寫不同的 `Access-Control-Allow-Origin`，但沒有加 `Vary: Origin`。CDN 把寫著 A 網站 origin 的回應快取起來，再交給 B 網站，B 的瀏覽器當然拒絕。

**對策**：只要回應會隨 `Origin` 改變就加上 `Vary: Origin`，被拒絕的回應也要加，範例的 `TestOtherOriginStillReachesHandler` 會檢查這件事。（示意案例）

## 小練習

範例程式在 `playground/golang/backend/l1/1-4-auth-basics/cors/`。先執行 `go test ./...` 確認全部通過。

**1. 讓前端讀得到自訂 header。** API 想在回應裡加 `X-Request-ID`，但跨源的 JavaScript 預設只能讀到少數幾個 header。替 `cors.Policy` 加一個 `Exposed []string` 欄位，在非預檢的回應寫出 `Access-Control-Expose-Headers`，並在 `cors/cors_test.go` 加一個測試驗證。

<details>
<summary>參考解答</summary>

在 `New` 裡先算好 `exposed := strings.Join(p.Exposed, ", ")`，在 `if !preflight {` 區塊內、呼叫 `next.ServeHTTP` 之前加上：

```go
if exposed != "" {
	h.Set("Access-Control-Expose-Headers", exposed)
}
```

測試用允許的 origin 發一個 `GET`，檢查回應的 `Access-Control-Expose-Headers` 是 `X-Request-ID`。它只需要出現在真正的回應上，預檢回應不用。

</details>

**2. 證明 CORS 不是存取控制。** 在 `server_test.go` 寫一個測試：用不在白名單的 origin `http://127.0.0.1:9191`、帶著 Cookie 發 `GET /api/links`。預測狀態碼和 `Access-Control-Allow-Origin`，再執行測試。

<details>
<summary>參考解答</summary>

```go
func TestOtherOriginGetStillServed(t *testing.T) {
	api, _ := newTestAPI(t)
	rec := request(api, "GET", "/api/links", "http://127.0.0.1:9191", true, nil)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q, want none", got)
	}
}
```

狀態碼是 200，回應本體也有完整清單，只是沒有 `Access-Control-Allow-Origin`。瀏覽器會因此不交給網頁，但 `curl` 或任何非瀏覽器程式都拿得到。保護資料的是 `requireSession`，不是 CORS。

</details>

**3. 在瀏覽器裡對照兩個前端。** 執行 `go run .`，先打開 `http://127.0.0.1:8080/login` 取得 Cookie，再分別打開 `http://127.0.0.1:9090/` 與 `http://127.0.0.1:9191/`，各按一次「讀取」與「刪除 todo」。對照終端機的 log，說明兩個前端的刪除結果為什麼不同，以及這和 [同源政策](/cs/n-network/n-4-application/same-origin-policy/) 的範例有什麼差別。

<details>
<summary>參考解答</summary>

`9090` 在白名單裡：讀取顯示清單，刪除回應 204，log 先後出現預檢與 `DELETE`。`9191` 不在白名單：讀取在 log 裡看得到 `GET`，頁面卻讀不到；刪除只會留下被拒絕的預檢，log 沒有 `deleted "todo"`，`todo` 還在。

同源政策那篇的刪除是表單格式的 `POST`，屬於簡單請求，不經預檢就送到伺服器，所以真的刪掉了。這裡的 `DELETE` 需要預檢，預檢被拒，真正的請求就不會送出。

</details>

### 辨識

下面這段程式碼由 AI 產生，目的是「只讓 sho.rt 自己的網頁帶著 Cookie 跨源讀取 API」。它能編譯，用 `https://app.sho.rt` 的頁面測試讀取與刪除也都正常。請做 code review，找出它的問題（至少兩個）。

```go title="cors.go"
// withCORS lets pages from sho.rt read the API with their cookies.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if strings.HasSuffix(origin, "sho.rt") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

完整程式碼在 `recognize/cors.go`，它有 `//go:build ignore`，不會被 `go test ./...` 掃到。

<details>
<summary>參考解答</summary>

1. **`strings.HasSuffix(origin, "sho.rt")`**：`https://evilsho.rt` 也以 `sho.rt` 結尾，會通過檢查。攻擊者註冊這樣的網域，就能讓已登入使用者的瀏覽器帶著 Cookie 讀走 API 的回應，因為這裡同時回了 `Access-Control-Allow-Credentials: true`。正確寫法是用 map 整串比對：`allowed[origin]`，白名單寫 `https://app.sho.rt`。
2. **沒有 `Vary: Origin`**：回應會隨 `Origin` 改變，卻沒有告訴快取。API 前面有 CDN 時，A 網站的回應可能被快取後交給 B 網站，B 的瀏覽器就拒絕讀取，症狀是時好時壞的 CORS 錯誤。正確寫法是在函式開頭 `w.Header().Add("Vary", "Origin")`，不在白名單的請求也要加。

</details>

## 快閃卡

<details>
<summary>CORS 是保護伺服器的機制，還是放寬同源政策的機制？</summary>

是放寬。沒有 CORS header 時，瀏覽器維持同源政策；伺服器用 CORS header 告訴瀏覽器哪些 origin 可以讀它的回應。

</details>

<details>
<summary>在 CORS 出現之前，前端要讀跨源 API 通常怎麼做？各有什麼問題？</summary>

用 JSONP 或在自己的網域架代理。JSONP 等於讓對方在你的頁面執行任意程式碼，代理則讓每個前端都多一層要維護。

</details>

<details>
<summary>fetch 送出 POST，Content-Type 是 application/json。瀏覽器會先發預檢嗎？</summary>

會。簡單請求的 `Content-Type` 只能是 `application/x-www-form-urlencoded`、`multipart/form-data` 或 `text/plain`，JSON 不在其中。

</details>

<details>
<summary>簡單請求被 CORS 擋下，和預檢請求被拒絕，對伺服器來說差在哪？</summary>

簡單請求已經送到伺服器，handler 照常執行，只是回應不交給網頁。預檢被拒時，瀏覽器不會送出真正的請求，handler 不會執行。

</details>

<details>
<summary>前端用 <code>credentials: "include"</code>，伺服器回 <code>Access-Control-Allow-Origin: *</code>。網頁讀得到回應嗎？</summary>

讀不到。帶 Cookie 時必須寫出明確的 origin 並加上 `Access-Control-Allow-Credentials: true`；否則任何網站都能用使用者的身分讀資料。

</details>

<details>
<summary>伺服器 log 出現 <code>OPTIONS /api/links/todo</code> 回應 401，前端刪除失敗。可能的原因是什麼？</summary>

預檢請求不帶 Cookie，卻先經過了認證 middleware。CORS middleware 要放在最外層，在認證之前回答預檢。

</details>

<details>
<summary>為什麼回寫白名單裡的 origin 時要加上 <code>Vary: Origin</code>？</summary>

因為回應內容會隨 `Origin` 改變。少了它，瀏覽器或 CDN 可能把寫著 A 網站 origin 的回應快取起來交給 B 網站，B 就會遇到 CORS 錯誤。

</details>

<details>
<summary>白名單寫成 <code>https://app.sho.rt/</code>，為什麼永遠比對不到？</summary>

瀏覽器送出的 `Origin` 沒有路徑，是 `https://app.sho.rt`。整串比對時多出的斜線讓兩者不相等，所以設定要寫成正規化的形式。

</details>

<details>
<summary>伺服器把收到的 Origin 原封不動寫回，並允許帶 Cookie，會發生什麼事？</summary>

任何網站都能讓已登入使用者的瀏覽器帶著 Cookie 呼叫 API，並讀走回應。這是最常見的 CORS 漏洞，應改成只回寫白名單裡的 origin。

</details>

<details>
<summary>開發時把 <code>null</code> 加進 CORS 白名單，為什麼危險？</summary>

任何網站都能用帶 `sandbox` 的 iframe 讓請求的 `Origin` 變成 `null`，信任 `null` 等於信任所有人。

</details>

## 延伸閱讀

- [Cross-Origin Resource Sharing (CORS)（MDN）](https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CORS)：簡單請求條件、預檢、帶 Cookie 的規則與各個 header 的完整說明。
- [Fetch Standard：CORS protocol](https://fetch.spec.whatwg.org/#http-cors-protocol)：瀏覽器實作 CORS 所依據的規格，遇到 MDN 沒寫清楚的細節時查這裡。
- [Exploiting CORS misconfigurations for Bitcoins and bounties（PortSwigger）](https://portswigger.net/research/exploiting-cors-misconfigurations-for-bitcoins-and-bounties)：反射 origin、`null`、前後綴比對三種錯誤設定的真實案例，資安領域 P2 2.3 的前導閱讀。
