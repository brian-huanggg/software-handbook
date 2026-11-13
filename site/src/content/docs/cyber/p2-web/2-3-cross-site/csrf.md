---
title: CSRF：原理與 token／SameSite 修補
description: 在自己的短網址服務上重現跨站請求偽造，再用 SameSite、token 與 Fetch Metadata 修好並驗證。
sidebar:
  order: 2
---

## 先想想

你在稽核自己寫的短網址服務 v1。alice 登入後可以在設定頁改自己短連結 `/a1` 指向哪裡，伺服器靠 Cookie 認出她。現在她在另一個分頁打開朋友傳來的「流量報告」，網址是 `http://evil.test:9666`。

那個頁面沒有讀到她的任何資料，同源政策也確實擋住了跨源讀取。可是幾秒後，`/a1` 悄悄指向了一個釣魚頁。

evil.test 連 victim.test 的回應都讀不到，它是怎麼改動 alice 的短連結的？伺服器為什麼會當成是 alice 本人操作？

<details>
<summary>提示</summary>

想想 [同源政策](/cs/n-network/n-4-application/same-origin-policy/) 擋的是什麼：它擋的是跨源「讀取」回應。那「送出」請求呢？Cookie 是依什麼條件被附上的？

</details>

## 是什麼

**跨站請求偽造（cross-site request forgery，CSRF）** 是攻擊者讓受害者已登入的瀏覽器，對目標網站送出一個受害者沒打算送的請求，並由瀏覽器自動附上受害者的憑證，讓伺服器誤以為是本人操作。

它和上一篇的 [CORS 錯誤設定](/cyber/p2-web/2-3-cross-site/cors-misconfig/) 是一對：CORS 錯誤設定是攻擊者「讀」到不該讀的資料，CSRF 是攻擊者「借用」受害者的身分「寫」。它濫用的是環境權限（ambient authority）：憑證不是請求自己帶的，而是瀏覽器依目的地自動附上。

## 為什麼存在

Cookie 的規則很單純：瀏覽器要送請求給 victim.test，就把 victim.test 的 Cookie 附上去。規則裡完全沒有「這個請求是哪個頁面發起的」。同源政策也沒補上這個洞，因為它擋的是跨源讀取，表單跨源提交本來就是 Web 的基本功能，不能擋。

於是任何網站都能在背景送一個 `POST` 到你的服務，而你的伺服器看到的是：有效的 Cookie、正確的路徑、格式完整的表單，跟 alice 自己按按鈕沒有差別。

