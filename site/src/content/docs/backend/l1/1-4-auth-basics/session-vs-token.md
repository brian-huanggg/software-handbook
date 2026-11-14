---
title: Session vs Token：有狀態與無狀態認證的取捨與選型
description: 能說出 session 與 token 把狀態放在哪裡、撤銷得了什麼，並依情境選型。
sidebar:
  order: 5
---

## 先想想

短網址服務 v1 現在有網頁，也有手機 App。後端從一台擴成三台，前面放負載平衡器。團隊分成兩派：一派要把 session 存進 Redis，另一派要全改成 JWT，理由是「無狀態，好擴展」。

某天有使用者回報帳號被盜，客服要求「立刻把這個人所有登入踢掉」。

兩種做法各自做得到嗎？如果 JWT 派堅持要做到，他們得加什麼東西？加完之後，「無狀態」還剩下什麼？

<details>
<summary>提示</summary>

先想「踢掉一個登入」在伺服器上具體是哪個動作：刪掉一筆紀錄？還是等一個時間到？

</details>

## 是什麼

**有狀態認證（stateful authentication）** 是伺服器為每次登入保存一筆紀錄、每個請求靠查這筆紀錄確認身分的做法；**無狀態認證（stateless authentication）** 是身分與期限直接寫在憑證裡、伺服器只驗簽章而不保存任何紀錄的做法。

「Session vs Token」其實是在問同一件事：**登入狀態放在哪裡**。前者放在伺服器的儲存（[session](/backend/l1/1-4-auth-basics/session-cookie/)），後者放在客戶端手上的憑證（常見是 [JWT](/backend/l1/1-4-auth-basics/jwt-basics/)）。Cookie 只是傳遞憑證的方式，不是 session 與 token 的對立面：session ID 可以放 Cookie，JWT 也可以放 Cookie。

## 為什麼存在

這個選擇是從「一台變多台」開始變難的。

單機時，session 放程序記憶體就能運作。擴成三台、請求輪流分配後，使用者在 A 登入，之後的請求只有約三分之一會落到 A，其餘約三分之二找不到 session 而被當成未登入（依輪詢估算）。使用者的感受是「一直被登出」。

解法有兩條，剛好對應兩種認證：

- **把狀態搬到共用儲存**：三台都去查同一個 Redis，仍是有狀態，只是狀態不在各台程序裡。
- **把狀態塞進憑證**：每台只要有金鑰就能驗，不需要共用任何東西。

兩條路都解決了「多台」，代價不同，下面分開講。

## 怎麼運作

### 狀態放在哪裡

```text
有狀態（session）                      無狀態（token）

 客戶端 ──session ID──▶ 伺服器          客戶端 ──token──▶ 伺服器
                          │                                 │
                          ▼                                 ▼
                     查共用儲存                        驗簽章與 exp
                   （Redis / DB）                    （只需要金鑰）
                          │                                 │
                    得到 user=42                      token 內就有 sub=42
```

這張圖的重點是：左邊每個請求都多一次儲存的往返，右邊只有計算。

範例用同一個介面實作兩種做法。有狀態的 `Authenticate` 每次都讀儲存：

```go title="session.go"
s.lookups++ // every request pays one store read
r, ok := s.records[id]
if !ok || !s.now().Before(r.expires) {
	delete(s.records, id)
	return "", ErrInvalid
}
```

無狀態的 `Authenticate` 只重算簽章、檢查期限，不碰任何儲存：

```go title="token.go"
got, err := enc.DecodeString(sig)
if err != nil || !hmac.Equal(got, t.mac(payload)) {
	return "", ErrInvalid
}
```

成本差距的量級是：同機房一次 Redis 往返通常在 1 毫秒上下（估算，視網路與負載而定），HMAC 驗證是微秒級。但多數 v1 規模的服務，一個請求本來就要查好幾次資料庫，多這一次不是瓶頸；它真正的代價是 Redis 變成每個請求都依賴的元件。

### 撤銷：真正的分水嶺

兩種做法最大的差別，不在效能，而在「事後能不能收回」：

```go title="token.go"
// Logout cannot do anything: there is no server-side record to delete, so
// every copy of the token stays valid until it expires.
func (t *Token) Logout(string) error { return nil }
```

Session 的 `Logout` 是 `delete(s.records, id)`，下一個請求就失效。Token 的 `Logout` 是空的：伺服器沒有任何東西可刪。執行 `go run .` 可以看到：

```text
session, after logout:       rejected
token, after logout:         still accepted
```

要讓 token 也能被踢掉，唯一的辦法是在伺服器記一份「已作廢」清單，每個請求都先查。這就是查表，狀態又回來了。

