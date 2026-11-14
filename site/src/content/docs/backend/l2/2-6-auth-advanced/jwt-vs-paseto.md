---
title: JWT vs PASETO：自包含 token 的選型
description: 能依驗證方是誰、能不能查儲存，在 opaque token、JWT 與 PASETO 之間選型，並說出換協定能消除與消除不了的風險。
sidebar:
  order: 3
---

## 先想想

短網址服務 v2 把登入拆成獨立的登入服務。它簽發 token，之後有兩個地方要驗證：

- 內部的分析服務，是我們自己寫的 Go 程式，統計點擊時要知道「是誰的連結」。
- 雲端託管的 API 閘道，給合作夥伴呼叫短網址 API。閘道只支援一種設定：填入一個 JWKS 網址，它就用那裡的公鑰驗證 token。

同事看完 JWT 的 `alg: none` 漏洞文章，提議：「PASETO 沒有 `alg` 欄位，更安全，兩邊全部換掉。」

兩個驗證方各自能換嗎？就算都換了，JWT 的風險有哪些會消失，哪些還在？

<details>
<summary>提示</summary>

分開想：「驗證方能不能讀懂這種 token」與「這種 token 的設計少掉哪些選擇」是兩個問題。再想想 [JWT 的限制](/backend/l1/1-4-auth-basics/jwt-basics/) 裡，哪些是 JWT 這個格式造成的，哪些是「自包含」這件事本身造成的。

</details>

## 是什麼

**PASETO（Platform-Agnostic SECurity TOkens）** 是一種自包含 token 的規格，格式是 `version.purpose.payload[.footer]`：用 version 固定整組演算法，用 purpose 決定是簽章還是加密，token 裡沒有任何讓人選演算法的欄位。

它和 [JWT](/backend/l1/1-4-auth-basics/jwt-basics/) 解決同一類問題：把身分與期限寫進 token，驗證方不必查儲存。差別在設計哲學：JWT 把演算法選擇留給使用者，再用最佳實務文件（RFC 8725）叮嚀別選錯；PASETO 直接把不安全的選項從協定裡拿掉。PASETO 目前不是 IETF 的 RFC，規格由社群在 GitHub 維護。

## 為什麼存在

JWT 的 header 有 `alg`，規格允許的值包含 `none`，也允許 RS256 與 HS256 並存。這帶來兩種真實的攻擊：

- **`alg: none`**：驗證程式讀 token 自己宣告的演算法，攻擊者宣告「不用簽章」，驗證被整個略過。
- **演算法混淆**：驗證程式預期 RS256（公鑰驗證），攻擊者把 header 改成 HS256，再用「公開的公鑰」當 HMAC 金鑰簽章，驗證通過。

這些攻擊不是密碼學被破解，而是「驗證方照 token 的指示做事」。防禦的責任全落在開發者的設定上：用的函式庫有沒有限定演算法、有沒有打開。一個團隊裡只要有一處忘了，就是漏洞。

PASETO 的做法是讓這類設定不存在：選了 v4，演算法就是固定的那組，token 沒有欄位可以改。這是它存在的理由，也是它最有說服力的賣點。

## 怎麼運作

### 第一步：要不要用自包含 token

比較 JWT 與 PASETO 之前，先確認真的需要自包含 token。這是整篇最重要的判斷，後面才是細節：

```text
驗證方查得到儲存嗎？
  │
  ├─ 查得到、也不介意每個請求多一次查詢
  │     → opaque token / session（預設；可立刻撤銷）
  │
  └─ 查不到，或不該查（跨服務、第三方、離線）
        │
        ├─ 驗證方是第三方，或要接 OIDC、API 閘道、IdP？
        │     → JWT（生態系是決定性因素）
        │
        └─ 簽發與驗證都在你控制的服務內？
              → JWT 或 PASETO 都行，見下面的取捨
```

這張圖的重點是：PASETO 只出現在最底下一個分支。多數第一方的網站與 App，在第一個分支就結束了，細節見〈[Session vs Token](/backend/l1/1-4-auth-basics/session-vs-token/)〉。

### PASETO 的格式

