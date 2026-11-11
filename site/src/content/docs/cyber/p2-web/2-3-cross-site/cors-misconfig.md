---
title: CORS 錯誤設定：反射 Origin、null origin 與憑證竊取
description: 在自己的範例服務上重現三種 CORS 錯誤設定，讀出登入者的私密資料，再用白名單修好。
sidebar:
  order: 1
---

## 先想想

你在稽核自己寫的短網址服務 v1。它有一個 `GET /api/me`，登入後會回傳目前使用者的 API 金鑰。前端部署在 `https://app.sho.rt`，為了讓它能跨源讀取，後端在 CORS 這樣寫：把收到的 `Origin` 原封不動回寫進 `Access-Control-Allow-Origin`，再加上 `Access-Control-Allow-Credentials: true`。

後端 1.4 說過這是最常見的 CORS 漏洞。現在換到攻擊方：你另外在 `http://127.0.0.1:9666` 放一個頁面，裡面只有一行 `fetch("http://127.0.0.1:8080/api/me", {credentials: "include"})`。

如果已登入的 alice 打開了這個頁面，這段 `fetch` 讀得到她的 API 金鑰嗎？如果讀得到，問題出在 alice 的瀏覽器，還是伺服器那一行 CORS？

<details>
<summary>提示</summary>

想想 [同源政策](/cs/n-network/n-4-application/same-origin-policy/) 本來會擋下跨源讀取，而 [CORS](/backend/l1/1-4-auth-basics/cors/) 是伺服器主動放寬這道限制。放寬的範圍是誰決定的？

</details>

## 是什麼

**CORS 錯誤設定（CORS misconfiguration）** 是伺服器把 `Access-Control-Allow-Origin` 開給了不該信任的 origin，讓那些 origin 的網頁能帶著使用者的 Cookie 讀取私密回應。

它屬於存取控制類漏洞，但位置很特別：真正放行的不是你的程式邏輯，而是你交給瀏覽器的那幾個 CORS header。瀏覽器忠實地照著 header 執行，於是一行寫錯的 header 就等於對全世界的網站開門。

## 為什麼存在

同源政策讓惡意網站預設讀不到你登入後的 API 回應。CORS 是在這道牆上開門，而開門的規則細節很多：要比對整串 origin、不能用 `*` 搭配 Cookie、`null` 不是安全值。任何一條搞錯，門就開得比你想的大。

這類錯誤特別危險，因為它平常完全不報錯。前端能跑、測試會過、瀏覽器不出警告，直到有人從別的 origin 帶著受害者的 Cookie 把資料讀走。PortSwigger 的研究者 James Kettle 在 2016 年的文章裡，就用反射 origin 的錯誤從一個比特幣交易所讀出使用者的私密資料。

攻擊方要學會辨認這些設定，一來是滲透測試時要找得到，二來是回到後端時知道怎麼一開始就不要寫錯。

## 怎麼運作

:::caution
以下所有攻擊只在你自己機器上的範例服務（`127.0.0.1`）進行。絕不要對第三方網站或你沒有書面授權的系統測試 CORS。
:::

範例在 `playground/golang/cyber/p2-web/2-3-cross-site/cors-misconfig/`。`-mode` 旗標切換伺服器用哪一種 origin 檢查，受害 API 在 `127.0.0.1:8080`，攻擊頁面在 `127.0.0.1:9666`。

### 攻擊的形狀：頁面什麼都沒做

攻擊頁面本身沒有任何技巧，它就是一個 `fetch`：

```go title="attacker.go"
    // credentials: "include" sends the user's :8080 cookies on this
    // cross-origin request. The browser only hands back the response body
    // if the API's Access-Control-Allow-Origin names this page's origin.
    const res = await fetch("http://127.0.0.1:8080/api/me", {credentials: "include"});
    out.textContent = "讀到了：" + await res.text();
```

`credentials: "include"` 讓瀏覽器在這個跨源請求上附帶使用者在 `:8080` 的 Cookie。請求一定會送到伺服器、handler 一定會執行、回應一定會送回瀏覽器。唯一的分歧點是：**瀏覽器要不要把回應本體交給這個頁面的 JavaScript**。那完全由伺服器回的 `Access-Control-Allow-Origin` 決定。

```text
攻擊頁面 (:9666)        alice 的瀏覽器              受害 API (:8080)
    │ fetch(/api/me,       │                              │
    │  include)            │ GET /api/me                  │
    ├─────────────────────►│ Origin: http://...:9666      │
    │                      │ Cookie: session=...（alice） │
    │                      ├─────────────────────────────►│
    │                      │ 200 {"api_key": "..."}       │
    │                      │ Access-Control-Allow-Origin: ?│
    │                      │◄─────────────────────────────┤
    │                      │ 若 Allow-Origin == :9666      │
    │   ◄── 交出本體 ──────┤   才把 api_key 交給 JS        │
```

