---
title: JWT 的基礎與限制
description: 能看懂 JWT 的三段結構，用 Go 簽發與驗證 HS256 token，並說出它不能加密、難撤銷的限制。
sidebar:
  order: 4
---

## 先想想

短網址服務 v1 除了網頁，還要給手機 App 用。App 沒有瀏覽器的 Cookie 機制，所以團隊決定：登入成功後回傳一串 token，App 之後每個請求都帶上 `Authorization: Bearer <token>`。

同事提議：token 直接放 `{"sub":"42"}` 的 base64 編碼，伺服器解開就知道是誰。另一位同事說，改成 JWT，把使用者編號放進去再簽章，伺服器連查表都不用。

第一個做法為什麼不行？第二個做法省掉了查表，但使用者按下「登出」之後，這個 token 還能用嗎？

<details>
<summary>提示</summary>

base64 是編碼，不是加密，任何人都能解開、也能重新編碼。想想：簽章能證明什麼，又證明不了什麼？

</details>

## 是什麼

**JWT（JSON Web Token）** 是一種把一組宣告（claims）編碼成 `header.payload.signature` 三段字串、並以簽章防止竄改的 token 格式，規格是 RFC 7519。

它和 [session ID](/backend/l1/1-4-auth-basics/session-cookie/) 最大的差別是：session ID 只是查表的鑰匙，意義存在伺服器；JWT 把意義（誰、何時過期）直接寫在 token 裡，由簽章保證內容沒被改過。所以它是 **自包含（self-contained）** 的：驗證者只要有金鑰，不必查任何儲存。

## 為什麼存在

Session 的前提是「伺服器查得到那筆紀錄」。當情境變成下面這些，這個前提就變成負擔：

- 客戶端不是瀏覽器，沒有 Cookie 自動附帶機制。
- 一個請求要經過好幾個服務（API 閘道、訂單服務、通知服務），每個都要知道「是誰」。若每個服務都去查同一個 session 儲存，儲存就變成單點，每個請求多一次網路往返。
- 認證由另一個系統負責（例如登入服務），業務服務不想、也不該連它的資料庫。

JWT 的做法是：登入服務簽發一次，其他服務各自用金鑰驗證簽章，就能在不碰任何儲存的情況下信任內容。代價在「取捨」一節。

## 怎麼運作

### 三段結構

一個 JWT 長這樣（用 `.` 分成三段，這裡換行方便看）：

```text
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9
.eyJpc3MiOiJzaG8ucnQiLCJzdWIiOiI0MiIsImlhdCI6MTc5NDYyNTE0OSwiZXhwIjoxNzk0NjI2MDQ5fQ
.nat6ViZNC4i9M5F0jvyAmHkXVT2DKt2QpKsdr2EtxYQ
```

每一段各自做 base64url 編碼（用 `-` 與 `_` 取代 `+` 與 `/`，並去掉結尾的 `=`）。解開前兩段是 JSON：

| 段 | 解碼後 | 作用 |
| --- | --- | --- |
| header | `{"alg":"HS256","typ":"JWT"}` | 說明用哪個演算法簽章 |
| payload | `{"iss":"sho.rt","sub":"42","iat":1794625149,"exp":1794626049}` | 宣告：誰簽發、主體是誰、何時簽發、何時過期 |
| signature | 一串位元組 | 前兩段的簽章 |

payload 裡的 `iss`、`sub`、`iat`、`exp` 是 RFC 7519 定義的標準宣告（registered claims）。`exp` 是過期時間，單位是 Unix 秒。

### 簽章怎麼算

最常見的 HS256 是 HMAC-SHA256：用只有伺服器知道的金鑰，對前兩段字串做雜湊訊息驗證碼。

```text
signingInput = base64url(header) + "." + base64url(payload)
signature    = base64url( HMAC-SHA256(key, signingInput) )
token        = signingInput + "." + signature
```

任何人改動 header 或 payload 的一個字元，signingInput 就變了，沒有金鑰的人算不出對應的簽章。這就是「防竄改」的全部意思。

用 Go 標準庫簽發只要幾行：