### 折衷：短效 access token 加可撤銷 refresh token

多數 App 採用的是中間路線，用兩個憑證分工：

```text
login ──▶ access token（5 分鐘，無狀態驗證）
      └─▶ refresh token（較長，伺服器保存，可刪除）

每個請求   帶 access token，只驗簽章，不查表
5 分鐘後   用 refresh token 換新的一組；舊 refresh token 作廢（輪替）
登出       刪掉 refresh token；access token 最多再撐 5 分鐘
```

這張圖的重點是：狀態沒有消失，只是從「每個請求都查」縮到「每 5 分鐘查一次」，撤銷的延遲上限變成 access token 的效期。

```go title="hybrid.go"
r, ok := h.refresh[refresh]
delete(h.refresh, refresh)
if !ok || !h.now().Before(r.expires) {
	return Pair{}, ErrInvalid
}
```

`Refresh` 先刪再判斷：不論成功與否，這個 refresh token 都只能用一次（**refresh token 輪替**，refresh token rotation）。

回到一開始的問題：三台伺服器共用 Redis 的 session 做得到「立刻踢掉所有登入」，刪掉那個使用者的所有 session 紀錄即可。純 JWT 做不到，在 `exp` 前每份副本都有效。JWT 派若堅持要做到，就得加黑名單或「使用者的 token 版本號」，而這兩者都是每個請求要查的伺服器端狀態，他們的「無狀態」只剩下「簽章驗證不用查」這一半。

## 取捨與替代方案

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| Session + 共用儲存 | 立即撤銷、權限即時生效、憑證小 | 每個請求一次儲存往返；儲存是依賴與單點 | 瀏覽器為主的網站，v1 的首選 |
| 純無狀態 token | 驗證不查儲存；跨服務傳遞方便 | 到期前不能撤銷；權限固定到過期；金鑰要管理 | 服務間呼叫、短效的一次性授權 |
| 短效 access + refresh token | 多數請求不查表，仍能撤銷 | 多一套 refresh 發放、輪替與儲存；撤銷有延遲 | 手機 App、多服務且要能登出 |
| 不透明 token（隨機字串加查表） | 非瀏覽器客戶端也能用，隨時撤銷，不洩漏內容 | 每個請求查表 | 給第三方的 API 金鑰、要即時撤銷的 API |

選型時依序問三個問題：

1. **要不要即時撤銷？** 要的話，每個請求都得有一個伺服器端的檢查點，選 session 或不透明 token。
2. **客戶端是誰？** 瀏覽器有 Cookie 機制，session 最順手；原生 App 與第三方客戶端沒有，改用 `Authorization` header 帶 token。
3. **有幾個服務各自要驗證？** 只有一個後端，查表很便宜；很多服務各自驗證時，無狀態驗證才省得下共用儲存的耦合。

### 什麼時候不該用

- **不該因為「無狀態比較現代」就選 token**：只有一個後端、一個網站的 v1，session 更簡單，撤銷與停權都是刪紀錄。
- **不該用純無狀態 token 做需要即時撤銷的場景**：後台管理、金流操作、帳號被盜的處置，都不能等 `exp`。
- **不該一開始就上 access 加 refresh 的完整架構**：它多出的發放、輪替與儲存都是要維護的程式。需求出現（有 App、有多服務）再升級。

## 常見坑與故障案例

### 坑 1：為了登出加黑名單，卻說自己是無狀態

症狀：系統宣稱「無狀態，可水平擴展」，登出卻要查一個 Redis 黑名單，Redis 一慢全站變慢。

原因：撤銷需要伺服器端狀態，這是無法繞開的。黑名單等於把 session 表換個名字；而且黑名單要共用，單機記憶體版在多台下會失效。

避免：先決定要不要即時撤銷。要，就老實用 session 或不透明 token；不要，才用純無狀態，並接受到期前不能收回。

### 坑 2：access token 效期太長

症狀：使用者改了密碼、被停權，舊 token 仍能操作好幾小時甚至好幾天。

原因：撤銷延遲的上限就是 access token 的效期。為了「使用者不用一直重登」把它設成 24 小時或 30 天，等於把折衷方案退化成純無狀態 token。

避免：access token 設 5 到 15 分鐘，體驗靠 refresh token 補，不靠拉長 access token。

### 坑 3：refresh token 也做成無狀態 JWT

症狀：使用者登出，refresh token 還能一直換新的 access token；被偷走的 refresh token 等於長期有效的登入。