這不是理論。Barth、Jackson 與 Mitchell 在 2008 年的論文 [Robust Defenses for Cross-Site Request Forgery](https://seclab.stanford.edu/websec/csrf/csrf.pdf) 裡，實測了包含 ING Direct、YouTube 在內的知名網站，都有 CSRF 漏洞。

後來瀏覽器自己補上一道防線：Cookie 的 `SameSite` 屬性。Chrome 自 2020 年起逐步把沒寫屬性的 Cookie 預設當成 `Lax`。這擋掉了最常見的情境，但不是全部，下面會一一對照。

## 怎麼運作

:::caution
以下所有攻擊只在你自己機器上的範例服務進行，兩個主機名都指向 `127.0.0.1`。絕不要對第三方網站或你沒有書面授權的系統測試 CSRF。
:::

範例在 `playground/golang/cyber/p2-web/2-3-cross-site/csrf/`。`victim.test:8080` 是受害服務，`evil.test:9666` 是攻擊者的頁面。先把兩個主機名指向本機（測完記得從 `/etc/hosts` 刪掉這一行）：

```bash
echo "127.0.0.1 victim.test evil.test" | sudo tee -a /etc/hosts
go run . -defense none -samesite default
```

### 為什麼要兩個主機名

上一篇用 `127.0.0.1` 的不同 port 就夠了，這篇不行。`SameSite` 判斷的是 **同站（same-site）**，只看 scheme 與可註冊網域，**不看 port**。`127.0.0.1:8080` 和 `127.0.0.1:9666` 是不同 origin，卻是同一個 site，Cookie 照送，你就看不到 `SameSite` 在擋什麼。所以要用 `victim.test` 與 `evil.test` 這兩個不同的站。

### 攻擊 A：跨站 POST 表單

攻擊頁面只是一個會自己送出的表單：

```html title="attacker.go"
{{if eq .Kind "post"}}
<form method="POST" action="{{.Victim}}/settings/target">
  <input type="hidden" name="url" value="{{.Evil}}">
</form>
<script>document.forms[0].submit()</script>
```

在瀏覽器登入（`http://victim.test:8080/login`）後，另開分頁打開 `http://evil.test:9666/post`。瀏覽器送出的請求長這樣：

```text
evil.test 的頁面        alice 的瀏覽器                 victim.test:8080
    │ form.submit()         │ POST /settings/target         │
    ├──────────────────────►│ Origin: http://evil.test:9666 │
    │                       │ Cookie: session=...（alice）  │
    │                       ├──────────────────────────────►│
    │                       │ 303，/a1 已被改走             │
    │                       │◄──────────────────────────────┤
```

注意攻擊者什麼都沒讀到，也不需要讀。請求自己就是目的：伺服器收到有效 Cookie 的 `POST`，就執行了 `setTarget`。

### 攻擊 B：頂層 GET 導覽

老系統常留一個「用網址改設定」的捷徑。範例裡的 `/legacy/set-target` 就是：

```go title="victim.go"
	// Legacy shortcut kept for old bookmarks: changes state on GET.
	mux.Handle("GET /legacy/set-target", requireSession(http.HandlerFunc(v.setTarget)))
```

攻擊頁面 `http://evil.test:9666/get` 不用表單，直接把整個分頁導到 `victim.test:8080/legacy/set-target?url=...`。這是 **頂層導覽（top-level navigation）**，下一節會看到它為什麼特別。

### 防線一：SameSite Cookie

`SameSite` 讓瀏覽器在「跨站」請求上不附 Cookie：

| Cookie 屬性 | 攻擊 A（跨站 POST） | 攻擊 B（頂層 GET 導覽） |
| --- | --- | --- |
| `SameSite=Strict` | 不附 Cookie，攻擊失敗 | 不附 Cookie，攻擊失敗 |
| `SameSite=Lax` | 不附 Cookie，攻擊失敗 | 附 Cookie，攻擊成功 |
| 沒寫屬性 | 看瀏覽器：預設 Lax 的不附 | 預設 Lax 的附 Cookie |

`Lax` 為了讓你從別的網站點連結進來還是登入狀態，放行「跨站、頂層、安全方法（GET）」的導覽。這是設計，不是漏洞，但它代表 **只靠 `Lax`，改狀態的 GET 仍然危險**。用 `-samesite lax` 與 `-samesite strict` 重跑，在瀏覽器 Network 分頁看請求有沒有 `Cookie`。

:::note
`SameSite` 由瀏覽器執行，Go 測試只能驗證伺服器發出的 `Set-Cookie` 屬性（`TestSessionCookieSameSite`），驗證不了瀏覽器的行為。要看實際效果得用瀏覽器。
:::

### 防線二：CSRF token

`SameSite` 是瀏覽器端的防線；伺服器要有自己的保證。最經典的是 token：每個 session 配一個攻擊者猜不到的值，表單要帶上它，伺服器比對。

```go title="defense.go"
func TokenFor(key []byte, session string) string {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(session))
	return hex.EncodeToString(m.Sum(nil))
}
```

token 是用伺服器私鑰對 session 算出的 HMAC，所以它 **綁定 session**。伺服器自己的設定頁把它放進表單：

```html title="victim.go"
<form method="POST" action="/settings/target">
  <input name="url" value="{{.Target}}">
  <input type="hidden" name="csrf_token" value="{{.Token}}">
```

evil.test 能送表單，卻拿不到 token，因為要拿到就得讀 victim.test 的 `/settings` 頁面，而那正是同源政策擋住的跨源讀取。兩道牆接起來：同源政策保護 token，token 保護寫入。

### 防線三：Fetch Metadata 與 Origin

第三條路是直接問瀏覽器：這個請求從哪來？現代瀏覽器會在請求上附 `Sec-Fetch-Site`（`same-origin`、`same-site`、`cross-site`、`none`），而且網頁的 JavaScript 無法偽造它。Go 1.25 起，標準函式庫把這套檢查做成 `http.CrossOriginProtection`：

```go title="defense.go"
func CrossSite(h http.Handler, trusted ...string) (http.Handler, error) {
	cop := http.NewCrossOriginProtection()
	for _, o := range trusted {
		if err := cop.AddTrustedOrigin(o); err != nil {
			return nil, err
		}
	}
	return cop.Handler(h), nil
}
```

對不安全的方法（`POST`、`PUT`、`DELETE`…），`Sec-Fetch-Site` 不是 `same-origin` 或 `none` 就回 403；舊瀏覽器沒有這個 header 時，改拿 `Origin` 跟 `Host` 比；兩個都沒有（`curl`、伺服器對伺服器）就放行，因為這種請求不是 CSRF 的受害者。

### 驗證：每種防線擋住了什麼

```bash
go test ./...
```

`crossSitePost` 重現攻擊 A 的請求（有效 Cookie、`Origin: http://evil.test:9666`、`Sec-Fetch-Site: cross-site`）：

| `-defense` | 攻擊 A 的結果 | 對應的測試 |
| --- | --- | --- |
| `none` | 303，`/a1` 被改走 | `TestForgedPostSucceedsWithoutDefense` |
| `token` | 403，`/a1` 不變 | `TestTokenBlocksForgedPost` |
| `crosssite` | 403，`/a1` 不變 | `TestCrossSiteBlocksForgedPost` |

攻擊 B 在 `token` 與 `crosssite` 下 **仍然成功**，`TestGetStateChangeBypassesBothDefenses` 證明了這一點。兩套伺服器端防線都只守不安全的方法，GET 不在裡面。真正的修法是別讓 GET 改狀態，這是下面的第一個坑，也是小練習第 2 題。

回到一開始的問題：evil.test 不需要讀任何東西。它只要讓 alice 的瀏覽器送出請求，瀏覽器就會自動附上 victim.test 的 Cookie，伺服器因此當成本人操作。擋住它的，是讓請求「附不附 Cookie」（`SameSite`）或「帶不帶得出伺服器才知道的值」（token、Fetch Metadata）。

## 取捨與替代方案

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| `SameSite=Lax` | 一行設定，擋掉多數跨站 POST | 不擋頂層 GET；同站的攻擊者（兄弟子網域）擋不住 | 所有 Cookie 認證的預設值 |
| `SameSite=Strict` | 連跨站導覽都不附 Cookie | 從外部連結點進來時看起來像沒登入 | 銀行、管理後台這類高敏感站 |
| 同步 token（綁 session） | 與瀏覽器版本無關，防護最完整 | 每個表單都要帶，前後端要配合 | 傳統表單、有伺服器渲染的頁面 |
| Fetch Metadata／`Origin` 檢查 | 不需要前端改動，標準函式庫就有 | 舊瀏覽器要退回 `Origin`；缺 header 時只能放行 | API 與表單，和 token 搭配 |
| 自訂 header（如 `X-CSRF-Token`） | 跨站請求要帶自訂 header 就得預檢 | 前提是 CORS 沒設錯，且 header 值仍要驗 | 單頁應用呼叫 JSON API |
| 敏感操作重新驗證 | 連被攻陷的 session 也擋得住 | 使用者體驗變差 | 改密碼、轉帳、刪帳號 |

建議的組合是 `SameSite=Lax`（或 `Strict`）當基線，再加上 Fetch Metadata 或 token 當伺服器端保證，並且不要讓 GET 改狀態。

什麼時候不該用或不需要：

- **認證完全不靠 Cookie 時。** 如果 API 只認 `Authorization: Bearer` header，而且前端用程式碼自己帶上去，瀏覽器不會自動附加，CSRF 就不成立，加 token 只是多餘的複雜度。
- **把它當成唯一一層時。** token 擋不住 XSS：攻擊者的腳本跑在 victim.test 的頁面裡，讀得到 token。`SameSite` 擋不住同站攻擊者。沒有哪一層單獨足夠。

## 常見坑與故障案例

### 讓 GET 改狀態

**症狀**：開了 token 與 Fetch Metadata，`/a1` 還是被改走。

**原因**：兩套防線只守不安全的方法。`GET /legacy/set-target` 帶著 Cookie 從頂層導覽進來，`Lax` 放行、token 與 `CrossOriginProtection` 都不檢查 GET。`TestGetStateChangeBypassesBothDefenses` 示範了這件事。

**對策**：改狀態的操作只接受 `POST`、`PUT`、`DELETE`，GET 一律唯讀。舊的捷徑路由直接移除。

### 以為預設 `Lax` 就夠了

**症狀**：新版 Chrome 上攻擊 A 打不進來，團隊就把 CSRF 防護拿掉。

**原因**：預設值是瀏覽器的行為，不是你的保證。別的瀏覽器或舊版本可能不套用；同站的攻擊者（被接管的子網域、同站的 XSS）本來就不受 `SameSite` 約束；為了嵌入第三方情境改成 `SameSite=None` 後，保護就消失了。（示意案例：本書範例未在多種瀏覽器上驗證，以 MDN 對 `SameSite` 的說明為準。）

**對策**：`SameSite` 當基線，伺服器端另外有 token 或 Fetch Metadata。

### double-submit cookie 沒簽章也沒綁 session

**症狀**：伺服器把隨機 token 同時放進 Cookie 與表單，兩者相等就放行，攻擊者在兄弟子網域用腳本寫入一個自己選的 Cookie，就能通過。

**原因**：這種做法只驗證「Cookie 和表單一致」，沒驗證「這個值是伺服器發的」。攻擊者能設定 Cookie（子網域、中間人），就能同時決定兩邊的值。

**對策**：像範例的 `TokenFor` 一樣，用私鑰對 session 簽，伺服器自己重算比對，比對用 `hmac.Equal` 做常數時間比較。

### 用前綴比對 Origin

**症狀**：`strings.HasPrefix(origin, "http://victim.test")` 看起來只信任自家，`http://victim.test.evil.test` 卻通過。

**原因**：和上一篇 CORS 的後綴比對是同一類錯誤：字串比對沒有邊界。Origin 是 scheme、host、port 的組合，要整串相等。

**對策**：用標準函式庫的 `CrossOriginProtection`，信任的 origin 用 `AddTrustedOrigin` 逐一列出；自己比對就整串相等。

### 登入頁沒有保護

**症狀**：使用者被登入成攻擊者的帳號卻沒發現，之後搜尋紀錄、地址、卡片資料都存進攻擊者的帳號。

**原因**：登入表單通常沒有 session 可綁 token，開發者因此跳過保護。這叫 **登入 CSRF（login CSRF）**，[Barth 等人的論文](https://seclab.stanford.edu/websec/csrf/csrf.pdf) 把它列為獨立的一類。

**對策**：登入表單也要有 token（登入前先發一個預先 session 的 token），或用 Fetch Metadata 檢查。

## 小練習

範例在 `playground/golang/cyber/p2-web/2-3-cross-site/csrf/`。先 `go test ./...` 確認全部通過。

**1. 預測並用瀏覽器驗證。** 不改程式，分別用 `-samesite lax` 與 `-samesite strict`、`-defense none` 啟動，登入後各打開攻擊 A 與攻擊 B 的頁面，記下四種組合（`lax`／`strict` × A／B）中 `/a1` 有沒有被改走。再換成 `-defense token`，哪一格會變？

<details>
<summary>參考解答</summary>

`lax`：A 不變（跨站 POST 不附 Cookie，伺服器回 401）、B 被改走（頂層 GET 導覽附 Cookie）。`strict`：A、B 都不變。換成 `-defense token` 後，`lax` 的 B 仍然被改走，因為 token 只守不安全的方法，GET 不在其中；A 在任何 `SameSite` 設定下都不會成功。這說明 `SameSite` 與 token 各自擋的不是同一件事。

</details>

**2. 移除改狀態的 GET。** 把 `victim.go` 的 `GET /legacy/set-target` 路由整條刪掉，並把 `TestGetStateChangeBypassesBothDefenses` 改寫成 `TestLegacyGetRemoved`：`token` 與 `crosssite` 兩種模式下，帶 Cookie 的跨站 GET 要拿到 404，且 `/a1` 維持 `initialTarget`。`go test ./...` 通過就算完成。

<details>
<summary>參考解答</summary>

刪掉 `victim.go` 裡那兩行（註解加 `mux.Handle`）後，測試改成：

```go
func TestLegacyGetRemoved(t *testing.T) {
	for _, def := range []string{"token", "crosssite"} {
		t.Run(def, func(t *testing.T) {
			h := newTestVictim(t, Config{Defense: def})
			rec := send(h, http.MethodGet, "/legacy/set-target?url="+url.QueryEscape(evilURL), "", map[string]string{
				"Cookie":         "session=" + sessionValue,
				"Sec-Fetch-Site": "cross-site",
			})
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", rec.Code)
			}
			if got := targetOf(h); got != initialTarget {
				t.Fatalf("/a1 -> %q, want unchanged", got)
			}
		})
	}
}
```

重點是 `setTarget` 現在只被 `POST /settings/target` 到達，受 token 或 `CrossOriginProtection` 保護；攻擊 B 失去入口。

</details>

### 辨識

下面這段中介層由 AI 產生，目的是「擋掉跨站表單對 `/settings/target` 的偽造請求」。用 victim.test 自己的設定頁測試，一切正常。請做 code review，找出它的問題（至少兩個）。

```go title="csrf.go"
// csrfGuard blocks cross-site form posts to state-changing endpoints.
func csrfGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			origin := r.Header.Get("Origin")
			if origin != "" && !strings.HasPrefix(origin, "http://victim.test") {
				http.Error(w, "cross-site request blocked", http.StatusForbidden)
				return
			}
			c, err := r.Cookie("csrf")
			if err != nil || c.Value != r.FormValue("csrf_token") {
				http.Error(w, "bad token", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
```

完整程式碼在 `recognize/csrf.go`，它有 `//go:build ignore`，不會被 `go test ./...` 掃到。

<details>
<summary>參考解答</summary>

1. **只檢查 `POST`，路由卻接受任何方法**：完整檔案裡是 `mux.HandleFunc("/settings/target", setTarget)`，沒有指定方法。攻擊者改用 `GET /settings/target?url=...` 做頂層導覽，`r.FormValue` 也讀得到 query，中介層整個被跳過。正確寫法是路由寫成 `POST /settings/target`，改狀態的操作不接受 GET。
2. **`strings.HasPrefix(origin, "http://victim.test")`**：`http://victim.test.evil.test` 也通過。而且只認 `http://`，上了 HTTPS 就全擋。正確寫法是整串相等的白名單，或改用 `http.CrossOriginProtection`。
3. **double-submit 的 token 沒簽章、沒綁 session**：`csrf` Cookie 是純隨機值，伺服器只比對「Cookie 等於表單欄位」。攻擊者能寫入 Cookie（兄弟子網域、中間人）就能同時決定兩邊。正確寫法是用私鑰對 session 做 HMAC，伺服器重算比對，並用 `hmac.Equal`。

</details>

## 快閃卡

<details>
<summary>CSRF 和上一篇的 CORS 錯誤設定，攻擊者分別拿到什麼？</summary>

CORS 錯誤設定讓攻擊者「讀」到受害者的私密回應；CSRF 讓攻擊者借受害者的身分「送出」一個會改狀態的請求，不需要讀到回應。

</details>

<details>
<summary>為什麼 evil.test 讀不到 victim.test 的任何回應，卻改得動 alice 的短連結？</summary>

同源政策只擋跨源讀取，不擋送出請求。瀏覽器依目的地自動附上 victim.test 的 Cookie，伺服器就當成 alice 本人操作。

</details>

<details>
<summary>為什麼 <code>127.0.0.1:8080</code> 和 <code>127.0.0.1:9666</code> 不能用來示範 <code>SameSite</code> 擋跨站請求？</summary>

`SameSite` 判斷同站，只看 scheme 與可註冊網域，不看 port。這兩個是同一個 site，Cookie 會照送，所以要用 `victim.test` 與 `evil.test`。

</details>

<details>
<summary>session Cookie 設 <code>SameSite=Lax</code>，evil.test 的頁面自動送出 <code>POST http://victim.test:8080/settings/target</code>，瀏覽器會附 Cookie 嗎？</summary>

不會。`Lax` 只在跨站、頂層、安全方法（GET）的導覽上附 Cookie，跨站 POST 不附，伺服器因此回 401。

</details>

<details>
<summary>同樣是 <code>SameSite=Lax</code>，evil.test 把分頁導到 <code>http://victim.test:8080/legacy/set-target?url=...</code>，會附 Cookie 嗎？</summary>

會。這是跨站、頂層、GET 的導覽，`Lax` 放行。所以只靠 `Lax`，會改狀態的 GET 仍然可以被攻擊。

</details>

<details>
<summary>為什麼綁 session 的 HMAC token 擋得住 evil.test？</summary>

token 只出現在 victim.test 自己的 `/settings` 頁面裡，evil.test 要拿到就得跨源讀那個頁面，同源政策擋住了它。沒有 token，伺服器重算比對就會失敗。

</details>

<details>
<summary>已開 <code>-defense crosssite</code>，<code>curl -X POST</code> 帶有效 Cookie 但沒有 <code>Origin</code> 與 <code>Sec-Fetch-Site</code>，會被擋嗎？</summary>

不會，兩個 header 都缺就放行（`TestCrossSiteAllowsSameOriginAndNonBrowser`）。這種請求不是瀏覽器自動附憑證的 CSRF 情境，攻擊者能自己帶 Cookie 就根本不需要 CSRF。

</details>

<details>
<summary>伺服器只檢查 <code>strings.HasPrefix(origin, "http://victim.test")</code>，<code>Origin: http://victim.test.evil.test</code> 會通過嗎？</summary>

會。前綴比對沒有邊界，`victim.test.evil.test` 以 `http://victim.test` 開頭。Origin 要整串相等才可靠。

</details>

<details>
<summary>token 與 Fetch Metadata 都開了，<code>/a1</code> 還是被改走，先檢查什麼？</summary>

檢查有沒有會改狀態的 GET 路由。兩套防線都只守不安全的方法，GET 帶 Cookie 的頂層導覽會直接穿過去。

</details>

## 延伸閱讀

- [Cross-Site Request Forgery Prevention Cheat Sheet（OWASP）](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)：各種防禦的完整清單與適用條件，本篇的取捨表可以對照它。
- [Robust Defenses for Cross-Site Request Forgery（Barth、Jackson、Mitchell）](https://seclab.stanford.edu/websec/csrf/csrf.pdf)：把 CSRF 與 login CSRF 系統化、並提出 Origin header 的原始論文。
- [Cross-site request forgery (CSRF)（PortSwigger Web Security Academy）](https://portswigger.net/web-security/csrf)：可動手練的實驗室，涵蓋 token 與 SameSite 的常見繞過。
- [Set-Cookie 的 SameSite 屬性（MDN）](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Set-Cookie)：`Strict`、`Lax`、`None` 的權威說明。
- [`http.CrossOriginProtection`（Go 文件）](https://pkg.go.dev/net/http#CrossOriginProtection)：本篇防線三用的標準函式庫實作，含它的判斷順序與例外設定。