```go title="jwt.go"
func mac(key []byte, signingInput string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(signingInput))
	return h.Sum(nil)
}

input := enc.EncodeToString(hb) + "." + enc.EncodeToString(pb)
return input + "." + enc.EncodeToString(mac(key, input)), nil
```

`enc` 是 `base64.RawURLEncoding`，正好是 JWT 要求的無填充 base64url。

:::note
HS256 是 **對稱** 簽章：簽發與驗證用同一把金鑰，所有要驗證的服務都握有能簽發的金鑰。RS256、ES256 是 **非對稱** 簽章：登入服務用私鑰簽發，其他服務只拿公鑰驗證，公鑰外洩也無法偽造。服務多、驗證方不該有簽發權時用後者。
:::

### 驗證：順序與兩個鐵則

驗證要依序做，缺一項都是漏洞：

```text
收到 token
  │
  ├─ 拆成三段，格式不對 → 拒絕
  ├─ 讀 header.alg，不等於伺服器預期的演算法 → 拒絕
  ├─ 重算簽章，與 token 的簽章比對 → 不符就拒絕
  ├─ 檢查 exp（沒有 exp 也拒絕）→ 過期就拒絕
  └─ 檢查 iss 等其他宣告 → 通過才信任 payload
```

這張圖的重點是：payload 在簽章通過之前，一個字都不能信。

範例的 `Verify` 把這個順序寫成程式：

```go title="jwt.go"
if h.Alg != "HS256" {
	return Claims{}, ErrAlgorithm
}
// ...
if !hmac.Equal(sig, mac(key, parts[0]+"."+parts[1])) {
	return Claims{}, ErrSignature
}
// ...
// A token without exp must not live forever.
if c.ExpiresAt == 0 || !now.Before(time.Unix(c.ExpiresAt, 0)) {
	return Claims{}, ErrExpired
}
```

兩個鐵則：

1. **演算法由伺服器決定，不看 token 自己說了什麼。** header 是攻擊者可以任意填的欄位。
2. **簽章比對用 `hmac.Equal`**，它是常數時間比較；用 `==` 逐位元組比較，回應時間會洩漏「前幾個位元組對了」。

### payload 是編碼，不是加密

任何拿到 token 的人，不需要金鑰就能讀 payload：

```go title="jwt.go"
// Peek decodes the payload WITHOUT verifying anything. It exists to show
// that a JWT payload is encoded, not encrypted.
func Peek(token string) (string, error) {
```

所以 payload 不能放密碼、身分證字號、內部權限細節這類不能讓使用者（或有人側錄到 token）看到的資料。要加密需要另一個規格 JWE（RFC 7516），一般的登入 token 用不到。

回到一開始的問題：第一個做法 `{"sub":"42"}` 只有編碼，沒有簽章，任何人都能改成 `{"sub":"1"}` 再編回去，伺服器無從分辨，和 `Set-Cookie: user_id=42` 是同一個錯。JWT 加上簽章後，沒有金鑰的人改不動 payload。但簽章只證明「這是我們簽的、內容沒被改」，不證明「現在還該被接受」：伺服器沒有任何一筆紀錄可以刪，所以使用者登出後，這個 token 在 `exp` 之前仍然通過驗證。這就是下一節的核心限制。

## 取捨與替代方案

JWT 把「查表」換成「驗簽章」，換來的與失去的：

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| 伺服器端 session | 可立刻撤銷、能隨時改權限、Cookie 很小 | 每個請求查一次儲存；多台伺服器要共用儲存 | 瀏覽器為主的網站 |
| JWT（短效） | 驗證不查儲存；跨服務、跨網域傳遞方便 | 到期前難以撤銷；token 偏大；payload 公開可讀；金鑰管理是新負擔 | 手機 App、第三方 API、微服務間 |
| JWT + refresh token | access token 短效，登出可撤銷 refresh token | 多一套發放與儲存 refresh token 的機制，等於把有狀態搬到另一處 | 要同時兼顧體驗與可撤銷 |
| 不透明 token（隨機字串 + 查表） | 隨時撤銷，不洩漏內容 | 每次要查儲存或呼叫驗證端點 | 要即時撤銷的 API |