```text
v4.public.<base64url(payload || signature)>.<base64url(footer)>
│  │       └─ 簽章附在 payload 後面             └─ 選填，不加密
│  └─ purpose：public = 簽章，local = 加密
└─ version：決定演算法
```

version 與 purpose 的組合決定一切：

| 組合 | 演算法 | payload |
| --- | --- | --- |
| `v4.public` | Ed25519 簽章 | 明文，人人可讀 |
| `v4.local` | XChaCha20 加密，BLAKE2b 做驗證碼 | 密文，沒有金鑰讀不到 |
| `v3.public`、`v3.local` | ECDSA P-384、AES-256-CTR 加 HMAC-SHA384 | 同上，給必須用 NIST 演算法的環境 |

較舊的 v1、v2 已被取代，新系統用 v4（或有合規需求時用 v3）。

:::note
有些整理表把 v4.local 寫成 XChaCha20-Poly1305。那是 v2.local。v4.local 是 XChaCha20 加上 BLAKE2b 的驗證碼（先加密再驗證）。細節看規格，這裡只需要知道：加密與驗證都由協定決定，你不用選。
:::

### v4.public：簽章怎麼算

PASETO 簽的不只是 payload。它把 header、payload、footer、implicit assertion 一起用 PAE（Pre-Authentication Encoding）串起來再簽：

```go title="paseto.go"
// pae is the Pre-Authentication Encoding: it joins the pieces so that no two
// different lists of pieces can produce the same bytes.
func pae(pieces ...[]byte) []byte {
	out := le64(uint64(len(pieces)))
	for _, p := range pieces {
		out = append(out, le64(uint64(len(p)))...)
		out = append(out, p...)
	}
	return out
}
```

PAE 先寫片段數量，再對每一片寫「長度加內容」。這樣 `("ab", "c")` 與 `("a", "bc")` 會編出不同的位元組，攻擊者無法在片段之間挪動邊界。

簽發與驗證都用同一個組合：

```go title="paseto.go"
sig := ed25519.Sign(priv, pae([]byte(header), m, footer, implicit))
// ...
if !ed25519.Verify(pub, pae([]byte(header), m, footer, implicit), sig) {
	return Claims{}, nil, ErrSignature
}
```

四個片段各有用途：

| 片段 | 在 token 裡嗎 | 作用 |
| --- | --- | --- |
| header（`v4.public.`） | 是 | 被簽進去，token 不能被當成別的版本或用途重放 |
| payload（`m`） | 是 | claims，例如 `sub`、`exp`（PASETO 的 `exp` 是 RFC 3339 字串） |
| footer | 是，明文 | 放 `kid` 之類不敏感的中繼資料；被簽進去，改了就驗不過 |
| implicit assertion | 否 | 簽發與驗證雙方各自帶入的值，例如租戶編號；兩邊不一致就驗不過 |

implicit assertion 值得多看一眼：它把 token 綁在「token 外面的脈絡」。例如簽發時帶入租戶 A，租戶 B 的服務即使拿到這張 token 並有正確的公鑰，驗證也會失敗。JWT 要做到同樣的事得自己加 `aud` 宣告並記得檢查。

範例用 `go run .` 印出同樣的 claims 在兩種格式下的樣子：

```text
JWT    (EdDSA)     183 bytes  eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJ...
PASETO (v4.public) 194 bytes  v4.public.eyJpc3MiOiJzaG8ucnQiLCJzdWIiOi...
payload readable without any key: {"iss":"sho.rt","sub":"42","exp":"2026-11-14T09:15:00Z"}
```

兩者大小接近（PASETO 這一筆還帶了 17 位元組的 footer），大小不是選型理由。最後一行提醒你：`public` 只簽章，payload 照樣人人可讀，和 JWT 一樣。

### v4.local：加密，但驗證方也是簽發方

`v4.local` 用同一把對稱金鑰加密與解密。payload 是密文，拿到 token 的人讀不到內容。代價和 [HS256](/backend/l1/1-4-auth-basics/jwt-basics/) 一樣：**有能力驗證的人，就有能力簽發**。所以它只適合簽發與驗證是同一方，或是互相完全信任的少數服務。