原因：refresh token 的存在理由就是「伺服器能撤銷」。若它也只靠簽章驗證，伺服器沒有東西可刪，整個折衷方案就失去撤銷能力。只在前端清掉 Cookie 也一樣，只是自己不用，別人手上的副本不受影響。

避免：refresh token 用不透明的隨機值（`crypto/rand`）存在伺服器，每次換新就輪替，登出就刪除。

### 坑 4：權限寫進 token，改了不生效

症狀：管理員把某人從 admin 降成 user，他在 token 過期前仍能做管理操作。

原因：權限固定在簽發時的 payload 裡，伺服器驗簽章通過就信任它。這與 session 內存權限會過時是同一個坑（見 [Session 與 Cookie](/backend/l1/1-4-auth-basics/session-cookie/)），只是 token 連「改伺服器端紀錄」都做不到。

避免：token 只放身分（`sub`），敏感操作前查目前的權限；或接受權限變更要等 access token 過期才生效，並把效期設短。

### 坑 5：session 放在單機記憶體（示意案例）

症狀：服務從一台擴成三台後，客服湧入「一直被登出」的回報，單機測試完全正常。

原因：session 只存在登入當下那台的記憶體，負載平衡器把後續請求分給別台。重啟或部署時，所有記憶體內的 session 也一起消失。

避免：session 一開始就放共用儲存（Redis 或資料庫），或確認要擴機前改好；不要用「session 保持」（讓同一個使用者固定到同一台）當長期解法，它在那台故障時一樣會掉登入。

## 小練習

範例放在 `playground/backend/l1/1-4-auth-basics/session-vs-token/`，執行 `go test ./...` 可以驗證，`go run .` 會印出三種做法登出後的結果。

### 動手寫

目前 `Token` 沒有辦法「登出所有裝置」。請在 `authn/token.go` 加上：

1. `Token` 內保存一個 `map[string]int`，記錄每個使用者目前的版本號（預設 0），用 `sync.Mutex` 保護。
2. `claims` 新增 `Ver int`（JSON 欄位 `ver`），`Login` 簽發時寫入該使用者目前的版本。
3. 新增 `RevokeAll(userID string)`，把該使用者的版本加一；`Authenticate` 在 `Ver` 與目前版本不同時回傳 `ErrInvalid`。

寫一個測試：同一個使用者登入兩次得到兩個 token，呼叫 `RevokeAll` 後兩個都被拒絕，重新登入得到的新 token 可用。完成後想想：你的 `Authenticate` 現在每次做了什麼，和 `Session` 有什麼不同？

<details>
<summary>參考解答</summary>

```go
t.mu.Lock()
ver := t.versions[userID]
t.mu.Unlock()
// ...sign claims{Sub: userID, Exp: ..., Ver: ver}

// in Authenticate, after the signature and exp checks:
t.mu.Lock()
current := t.versions[c.Sub]
t.mu.Unlock()
if c.Ver != current {
	return "", ErrInvalid
}
```

`RevokeAll` 是 `t.versions[userID]++`（持鎖）。

回答：`Authenticate` 現在每個請求都讀一次 `versions`，這就是一次查表。多台伺服器時，`versions` 必須放進共用儲存，結果和 session 一樣是每個請求一次儲存往返，只是存的資料更小。想要「能立即撤銷」，就得付出這個成本。

</details>

### 辨識

下面是 AI 產生的 refresh 與登出處理程式。程式碼能編譯，簽章被竄改的 token 會被擋下，過期的 token 也會被拒絕。請做 code review，找出 3 個問題。

```go title="refresh.go"
func refresh(w http.ResponseWriter, r *http.Request) {
	ck, err := r.Cookie("refresh")
	// ... err handling ...
	c, err := verify(ck.Value)
	// ... err handling ...
	access := sign(Claims{Sub: c.Sub, Role: c.Role, Exp: time.Now().Add(24 * time.Hour).Unix()})
	w.Write([]byte(access))
}

func logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "refresh", Value: "", MaxAge: -1, Path: "/"})
	w.WriteHeader(http.StatusNoContent)
}

func adminOnly(next http.HandlerFunc) http.HandlerFunc {
	// ... c, err := verify(bearer token); if err != nil || c.Role != "admin" → 403
}
```

完整檔案在 `recognize/refresh.go`（AI 產生，不參與測試）。

<details>
<summary>參考解答</summary>