JWT 的限制可以濃縮成三句：

- **難以撤銷**：簽發出去的 token 在 `exp` 前一律有效。要「立刻登出」只能讓伺服器另存一份黑名單，而查黑名單就是查表，省下的成本又回來了。
- **內容在簽發時就固定**：使用者被停權、被降級，已經發出去的 token 還帶著舊權限，直到過期。
- **大小與暴露**：payload 越多，每個請求的 header 越大；而且內容人人可讀。

所以實務上的折衷是 **access token 設 5 到 15 分鐘**，配一個伺服器能撤銷的 refresh token，細節在下一篇〈Session vs Token〉。

### 什麼時候不該用

- **只有一個瀏覽器網站、一個後端時**：用 session 更簡單，撤銷、停權都是刪一筆紀錄，沒有理由為「不查表」多背金鑰管理與撤銷難題。
- **需要即時撤銷或即時改權限時**：例如金融操作、後台管理。短效 JWT 也只是縮小視窗，不是消除。
- **想把它當「加密的資料容器」時**：JWT 預設只簽章不加密，不要塞敏感資料。

## 常見坑與故障案例

### 坑 1：信任 header 裡的 `alg`

症狀：攻擊者不需要金鑰就能偽造任何使用者的 token。

原因：規格允許 `alg` 設成 `none`（無簽章）。有些函式庫與手寫程式會先讀 token 自己宣告的演算法再決定怎麼驗，攻擊者把 header 改成 `{"alg":"none"}`、簽章留空，驗證就被整個略過。2015 年 Tim McLean 公開了這類漏洞：多個主流函式庫接受 `alg: none`，另一種手法是把 RS256 的 token 改成 HS256，再用公開的公鑰當 HMAC 金鑰簽章（見延伸閱讀）。

避免：驗證時指定允許的演算法，其餘一律拒絕。範例的 `TestAlgNoneRejected` 檢查帶 `alg: none` 的 token 會得到 `ErrAlgorithm`。使用函式庫時，要找到「限定演算法」的選項並打開。

### 坑 2：沒檢查 `exp`

症狀：幾個月前簽發的 token 仍然能呼叫 API；被偷走的 token 永遠有效。

原因：簽章驗證通過就回傳 payload，沒有比對 `exp`；或簽發時根本沒設 `exp`，這種 token 永遠不會過期。

避免：簽發一律設 `exp`，驗證一律檢查，並把「沒有 `exp`」當成錯誤。`TestMissingExpRejected` 與 `TestExpired` 涵蓋這兩種情況。

### 坑 3：登出只是前端把 token 丟掉

症狀：使用者在 App 按下登出，但先前被複製或側錄的 token 仍能操作帳號，直到過期。

原因：JWT 自包含，伺服器沒有任何東西可以刪。前端丟掉 token 只是自己不再用，不影響別人手上的副本。這和 session 的〈登出只清了 Cookie〉是同一件事，只是 JWT 連「刪伺服器端紀錄」這個選項都沒有。

避免：access token 設短效；需要可撤銷時搭配 refresh token，登出時撤銷 refresh token；真的需要立刻失效再加黑名單。

### 坑 4：把敏感資料或會過時的權限放進 payload