範例沒有實作 `v4.local`，因為 XChaCha20 與 BLAKE2b 都不在 Go 標準庫裡。這也是個提醒：正式環境用維護中的函式庫，不要自己寫加密。範例的 PASETO 只實作 `v4.public`，只為了讓你看見格式。

### JWT 怎麼做到同樣安全

JWT 不是不能安全，只是安全要靠你設定。把 PASETO 做掉的選擇自己做掉：

```go title="jwt.go"
if h.Alg != "EdDSA" {
	return Claims{}, ErrAlgorithm
}
sig, err := enc.DecodeString(parts[2])
if err != nil || !ed25519.Verify(pub, []byte(parts[0]+"."+parts[1]), sig) {
	return Claims{}, ErrSignature
}
```

演算法固定成 `EdDSA`（Ed25519 用在 JOSE 的註冊名稱，RFC 8037），header 寫什麼都不採信。用 golang-jwt 時對應的是 `jwt.WithValidMethods([]string{"EdDSA"})`，authkit 的 `token` 套件就是這樣做的（它用的是 HS256，原則相同）。

換句話說，PASETO 的 version 做的是「把這兩行寫進協定」。選了 JWT，就是你自己負責這兩行。

### 回到一開始的問題

- **分析服務**：簽發與驗證都在我們控制的 Go 服務內，兩種都可以。要 PASETO 就用 `v4.public`（分析服務只需要驗證，不該有簽發權，所以不用 `local`）；要 JWT 就用固定 `EdDSA` 的做法。這裡 PASETO 是加分，不是必要。
- **雲端 API 閘道**：它只認 JWKS 與 JWT。換成 PASETO，閘道就驗不了，這邊沒有選擇。
- **換了之後消失的風險**：`alg: none`、演算法混淆這類「演算法由 token 決定」的漏洞。
- **沒有消失的風險**：到期前難以撤銷、`exp` 與 `iss` 與 `aud` 都要自己檢查、`public` 的 payload 人人可讀、金鑰外洩、權限寫進 token 會過時。這些是「自包含」的代價，與格式無關。

所以結論不是「全部換」，而是：預設用 opaque token，需要自包含時看驗證方是誰，內部自控才有餘地選 PASETO。

## 取捨與替代方案

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| opaque token（session） | 可立刻撤銷，不洩漏內容，沒有演算法可選 | 每個請求查一次儲存 | 第一方網站與 App（預設） |
| JWT（固定 EdDSA 或 ES256） | 生態系完整：OIDC、閘道、IdP、JWKS、各語言函式庫；可非對稱簽章 | 安全靠設定；加密要 JWE，複雜且少用 | 要接第三方、閘道、OIDC |
| PASETO v4.public | 沒有演算法選擇；footer 與 implicit assertion 內建 | 閘道與 IdP 幾乎不支援；函式庫選擇少；非 IETF 標準 | 內部服務間、驗證方自控 |
| PASETO v4.local | payload 預設加密，演算法固定 | 驗證方也能簽發；不能發給第三方 | 簽發與驗證同一方，且 payload 不想外露 |
| JWT + JWE | 標準內做得到加密 | 實作與除錯都最複雜 | 合規要求 token 內容加密，又必須用 JWT |

### 什麼時候不該用 PASETO

- **驗證方不是你控制的**：合作夥伴、API 閘道、IdP、前端用函式庫直接解碼 token，都預期 JWT。為了「更安全」犧牲互通，通常得不償失。
- **要接 OAuth 2.0 或 OIDC**：這些標準指定 JWT 與 JWKS，PASETO 放不進去。
- **團隊沒人能維護或審查它**：函式庫選擇少，出問題時能參考的資料也少。一個固定 `EdDSA`、設定正確的 JWT 函式庫，通常比一個沒人熟的 PASETO 函式庫更可靠。

### 什麼時候不該用自包含 token

需要立刻登出、立刻停權、立刻改權限的場景，兩者都不是好選擇。回到 opaque token，或短效 access token 搭配可撤銷的 refresh token。

## 常見坑與故障案例

### 坑 1：以為 `public` token 是加密的