要看的是最後一步：金鑰已經回到瀏覽器了，交不交給攻擊頁面只看那一行 header。

### 反射 Origin：對所有人開門

最寬的錯誤設定是把收到的 `Origin` 直接回寫：

```go title="policy.go"
// Reflect trusts whatever Origin the request claims. Vulnerable.
func Reflect() Check {
	return func(origin string) bool { return origin != "" }
}
```

任何 origin 送來都會被寫回 `Access-Control-Allow-Origin`，再配上 `Access-Control-Allow-Credentials: true`。`:9666` 的頁面因此讀得到 alice 的金鑰。`TestMisconfigsLeak` 以 `http://127.0.0.1:9666` 這個 origin 驗證了這一點。

### null origin：用 sandbox iframe 偽裝

開發時常從 `file:` 開頁面，這時 `Origin` 是 `null`，有人就把 `null` 加進白名單：

```go title="policy.go"
// AllowNull is an exact allowlist that also trusts "null"...
// Vulnerable: any site can send Origin: null from a sandboxed iframe.
func AllowNull(origins ...string) Check {
	exact := Allowlist(origins...)
	return func(origin string) bool { return origin == "null" || exact(origin) }
}
```

問題是 `null` 不是只有 `file:` 會產生。任何網站都能用帶 `sandbox` 屬性的 `<iframe>` 載入攻擊頁面，讓它送出的請求 `Origin` 變成 `null`。信任 `null` 等於信任所有人。

### 前綴與後綴比對：少了邊界

要支援很多子網域時，有人用字串比對代替整串相等：

```go title="playground/golang/cyber/p2-web/2-3-cross-site/cors-misconfig/policy/policy.go"
// Suffix trusts any origin ending in domain. Vulnerable: https://evilsho.rt
// also ends in "sho.rt".
func Suffix(domain string) Check {
	return func(origin string) bool { return strings.HasSuffix(origin, domain) }
}
```

`HasSuffix(origin, "sho.rt")` 會放行攻擊者註冊的 `https://evilsho.rt`；改成 `HasPrefix(origin, "https://app.sho.rt")`，則 `https://app.sho.rt.evil.example` 又通過。和反射 origin 不同，這兩種錯誤**只對特定網域生效**：攻擊者得先擁有一個名字剛好能騙過比對的網域。所以範例裡這兩種模式不會對 `:9666` 的頁面生效，`TestMisconfigsLeak` 分別用 `https://evilsho.rt` 和 `https://app.sho.rt.evil.example` 來觸發。

### 修補：整串比對的白名單

四種錯誤的共通修法只有一個：準備一份明確的 origin 清單，整串相等才放行。

```go title="playground/golang/cyber/p2-web/2-3-cross-site/cors-misconfig/policy/policy.go"
// Allowlist trusts only origins that match an entry exactly. The fix.
func Allowlist(origins ...string) Check {
	set := make(map[string]bool, len(origins))
	for _, o := range origins {
		set[o] = true
	}
	return func(origin string) bool { return set[origin] }
}
```

`fixed` 模式用的就是它。`TestFixedModeBlocksAttacker` 驗證攻擊頁面、`null`、`https://evilsho.rt` 全部讀不到金鑰，而真正的前端 `https://app.sho.rt` 照常能讀。白名單的每個項目要寫成正規化的 origin（小寫、無路徑、省略預設 port），這點和後端 1.4 的 `checkOrigin` 一致。

回到一開始的問題：那段 `fetch` 讀不讀得到金鑰，不取決於 alice 的瀏覽器——瀏覽器只是照 header 辦事——而取決於伺服器那一行反射 `Origin` 的 CORS。把它換成整串比對的白名單，金鑰就只交給你信任的前端。

## 取捨與替代方案

| origin 檢查 | 能擋住攻擊頁面嗎 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| 反射 `Origin`（原封回寫） | 不能，對所有 origin 開門 | 等於沒有限制 | 任何情境都不該用 |
| 信任 `null` | 不能，sandbox iframe 可偽裝 | 開發方便，但開給所有人 | 任何情境都不該用 |
| 後綴／前綴比對 | 部分，攻擊者可註冊匹配網域 | 看似方便，邊界難寫對 | 不該用；真要多子網域就解析 host 再比對 |
| 整串比對白名單 | 能 | 多子網域時清單要維護 | 預設做法 |
| 不開 `Credentials`、資料也公開 | 不適用，本來就沒有秘密可偷 | 只適合公開資料 | 公開的短網址統計這類端點 |

什麼時候這個攻擊不成立：