1. **refresh token 只靠簽章與 `exp` 驗證，伺服器不存它，`logout` 也只是清 Cookie**：登出後，別人手上複製的 refresh token 仍然能一直換新的 access token，直到它自己過期；伺服器沒有任何方法撤銷。正確寫法：refresh token 用 `crypto/rand` 產生的不透明隨機值，存在伺服器；`refresh` 先查表、換新時輪替（刪舊發新），`logout` 要刪除伺服器端的紀錄，而不只是清 Cookie。
2. **新簽發的 access token 效期是 24 小時**：撤銷、停權、改密碼後，被偷的 access token 最長還能用一整天。正確寫法：5 到 15 分鐘，使用者體驗交給 refresh token。
3. **`adminOnly` 直接信任 token 裡的 `Role`，`refresh` 還把舊的 `Role` 原封不動複製到新 token**：管理員被降權後，只要持續換新，舊的 `admin` 角色會一直延續下去。正確寫法：token 只放 `sub`，`refresh` 與敏感操作都重新查目前的角色。

這份程式碼對「簽章被竄改」與「過期」兩種情況都表現正確，所以只測這兩種的測試看不出問題。

</details>

## 快閃卡

<details>
<summary>有狀態認證與無狀態認證，差在「狀態」放在哪裡？</summary>

有狀態把登入紀錄放在伺服器的儲存，憑證只是查表的鑰匙。無狀態把身分與期限寫進憑證，由簽章保證沒被改，伺服器不保存任何紀錄。

</details>

<details>
<summary>Cookie 是 session 的同義詞嗎？JWT 可以放進 Cookie 嗎？</summary>

不是。Cookie 只是瀏覽器保存與自動附帶憑證的方式。放進去的可以是 session ID，也可以是 JWT，所以 Session vs Token 比的是狀態放哪裡，不是用不用 Cookie。

</details>

<details>
<summary>一個服務從一台擴成三台，session 放在各台的記憶體，輪詢分配請求。約有多少比例的後續請求會被當成未登入？</summary>

約三分之二。登入那台只佔三台之一，其餘請求落到沒有該 session 的機器。解法是把 session 放到共用儲存，或改用無狀態 token。

</details>

<details>
<summary>每個請求，session 與純無狀態 token 各多做了什麼事？</summary>

Session 每個請求要查一次儲存（例如 Redis）。無狀態 token 只重算簽章、檢查 exp，不碰儲存，所以儲存不是每個請求都依賴的元件。

</details>

<details>
<summary>使用者按下登出，伺服器呼叫 Logout。純無狀態 token 與 session 的結果有什麼不同？</summary>

Session 的 Logout 刪掉伺服器上的紀錄，下一個請求就失效。純無狀態 token 的 Logout 沒有東西可刪，任何一份副本在 exp 前都仍然通過驗證。

</details>

<details>
<summary>access token 5 分鐘、refresh token 24 小時。登出後 3 分鐘，複製到舊 access token 的人還能呼叫 API 嗎？能換新的 token 嗎？</summary>

還能呼叫 API，到 5 分鐘到期為止。但不能換新：登出刪掉了伺服器上的 refresh token，換新請求會被拒絕。撤銷的延遲上限就是 access token 的效期。

</details>

<details>
<summary>為什麼 Refresh 要在判斷之前就刪掉舊的 refresh token？</summary>

這是輪替：每個 refresh token 只能用一次。不論成功與否都刪掉，被偷走的舊 token 在合法使用者換過之後就失效，不能重複使用。

</details>

<details>
<summary>團隊想「為了登出」在純無狀態 token 上加一份黑名單，每個請求先查黑名單。這個系統還算無狀態嗎？</summary>

不算完整的無狀態。每個請求都要讀一份伺服器端的狀態，和 session 的查表是同一類成本。只剩簽章驗證本身不用查而已。

</details>

<details>
<summary>管理員把使用者從 admin 降成 user，他手上 role=admin 的 15 分鐘 token 發生什麼事？怎麼降低風險？</summary>

在 token 過期前仍被信任為 admin，因為簽章有效。降低風險：token 只放 sub、效期設短，敏感操作前再查目前的權限。

</details>

## 延伸閱讀

- [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)：session ID、過期、登出與 Cookie 屬性的安全做法
- [RFC 6749：The OAuth 2.0 Authorization Framework](https://www.rfc-editor.org/rfc/rfc6749)：access token 與 refresh token 分工的原始規格
- [RFC 9700：Best Current Practice for OAuth 2.0 Security](https://www.rfc-editor.org/rfc/rfc9700)：refresh token 輪替與外洩對策的官方建議
- [RFC 8725：JSON Web Token Best Current Practices](https://www.rfc-editor.org/rfc/rfc8725)：使用 JWT 時驗證與宣告檢查的建議做法
- [Auth0：Refresh Token Rotation](https://auth0.com/docs/secure/tokens/refresh-tokens/refresh-token-rotation)：輪替與重複使用偵測的實務說明