症狀：有人把使用者 email、內部角色放進 `v4.public` 的 payload，後來發現前端工程師貼進解碼工具就看得到。

原因：`public` 表示「簽章」，不是「公開金鑰加密」。payload 是明文，簽章只防竄改，與 JWT 的 payload 一樣。

避免：要保密就用 `local`（且接受驗證方也能簽發），或乾脆不要把敏感資料放進 token，只放 `sub` 與 `exp`，其餘從資料庫查。範例的 `TestPublicPayloadIsReadable` 驗證了這件事。

### 坑 2：把 `local` 金鑰發給多個服務（示意案例）

症狀：分析服務被入侵後，攻擊者簽出管理員的 token，登入服務與所有其他服務都接受。

原因：`local` 是對稱的，能解密就能簽發。為了「payload 不外露」把金鑰發給五個服務，等於有五個地方可以簽發 token。

避免：驗證方多、簽發方只有一個時，用 `public`：私鑰只放在登入服務，其他服務只有公鑰。

### 坑 3：先信任 footer，再驗簽章

症狀：程式從 footer 讀出 `kid` 去找金鑰，這本身沒問題；但後面又直接讀 footer 裡的 `tenant`、`role` 之類欄位來做授權，沒有確認簽章已經驗過。

原因：footer 在驗證通過之前，是攻擊者可以任意填的輸入。PASETO 的簽章確實涵蓋 footer，但前提是你真的完整驗證了，而且正確使用 PAE。

避免：驗證順序是「從 footer 取 `kid`，只在自己持有的金鑰集合裡查，查不到就拒絕，找到再驗簽章，通過後才信任 footer」。不要從 token 內容裡取金鑰本身（像是 JWT 的 `jwk`、`jku` header），只能用 `kid` 查自己的金鑰表。範例的 `TestPublicFooterIsAuthenticated` 驗證了改 footer 會導致簽章失敗。

### 坑 4：換了協定，卻沒檢查 `exp`、`iss`、`aud`

症狀：改用 PASETO 後，被偷走的 token 仍然可以無限期使用；A 服務簽的 token 被拿到 B 服務也通過。

原因：簽章只證明「這是我們簽的、沒被改」。有沒有過期、是不是簽給這個服務，仍然要驗證方自己檢查。PASETO 的 `exp` 是 RFC 3339 字串，函式庫通常會處理，但不是所有函式庫預設都檢查。

避免：簽發一律設 `exp`，驗證一律檢查，沒有 `exp` 當成錯誤；用 `iss` 與 implicit assertion（或 JWT 的 `aud`）把 token 綁在預期的簽發方與接收方。

### 坑 5：還在用沒限定演算法的 JWT 函式庫

症狀：攻擊者不需要金鑰就能偽造任何使用者。

原因：驗證程式讀 token 自己宣告的 `alg`。2015 年 Tim McLean 公開了多個主流 JWT 函式庫接受 `alg: none` 與演算法混淆（見延伸閱讀），PASETO 的設計正是對這類問題的回應。

避免：不一定要換 PASETO。驗證時用 `jwt.WithValidMethods` 限定演算法，並讓測試涵蓋 `none` 與錯誤的演算法，範例的 `TestJWTRejectsOtherAlgorithms` 就是這個測試。詳見〈[JWT 的基礎與限制](/backend/l1/1-4-auth-basics/jwt-basics/)〉的坑 1。

## 小練習

範例放在 `playground/backend/l2/2-6-auth-advanced/jwt-vs-paseto/`，執行 `go test ./...` 可以驗證，`go run .` 會印出兩種 token 的長度與 payload。

### 動手寫

目前 `token` 套件只能用單一公鑰驗證。金鑰輪替時要讓新舊兩把公鑰同時有效。請在 `token/keyring.go` 新增：

```go
var ErrUnknownKey = errors.New("token: unknown key id")

func VerifyKeyring(keys map[string]ed25519.PublicKey, tok, iss string, now time.Time) (Claims, error)
```