- **API 不使用 Cookie 或其他憑證時。** 沒有 `credentials: "include"`，攻擊頁面即使讀到回應，也只是沒登入的公開內容，偷不到任何人的資料。James Kettle 的文章也指出，少了憑證，多數 CORS 攻擊就失去意義。
- **目標資料本來就公開時。** 開 `Access-Control-Allow-Origin: *`（且不帶 Cookie）讀公開資料，沒有秘密會外洩。例外是用網路位置當認證的內網服務，這時「讀得到」本身就是問題。

## 常見坑與故障案例

### 以為 `curl` 讀不到就代表安全

**症狀**：你用 `curl` 測 `/api/me`，發現沒有 `Access-Control-Allow-Origin` 就放心了，認為跨源讀不到。

**原因**：CORS 只約束瀏覽器。`curl` 和任何非瀏覽器程式根本不看這個 header，帶著有效 Cookie 一樣讀得到。`TestNonBrowserClientAlwaysReads` 證明即使 `fixed` 模式，沒有 Origin 的請求照樣拿到金鑰。擋住非瀏覽器客戶端的是認證（`requireSession`），不是 CORS。

**對策**：用瀏覽器或帶 `Origin` header 的請求測 CORS；用認證而非 CORS 來做存取控制。

### 把 `null` 當成安全的開發用值

**症狀**：為了本機 `file:` 開發方便，把 `null` 放進白名單，上線也忘了拿掉。

**原因**：任何網站都能用 sandbox iframe 送出 `Origin: null`。James Kettle 記錄過 `docs.google.com` 的一個端點曾回傳 `Access-Control-Allow-Origin: null` 並允許帶 Cookie。

**對策**：開發改用 `http://localhost:<port>` 起伺服器，白名單永遠不要有 `null`。範例的 `checkOrigin`（後端 1.4）遇到 `null` 會直接拒絕啟動。

### 用後綴比對信任「自家網域」

**症狀**：`strings.HasSuffix(origin, "sho.rt")` 看起來只信任自家網域，卻放行了 `https://evilsho.rt`。

**原因**：James Kettle 的研究記錄了真實網站這類錯誤：有站台信任任何以 `advisor.com` 結尾的 origin，結果 `definitelynotadvisor.com` 也通過；有站台信任任何以 `https://btc.net` 開頭的，`https://btc.net.evil.net` 也通過。

**對策**：整串比對。真要支援子網域，先解析出 host，再比對 `host == "sho.rt" || strings.HasSuffix(host, ".sho.rt")`，注意那個前導的點。

### 從 HTTP origin 放行，破壞了 HTTPS

**症狀**：HTTPS 的 API 為了相容，白名單裡留了一個 `http://` 的子網域。

**原因**：James Kettle 指出，一旦 HTTPS 站台信任某個 HTTP origin，網路中間人就能攔截那個 HTTP 連線、偽造請求，繞過大半的 HTTPS 保護，連 HSTS 和 secure cookie 都幫不上忙。（示意案例：本書範例未示範中間人環境）

**對策**：白名單只放 `https://` 的 origin；開發用的 `http://localhost` 不要出現在正式設定裡。

## 小練習

範例在 `playground/golang/cyber/p2-web/2-3-cross-site/cors-misconfig/`。先 `go test ./...` 確認全部通過。

**1. 預測每種模式對應的受害 origin。** 讀 `policy/policy.go` 的四個檢查，寫下每一種會把金鑰洩漏給哪一個 origin，再對照 `server_test.go` 的 `TestMisconfigsLeak`。為什麼 `suffix` 和 `prefix` 不會對 `http://127.0.0.1:9666` 生效？

<details>
<summary>參考解答</summary>

`reflect` 洩漏給任何 origin，包含 `:9666`；`null` 洩漏給 `Origin: null`（sandbox iframe）；`suffix` 洩漏給 `https://evilsho.rt`；`prefix` 洩漏給 `https://app.sho.rt.evil.example`。`suffix`／`prefix` 比對的是特定字串邊界，`http://127.0.0.1:9666` 既不以 `sho.rt` 結尾、也不以 `https://app.sho.rt` 開頭，所以不符合——這兩種攻擊需要攻擊者先擁有一個名字匹配的網域。

</details>

**2. 加一種只比對協定與 host、卻忽略 port 的錯誤檢查。** 在 `policy` 加一個 `IgnorePort` 檢查：把 `https://app.sho.rt:8443` 也視為等於 `https://app.sho.rt`。寫一個測試，證明攻擊者用 `https://app.sho.rt:1` 就能通過。

<details>
<summary>參考解答</summary>

```go
func IgnorePort(origin string) Check {
	base, _, _ := strings.Cut(strings.TrimPrefix(origin, "https://"), ":")
	return func(got string) bool {
		g, _, _ := strings.Cut(strings.TrimPrefix(got, "https://"), ":")
		return g == base
	}
}
```