症狀：有人用 [jwt.io](https://jwt.io/) 之類的工具貼上 token，看到了使用者的 email、內部角色，甚至不該出現的欄位；管理員停權某人後，他仍能做管理操作。

原因：payload 只是 base64url，人人可讀；權限寫進 token 就固定到過期為止，和 session 的〈存了會過時的權限〉一樣。

避免：payload 只放識別用的最小資訊（`sub`、`exp` 等），其餘從資料庫查；權限變動頻繁的系統，驗證通過後仍查一次目前的權限。

### 坑 5：金鑰太弱或寫死在程式碼裡（示意案例）

症狀：攻擊者拿到一個有效 token，離線嘗試字典攻擊，幾分鐘內就猜出金鑰，接著能簽發任意 token。

原因：HS256 的簽章可以離線驗證，攻擊者不需要連到你的伺服器。金鑰若是 `secret`、`change-me` 之類的弱字串，或寫在公開的程式碼庫裡，就能被猜出或直接取得。

避免：金鑰用 `crypto/rand` 產生至少 32 bytes 的隨機值，由環境變數或密鑰管理服務注入，不進版本控制；怕外洩影響範圍太大時，改用非對稱簽章，讓驗證方沒有簽發能力。

## 小練習

範例放在 `playground/backend/l1/1-4-auth-basics/jwt-basics/`，執行 `go test ./...` 可以驗證，`go run .` 會印出一個 token 與解碼後的 payload。

### 動手寫

目前 `Verify` 嚴格拒絕剛過期的 token。但多台伺服器的時鐘常有幾秒的誤差，剛簽發的 token 可能因為驗證端時鐘稍慢而被認為「尚未生效」，或因為稍快而提早過期。請在 `jwt/jwt.go` 加兩件事：

1. `Claims` 新增 `NotBefore int64`（JSON 欄位 `nbf`）；`Verify` 在 `now` 早於 `nbf` 時回傳新的 `ErrNotYetValid`（`nbf` 為 0 代表不限制）。
2. `Verify` 新增一個 `leeway time.Duration` 參數，檢查 `exp` 與 `nbf` 時容許這麼多的時鐘誤差。

寫兩個測試：`leeway` 為 0 時，`nbf` 在 10 秒後的 token 要被拒絕；`leeway` 為 30 秒時要通過。

<details>
<summary>參考解答</summary>

```go
if c.NotBefore != 0 && now.Add(leeway).Before(time.Unix(c.NotBefore, 0)) {
	return Claims{}, ErrNotYetValid
}
if c.ExpiresAt == 0 || !now.Add(-leeway).Before(time.Unix(c.ExpiresAt, 0)) {
	return Claims{}, ErrExpired
}
```

測試：簽發 `NotBefore: t0.Add(10 * time.Second).Unix()` 的 token，`Verify(tok, key, "sho.rt", 0, t0)` 回傳 `ErrNotYetValid`，`Verify(tok, key, "sho.rt", 30*time.Second, t0)` 通過。

`leeway` 要保持很小（幾秒到幾十秒）。設太大，等於變相延長了每個 token 的有效時間。

</details>

### 辨識

下面是 AI 產生的 token 解析函式。給它正確簽發的 token，會回傳正確的使用者編號；簽章被改過的 token 也會被擋下來。請做 code review，找出 3 個問題。

```go title="verify.go"
func parseUserID(token string) (string, error) {
	parts := strings.Split(token, ".")
	// ... decode parts[0] into h.Alg ...

	if h.Alg == "HS256" {
		m := hmac.New(sha256.New, secret)
		m.Write([]byte(parts[0] + "." + parts[1]))
		want := base64.RawURLEncoding.EncodeToString(m.Sum(nil))
		if want != parts[2] {
			return "", errors.New("bad signature")
		}
	}

	// ... decode parts[1] into c.Sub ...
	return c.Sub, nil
}
```

完整檔案在 `recognize/verify.go`（AI 產生，不參與測試）。

<details>
<summary>參考解答</summary>

1. **只有 `h.Alg == "HS256"` 時才驗簽章，其他演算法直接跳過**：攻擊者把 header 改成 `{"alg":"none"}`（或任何非 `HS256` 的值）、payload 改成 `{"sub":"1"}`，驗證完全被略過，不需要金鑰就能偽造任何人。正確寫法：`alg` 不等於預期值就回傳錯誤，演算法由伺服器決定。
2. **沒有檢查 `exp`（也沒有 `iss`）**：簽章對的 token 永遠有效，被偷走的 token 沒有失效時間。正確寫法：解出 payload 後檢查 `exp`，沒有 `exp` 也拒絕；也要比對 `iss`。
3. **簽章用 `!=` 比對字串**：逐位元組比較在第一個不同處就結束，回應時間會洩漏猜對了多少；雖然實際攻擊較難，但正確做法很簡單。正確寫法：把簽章解碼成位元組，用 `hmac.Equal` 比較。

這份程式碼對「正常 token」與「簽章被竄改的 HS256 token」都表現正確，所以只測這兩種情況的測試看不出問題。

</details>

## 快閃卡

<details>
<summary>JWT 的三段分別是什麼？怎麼編碼？</summary>

header、payload、signature，以 . 分隔。header 與 payload 是 JSON，三段各自做 base64url 編碼。簽章是對前兩段字串算出來的。

</details>

<details>
<summary>JWT 與 session ID 在「意義存在哪裡」上有什麼不同？</summary>

session ID 只是查表的鑰匙，使用者是誰、何時過期都存在伺服器。JWT 把這些宣告直接寫在 token 裡，由簽章保證沒被改，所以驗證時不必查儲存。

</details>

<details>
<summary>token 的 payload 是 {"sub":"42"} 的 base64url，沒有簽章。為什麼不安全？</summary>

base64url 只是編碼，任何人都能解開並改成 {"sub":"1"} 再編回去，伺服器無法分辨。簽章要用只有伺服器知道的金鑰算出來，別人才改不動。

</details>

<details>
<summary>把 JWT 貼到線上的解碼工具，不用金鑰就看得到 payload。這代表簽章被破解了嗎？</summary>

不是。payload 本來就只是編碼，不是加密，簽章只防竄改、不防閱讀。所以不能把密碼或敏感資料放進 payload。

</details>

<details>
<summary>驗證時應該依 token header 的 alg 決定演算法嗎？</summary>

不應該。header 是攻擊者可以任意填的欄位，填 none 就可能讓驗證被略過。演算法要由伺服器指定，不符就拒絕。

</details>

<details>
<summary>驗證 JWT 的順序為什麼是先驗簽章、再看 payload 裡的 exp？</summary>

簽章通過之前，payload 可能被攻擊者改過，裡面的任何欄位都不可信，包含 exp。先驗簽章，才能確定要檢查的內容是伺服器簽發的。

</details>

<details>
<summary>使用者在 App 按下登出，前端刪除了 token。15 分鐘有效的 access token 被別人複製了，還能用嗎？</summary>

能，直到 exp 到期。JWT 自包含，伺服器沒有可刪除的紀錄。要縮小風險得設短效、搭配可撤銷的 refresh token，或加黑名單（但就又要查表）。

</details>

<details>
<summary>HS256 與 RS256 在金鑰上差在哪？什麼時候選後者？</summary>

HS256 簽發與驗證用同一把金鑰，驗證方也有能力簽發。RS256 用私鑰簽、公鑰驗，驗證方只拿公鑰。多個服務都要驗證、但只有登入服務該簽發時，選 RS256。

</details>

<details>
<summary>管理員在資料庫把使用者從 admin 降成 user，他手上的 JWT 是 role=admin，會發生什麼事？</summary>

在 token 過期前，伺服器仍會相信 payload 裡的 role=admin，因為簽章有效。權限寫進 token 就固定到 exp 為止，所以要查目前的權限，或把 token 效期設很短。

</details>

## 延伸閱讀

- [RFC 7519：JSON Web Token (JWT)](https://www.rfc-editor.org/rfc/rfc7519)：JWT 與標準宣告（`iss`、`sub`、`exp` 等）的原始規格
- [RFC 8725：JSON Web Token Best Current Practices](https://www.rfc-editor.org/rfc/rfc8725)：演算法驗證、宣告檢查等官方建議做法
- [Critical vulnerabilities in JSON Web Token libraries](https://auth0.com/blog/critical-vulnerabilities-in-json-web-token-libraries/)：2015 年 Tim McLean 揭露 `alg: none` 與演算法混淆的原始文章
- [jwt.io](https://jwt.io/)：貼上 token 就能看到解碼後的內容，可以親眼確認 payload 是公開的
- [golang-jwt/jwt](https://github.com/golang-jwt/jwt)：Go 社群常用的 JWT 函式庫，正式環境用它而不是自己寫