簽發時 footer 是 `{"kid":"2026-11"}`。`VerifyKeyring` 從 footer 取出 `kid`，只在 `keys` 裡查金鑰，找不到回傳 `ErrUnknownKey`，找到就呼叫 `VerifyPublic`。沒有 footer 或 footer 不是合法 JSON 時回傳 `ErrFormat`。

寫三個測試：新舊兩把金鑰都在 `keys` 裡時，兩種 `kid` 的 token 都通過；把舊金鑰從 `keys` 移除後，舊 token 回傳 `ErrUnknownKey`；把 token 的 footer 換成另一個存在的 `kid`，回傳 `ErrSignature`。

<details>
<summary>參考解答</summary>

```go title="keyring.go"
func VerifyKeyring(keys map[string]ed25519.PublicKey, tok, iss string, now time.Time) (Claims, error) {
	rest, ok := strings.CutPrefix(tok, header)
	if !ok {
		return Claims{}, ErrFormat
	}
	parts := strings.Split(rest, ".")
	if len(parts) != 2 {
		return Claims{}, ErrFormat
	}
	fb, err := enc.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrFormat
	}
	var f struct {
		KID string `json:"kid"`
	}
	if err := json.Unmarshal(fb, &f); err != nil {
		return Claims{}, ErrFormat
	}
	pub, ok := keys[f.KID]
	if !ok {
		return Claims{}, ErrUnknownKey
	}
	c, _, err := VerifyPublic(pub, tok, iss, nil, now)
	return c, err
}
```

重點有兩個：金鑰只從自己的 `keys` 查，不用 token 裡帶的任何金鑰；`kid` 只是索引，真正的信任來自後面 `VerifyPublic` 的簽章驗證。第三個測試會通過，是因為 footer 被簽進去了：換成另一個 `kid` 會讓簽章不符。

</details>

### 辨識

下面是 AI 產生的 `v4.public` 驗證函式。用同一組程式簽發的 token 能正常驗證，換一把金鑰簽的 token 也會被擋下來。請做 code review，找出 3 個問題。

```go title="verify.go"
func verifyPublic(tok string, keys map[string]ed25519.PublicKey) (string, error) {
	parts := strings.Split(tok, ".")
	body, err := enc.DecodeString(parts[2])
	// ... split body into msg and sig ...

	fb, err := enc.DecodeString(parts[3])
	// ... decode footer into f.KID, look up keys[f.KID] ...

	if !ed25519.Verify(pub, msg, sig) {
		return "", errors.New("bad signature")
	}

	// ... decode msg into c ...
	return c.Sub, nil
}
```

完整檔案在 `recognize/verify.go`（AI 產生，不參與測試）。

<details>
<summary>參考解答</summary>

1. **直接取 `parts[2]` 與 `parts[3]`，沒有檢查切出幾段，也沒檢查前兩段是 `v4` 與 `public`**：沒有 footer 的 token（`v4.public.xxx`，只有三段）會觸發 index out of range 的 panic；沒有 recover 的話，一個畸形 token 就能讓請求處理失敗甚至拖垮行程。也會接受 `v3` 或 `local` 之類不該接受的前綴。正確寫法：先檢查前綴與段數，不符就回傳 `ErrFormat`。
2. **簽章只驗 `msg`，沒有用 PAE 把 header、footer、implicit assertion 一起驗**：footer 沒有被簽進去，攻擊者可以任意改 footer 而簽章仍然通過；header 也沒被綁定。這個實作也不符合規格，用標準函式庫簽發的 token 會被它拒絕，只有用同一組錯誤程式簽發的 token 才驗得過。正確寫法：驗證 `pae(header, msg, footer, implicit)`，與簽發端一致。
3. **簽章通過後沒有檢查 `exp`（也沒有 `iss`）**：簽發過的 token 永遠有效，被偷走的 token 沒有失效時間。正確寫法：解出 claims 後檢查 `exp`，沒有 `exp` 也拒絕，並比對 `iss`。

做對的地方是：金鑰只從自己的 `keys` 表用 `kid` 查，沒有信任 token 內帶的金鑰。

這份程式碼對「自己簽發的正常 token」與「別的金鑰簽的 token」都表現正確，所以只測這兩種情況的測試看不出問題。

</details>