測試用 `IgnorePort("https://app.sho.rt")` 檢查 `https://app.sho.rt:1` 回傳 `true`。重點是 origin 的三個組成（scheme、host、port）任何一個被忽略，就多開了一扇門；port 也是 origin 的一部分。

</details>

**3. 證明修補後攻擊失效。** 不改測試的前提下，想清楚 `TestFixedModeBlocksAttacker` 為什麼能同時證明「攻擊者讀不到」與「真前端讀得到」。它檢查了哪兩件事的組合？

<details>
<summary>參考解答</summary>

`leaks` 同時檢查「回應本體含金鑰」且「`Access-Control-Allow-Origin` 等於該 origin」且「`Access-Control-Allow-Credentials` 是 true」——這正是瀏覽器交出本體前會檢查的條件。`fixed` 模式下，handler 對每個 origin 都回了金鑰（本體相同），差別只在 header：攻擊 origin 拿不到 `Allow-Origin`，`app.sho.rt` 拿得到。所以測試證明的是「瀏覽器會不會交出」，而不是「伺服器有沒有產生」。

</details>

## 快閃卡

<details>
<summary>CORS 錯誤設定洩漏資料時，真正「放行」的是你的程式邏輯，還是回應的 header？</summary>

是 header。handler 照常執行並回傳資料，瀏覽器只根據 `Access-Control-Allow-Origin` 決定要不要把本體交給跨源頁面。

</details>

<details>
<summary>攻擊頁面在 <code>http://127.0.0.1:9666</code>，用 <code>fetch(api, {credentials: "include"})</code> 讀 alice 的 API。伺服器是 reflect 模式，讀得到金鑰嗎？</summary>

讀得到。reflect 把 `:9666` 原封回寫進 `Access-Control-Allow-Origin` 並允許帶 Cookie，瀏覽器因此把金鑰交給攻擊頁面。

</details>

<details>
<summary>同樣的攻擊頁面在 <code>:9666</code>，伺服器改用 <code>HasSuffix(origin, "sho.rt")</code>，讀得到嗎？</summary>

讀不到。`http://127.0.0.1:9666` 不以 `sho.rt` 結尾。後綴比對只會洩漏給攻擊者註冊的、名字真的以 `sho.rt` 結尾的網域，例如 `https://evilsho.rt`。

</details>

<details>
<summary>為什麼把 <code>null</code> 放進白名單等於信任所有人？</summary>

任何網站都能用帶 `sandbox` 屬性的 iframe 載入頁面，讓請求的 `Origin` 變成 `null`，不是只有本機 `file:` 才會產生 `null`。

</details>

<details>
<summary>你用 curl 測 <code>/api/me</code>，沒看到 <code>Access-Control-Allow-Origin</code>，能據此說「跨源偷不到」嗎？</summary>

不能。curl 不受 CORS 約束，帶著有效 Cookie 一樣讀得到。CORS 只影響瀏覽器要不要把回應交給網頁，擋非瀏覽器客戶端的是認證。

</details>

<details>
<summary>API 完全不使用 Cookie 或憑證，反射 Origin 的設定還能偷到使用者資料嗎？</summary>

不能。沒有憑證，攻擊頁面讀到的只是未登入的公開回應，偷不到特定使用者的私密資料。多數 CORS 攻擊都需要憑證才成立。

</details>

<details>
<summary>整串比對的白名單，怎麼同時擋住攻擊者又放行真前端？</summary>

它只在 `Origin` 和清單項目完全相等時回寫 `Access-Control-Allow-Origin`。攻擊 origin 不在清單裡拿不到這個 header，真前端在清單裡拿得到，瀏覽器據此分別拒絕與放行。

</details>

<details>
<summary>HTTPS 的 API 在白名單裡放了一個 <code>http://</code> 的 origin，有什麼風險？</summary>

網路中間人能攔截那個 HTTP 連線並偽造跨源請求，繞過大半 HTTPS 保護，HSTS 與 secure cookie 都擋不住。白名單應只放 `https://` 的 origin。

</details>

## 延伸閱讀

- [Exploiting CORS misconfigurations for Bitcoins and bounties（PortSwigger，James Kettle）](https://portswigger.net/research/exploiting-cors-misconfigurations-for-bitcoins-and-bounties)：本篇四種錯誤設定的一手來源，含反射 origin、`null`、前後綴比對與破壞 HTTPS 的真實案例。
- [CORS（PortSwigger Web Security Academy）](https://portswigger.net/web-security/cors)：把上述漏洞整理成可練習的教材，附 origin 驗證繞過的對照清單。
- [Cross-Origin Resource Sharing (CORS)（MDN）](https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CORS)：header 與預檢規則的權威說明，修補時對照用。