## 快閃卡

<details>
<summary>PASETO 的 token 格式 v4.public.xxx 中，version 與 purpose 各自決定什麼？</summary>

version 決定整組演算法（v4 是 Ed25519 與 XChaCha20 加 BLAKE2b），purpose 決定用途：public 是簽章，local 是加密。token 裡沒有可以選演算法的欄位。

</details>

<details>
<summary>JWT 的 alg: none 與演算法混淆攻擊，問題出在哪？</summary>

驗證程式照 token 自己宣告的 alg 做事，而 header 是攻擊者可以任意填的。防禦要靠驗證方限定演算法，PASETO 把這件事寫進協定。

</details>

<details>
<summary>v4.public 的 payload 可以被沒有金鑰的人讀到嗎？</summary>

可以。public 只簽章，不加密，payload 是明文，簽章只防竄改。要保密得用 local，或乾脆不放敏感資料。

</details>

<details>
<summary>PASETO 簽章為什麼要用 PAE，而不是直接把 header、payload、footer 接起來簽？</summary>

直接接起來，("ab", "c") 與 ("a", "bc") 會得到同樣的位元組，攻擊者可以挪動片段邊界。PAE 先寫片段數與每片的長度，不同的片段組合不會編出同樣的結果。

</details>

<details>
<summary>登入服務簽發 v4.local token，發給分析服務用同一把金鑰解密驗證。這有什麼風險？</summary>

local 是對稱的，能驗證就能簽發。分析服務被入侵時，攻擊者可以簽出任何使用者的 token。驗證方多於簽發方時，應該用 public，只在登入服務放私鑰。

</details>

<details>
<summary>你的雲端 API 閘道設定只接受 JWKS 網址。要換成 PASETO 讓 token 更安全，可行嗎？</summary>

不可行。閘道只認 JWT 與 JWKS，驗不了 PASETO。生態系與互通是這個情境的決定性因素，演算法安全性可以靠固定 EdDSA 或 ES256 補足。

</details>

<details>
<summary>收到 token，footer 是 {"kid":"k2"}。應該怎麼用這個 kid 找金鑰？</summary>

只在自己持有的金鑰集合裡用 kid 查，查不到就拒絕，找到再驗簽章，通過後才信任 footer 的內容。不能直接採用 token 帶的金鑰。

</details>

<details>
<summary>改用 PASETO 後，使用者登出，他手上的 v4.public token 還能用嗎？</summary>

能，直到 exp。自包含 token 的撤銷困難與格式無關，PASETO 一樣有。要立刻失效得用 opaque token，或短效 token 搭配可撤銷的 refresh token。

</details>

<details>
<summary>v4.public token 簽發時帶了 implicit assertion "tenant-a"，另一個服務用 "tenant-b" 驗證同一張 token，結果如何？</summary>

驗證失敗（簽章不符）。implicit assertion 不在 token 裡，但被簽進去，雙方帶入的值必須一致，所以它能把 token 綁在 token 外面的脈絡。

</details>

## 延伸閱讀

- [PASETO 規格（paseto-standard/paseto-spec）](https://github.com/paseto-standard/paseto-spec)：版本、PAE、`v4.public` 與 `v4.local` 的原始定義
- [RFC 8725：JSON Web Token Best Current Practices](https://www.rfc-editor.org/rfc/rfc8725)：官方整理的 JWT 防禦做法，包含限定演算法與檢查宣告
- [RFC 8037：CFRG Elliptic Curve in JOSE](https://www.rfc-editor.org/rfc/rfc8037)：JWT 使用 Ed25519（`EdDSA`）的規格
- [RFC 7517：JSON Web Key (JWK)](https://www.rfc-editor.org/rfc/rfc7517)：JWKS 與金鑰發布的格式
- [Critical vulnerabilities in JSON Web Token libraries](https://auth0.com/blog/critical-vulnerabilities-in-json-web-token-libraries/)：2015 年揭露 `alg: none` 與演算法混淆的原始文章
- [aidantwoods/go-paseto](https://github.com/aidantwoods/go-paseto)：Go 的 PASETO 函式庫，正式環境用它而不是自己寫
