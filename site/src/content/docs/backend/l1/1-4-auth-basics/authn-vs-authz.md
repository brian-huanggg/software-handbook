---
title: 認證（authn）vs 授權（authz）
description: 能分清「你是誰」與「你能做什麼」，並在 Go 服務中把兩者拆成各自的一層，擋下 IDOR。
sidebar:
  order: 1
---

## 先想想

短網址服務 v1 加了登入。現在有 `GET /links/{code}` 與 `DELETE /links/{code}`，alice 建了短網址 `/links/1`。

同事把這兩個 endpoint 都包在「必須登入」的 middleware 後面，然後說：「安全了，沒登入的人碰不到。」

bob 也是合法使用者，他登入後送出 `DELETE /links/1`。會發生什麼事？同事漏掉了哪一件事？

<details>
<summary>提示</summary>

「必須登入」只回答了「這個人是誰」。想想 handler 裡有沒有任何一行在問「這個人可不可以動這一筆資料」。

</details>

## 是什麼

**認證（authentication，簡稱 authn）** 是確認呼叫者是誰；**授權（authorization，簡稱 authz）** 是決定這個已確認身分的人，可以對哪些資源做哪些事。

兩者是前後相接的兩道關卡，回答不同的問題：認證問「你是誰？」，授權問「你可以做這件事嗎？」。授權依賴認證的結果，沒有身分，就無從判斷權限；但認證通過，不代表任何事都能做。

## 為什麼存在

分成兩件事，是因為它們變化的速度與原因都不同。

認證只在入口發生一次：驗證密碼、session 或 token，產出「這個請求來自 alice」。授權卻在每個動作都要發生：同樣是 alice，可以刪自己的短網址，不能刪 bob 的。

沒有把授權獨立出來的時候，常見後果是：每個 handler 各寫各的檢查，有的有、有的漏。漏掉的那個就是漏洞。OWASP 的 2021 年 Top 10 把「權限控制失效（Broken Access Control）」列為第一名（A01），這類漏洞的共通點，幾乎都是「身分確認了，但沒檢查這個人能不能做這件事」。

把兩者拆開以後，認證集中在一個入口，授權集中在一個判斷函式，各自可以單獨測試、單獨修改。

## 怎麼運作

### 兩道關卡

```text
請求 ──▶ [認證 authn] ──▶ [授權 authz] ──▶ 執行動作
          你是誰？         可以做這件事嗎？
            │                 │
            ▼                 ▼
    失敗：401 Unauthorized   失敗：403 Forbidden
    （不知道你是誰）          （知道你是誰，但不行）
```

先看兩個失敗出口：認證失敗代表「身分不明」，授權失敗代表「身分明確、但被拒絕」。狀態碼要對應這兩種情況。

HTTP 規格（RFC 9110）定義 401 為「缺少有效的認證憑證」，回應必須帶 `WWW-Authenticate` header；403 為「伺服器理解請求，但拒絕執行」。401 的名字叫 Unauthorized，但意思其實是 unauthenticated，這是歷史遺留的命名。

### 認證層：只回答「是誰」

範例的 middleware 只做一件事：從 `Authorization` header 找出使用者，放進 `context`，找不到就回 401。它不判斷使用者能做什麼。

```go title="auth.go"
user, ok := a.users[token]
if !ok {
	unauthorized(w)
	return
}
ctx := context.WithValue(r.Context(), ctxKey{}, user)
next.ServeHTTP(w, r.WithContext(ctx))
```

這裡用一張 token 對照表代替真正的驗證機制。[Session 與 Cookie](/backend/l1/1-4-auth-basics/session-cookie/) 與 [JWT 的基礎與限制](/backend/l1/1-4-auth-basics/jwt-basics/) 會換成真實做法，但 middleware 的角色不變：輸出一個身分，或回 401。

### 授權層：只回答「可不可以」

授權集中成一個純函式：輸入是使用者、動作、資源，輸出是 `bool`。

```go title="policy.go"
func Allowed(u User, a Action, l Link) bool {
	switch a {
	case ActionView:
		return l.OwnerID == u.ID
	case ActionDelete:
		return l.OwnerID == u.ID || u.Role == RoleAdmin
	default:
		return false
	}
}
```

三個細節：

- 判斷需要 **資源**（這一筆短網址是誰的），所以它必須在資料取出之後做，不能只在 middleware 看網址。
- 規則依 **動作** 而異：管理員能刪任何短網址（處理濫用），但看不到別人的私有連結。
- `default` 回 `false`：沒列出的動作一律拒絕（deny-by-default），新增動作時忘了寫規則，結果是被擋住，而不是放行。

### 每個動作都要過這一關

每個會碰到某一筆資料的 handler，先取資料、再問政策：

```go title="links.go"
l, ok := s.store.Get(r.PathValue("code"))
if !ok {
	http.NotFound(w, r)
	return Link{}, false
}
if !Allowed(user, a, l) {
	http.Error(w, "forbidden", http.StatusForbidden)
	return Link{}, false
}
```

另外，建立短網址時，擁有者 **來自認證結果，不來自請求內容**：

```go title="links.go"
// The body has no owner field: the owner is whoever authn says is calling.
var in struct {
	URL string `json:"url"`
}
// ...
l := s.store.Create(user.ID, in.URL)
```

回到一開始的問題：bob 的 `DELETE /links/1` 會成功。同事的 middleware 只確認了「bob 是合法使用者」，沒有任何地方問「bob 能不能刪這一筆」，所以 alice 的短網址被刪了。漏掉的是授權：要在 handler 裡比對資源的擁有者，不符就回 403。範例的 `TestForbiddenDeleteKeepsLink` 驗證 bob 被拒絕後，連結仍然存在。

## 取捨與替代方案

授權規則可以放在不同位置：

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| handler 裡各自 `if` 判斷 | 最直覺、不需要抽象 | 規則散落，容易漏、難審查 | 只有一兩個 endpoint |
| 集中的 policy 函式（本文） | 規則一處可讀、可單獨測試 | 每個 handler 仍要記得呼叫 | 規則不多、以「擁有者」為主 |
| 角色型（RBAC）：依角色授權 | 規則簡單，新增使用者只要指定角色 | 不能表達「只有這一筆的擁有者」這類條件 | 後台、有明確職位分工 |
| 外部授權服務或規則引擎 | 規則可獨立更新、跨服務一致 | 多一個依賴與延遲，要學新語法 | 多服務、規則複雜的組織 |

本文的政策同時用到角色（admin）與擁有者關係，這種混合在實務上很常見。RBAC 與更細粒度的模型留到 L4 的權限系統一節。

失敗時回 403 還是 404 也是取捨：403 明確，但告訴對方「這一筆存在」；對 bob 回 404，則不洩漏資源是否存在，代價是除錯時更難分辨「真的不存在」與「沒權限」。多數 API 用 403，資源存在與否本身就是祕密時才用 404。

### 什麼時候不該用

- **完全公開的資源**：公開的短網址跳轉（`GET /{code}`）任何人都能用，不需要認證，也沒有授權可判斷。硬加登入只會擋掉正常使用者。
- **只有一個使用者的內部工具**：所有人都是管理員時，拆出政策函式只是多寫程式；先把認證做對即可，等出現第二種角色再拆。

## 常見坑與故障案例

### 坑 1：只做認證，沒做授權（IDOR）

症狀：登入後把網址裡的 `1` 改成 `2`，就看到別人的資料；或像前面的情境，能刪別人的短網址。

原因：這是 IDOR（insecure direct object reference，不安全的直接物件參照）：只靠 ID 取資料，沒有檢查呼叫者與資料的關係。OWASP API Security Top 10（2023）把它放在第一位，名稱是 Broken Object Level Authorization（API1）。

避免：每個以 ID 存取單一資源的 handler，都走同一個「取資料、再問政策」的函式，如範例的 `load`。XSS、CSRF 與其他常見漏洞見 [常見漏洞入門](/backend/l1/1-4-auth-basics/common-vulnerabilities/)。

### 坑 2：信任客戶端送來的身分或角色

症狀：一般使用者在請求加上 `X-Role: admin`，就能執行管理員操作；或在 JSON 加上 `"owner_id": "alice"`，替別人建立資料。

原因：header、Cookie 內容、請求 body 都是客戶端自己填的，不能當事實。身分只能來自認證層驗證過的結果。

避免：角色與使用者 ID 從 `context` 取（認證的產出），再從資料庫讀權限；請求 body 的結構體不要放 `owner_id`、`role` 這類欄位。`TestOwnerComesFromIdentityNotBody` 驗證 body 裡的 `owner_id` 被忽略。

### 坑 3：401 與 403 混用

症狀：前端收到 403 就跳登入頁，使用者登入後還是 403，陷入循環；或沒登入卻拿到 403，客戶端不知道該先登入。

原因：兩種失敗的處理方式不同。401 的下一步是重新認證，403 的下一步是「換個有權限的帳號，或放棄」。

避免：沒有身分、身分無效回 401 並帶 `WWW-Authenticate`；有身分但不允許回 403。

### 坑 4：新 route 忘了掛認證

症狀：新增一個 endpoint 後，沒登入就能呼叫。

原因：如果認證是「每個 route 自己包 middleware」，新加的 route 只要忘記包，就是公開的。

避免：在最外層包整個 mux，預設全部都需要認證；真正公開的 route（如登入、跳轉）才明確排除。範例的 `Routes()` 是這個做法。

### 坑 5：只在前端隱藏按鈕（示意案例）

症狀：管理後台對一般使用者隱藏了「刪除」按鈕，但有人直接用 `curl` 呼叫 API，刪除成功。

原因：前端的顯示邏輯只是使用者體驗，不是防線。攻擊者不需要經過你的網頁。

避免：授權永遠在伺服器端強制執行；前端的隱藏只是方便，不能取代後端的檢查。

## 小練習

範例放在 `playground/backend/l1/1-4-auth-basics/authn-vs-authz/`，執行 `go test ./...` 驗證。想自己試，執行 `go run .`，再用 `curl` 呼叫（只在本機的 `127.0.0.1` 上）：

```bash
curl -i -X POST -H 'Authorization: Bearer alice-token' -d '{"url":"https://example.com"}' http://127.0.0.1:8080/links
curl -i -X DELETE -H 'Authorization: Bearer bob-token' http://127.0.0.1:8080/links/1
```

### 動手寫

目前只有 `view` 與 `delete`。請在 `policy.go` 新增 `ActionUpdate`（修改目標網址）：只有擁有者可以，管理員也不行。接著在 `links.go` 加上 `PUT /links/{code}`（body 為 `{"url": "..."}`），並在 `api_test.go` 的矩陣加上三列：擁有者成功（200）、其他使用者失敗（403）、管理員失敗（403）。

<details>
<summary>參考解答</summary>

```go
const ActionUpdate Action = "update"

// in Allowed:
case ActionUpdate:
	return l.OwnerID == u.ID
```

handler 一樣呼叫 `s.load(w, r, ActionUpdate)`，通過後再更新 URL。沒有改 `load`，是因為它對所有動作都適用，新動作只需要在政策裡補一條規則；若忘了補，`default` 會讓它被拒絕。

</details>

### 辨識

下面是 AI 產生的 handler。它能通過「已登入的 alice 建立並刪除自己的短網址」的測試，請做 code review，找出 3 個問題。

```go title="handlers.go"
func (s *Server) createLink(w http.ResponseWriter, r *http.Request) {
	var in struct {
		URL     string `json:"url"`
		OwnerID string `json:"owner_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	l := s.store.Create(in.OwnerID, in.URL)
	json.NewEncoder(w).Encode(l)
}

func (s *Server) getLink(w http.ResponseWriter, r *http.Request) {
	l, ok := s.store.Get(r.PathValue("code"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	json.NewEncoder(w).Encode(l)
}

func (s *Server) deleteLink(w http.ResponseWriter, r *http.Request) {
	user, _ := UserFrom(r.Context())
	l, ok := s.store.Get(r.PathValue("code"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	isAdmin := r.Header.Get("X-Role") == "admin"
	if l.OwnerID != user.ID && !isAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	s.store.Delete(l.Code)
	w.WriteHeader(http.StatusNoContent)
}
```

完整檔案在 `recognize/handlers.go`（AI 產生，不參與測試）。

<details>
<summary>參考解答</summary>

1. **`createLink` 從 body 讀 `OwnerID`**：任何已登入的人都能用別人的名義建立資料，也能把資料塞到 alice 名下。正確寫法：結構體只留 `URL`，擁有者用 `UserFrom(r.Context())` 取得的 `user.ID`。
2. **`getLink` 完全沒有授權檢查**：只要知道（或猜到）code，任何人都能讀到別人的短網址，這是 IDOR。正確寫法：取出資料後呼叫 `Allowed(user, ActionView, l)`，不符回 403。
3. **`deleteLink` 的 `isAdmin` 來自 `X-Role` header**：這個 header 由客戶端填寫，任何人加一行就變成管理員。正確寫法：角色從認證層放進 `context` 的 `user.Role` 取得。另外 `user, _ := UserFrom(...)` 忽略了「沒有身分」的情況，應該回 401。

這份程式在 alice 操作自己資料時完全正常，測試看不出問題；漏洞只在「別人」操作時出現。

</details>

## 快閃卡

<details>
<summary>認證與授權各自回答什麼問題？</summary>

認證回答「你是誰」，確認呼叫者的身分；授權回答「你可以做這件事嗎」，決定這個已知身分對某個資源能做什麼。授權要靠認證的結果才能運作。

</details>

<details>
<summary>為什麼要把認證與授權拆成兩層，而不是寫在一起？</summary>

認證在入口發生一次，授權則在每個動作、每筆資源都要判斷，變化的原因不同。拆開後，授權規則集中在一處，能單獨測試、審查，不會因為某個 handler 漏寫而變成漏洞。

</details>

<details>
<summary>請求沒有帶任何 Authorization header，應回 401 還是 403？</summary>

401。沒有身分代表尚未通過認證，且要附上 WWW-Authenticate header，告訴客戶端用什麼方式重新認證。

</details>

<details>
<summary>已登入的 bob 送出 DELETE /links/1，該短網址屬於 alice，bob 不是管理員。應回什麼狀態碼？</summary>

403。身分明確（bob），但政策不允許他刪除別人的短網址。回 401 會讓客戶端以為重新登入就有用。

</details>

<details>
<summary>登入後把網址 /orders/100 改成 /orders/101，就看到別人的訂單，這是哪種漏洞？根本原因是什麼？</summary>

這是 IDOR。只靠 ID 取資料，沒有檢查呼叫者與這筆資料的關係，也就是缺少授權檢查，不是認證的問題。

</details>

<details>
<summary>為什麼建立資料時，owner_id 不能從請求 body 讀取？</summary>

body 是客戶端自己填的，不可信。若讀它，任何人都能用別人的名義建立資料。擁有者應取自認證層驗證過的身分。

</details>

<details>
<summary>請求帶有 X-Role: admin，伺服器因此放行了管理操作。問題出在哪裡？</summary>

header 由客戶端控制，任何人都能偽造。角色必須來自伺服器驗證過的身分（例如 context 或資料庫），不能相信客戶端的聲明。

</details>

<details>
<summary>Allowed 函式的 default 為什麼回 false？</summary>

這是 deny-by-default：沒有列出的動作一律拒絕。新增動作卻忘記寫規則時，結果是被擋住，而不是被放行，錯誤比較容易被發現，也比較安全。

</details>

<details>
<summary>前端對一般使用者隱藏了「刪除」按鈕，後端沒有檢查。這樣安全嗎？</summary>

不安全。攻擊者可以不經過網頁，直接用 curl 呼叫 API。前端隱藏只是使用者體驗，授權一定要在伺服器端強制執行。

</details>

## 延伸閱讀

- [OWASP Top 10:2021 A01 Broken Access Control](https://owasp.org/Top10/A01_2021-Broken_Access_Control/)：權限控制失效為何是最常見的漏洞類別，與常見弱點對照
- [OWASP API Security Top 10 2023：API1 Broken Object Level Authorization](https://owasp.org/API-Security/editions/2023/en/0x11-t10/)：IDOR 在 API 中的樣貌與防禦
- [OWASP Authorization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html)：授權設計原則，包含 deny-by-default 與每個請求都檢查
- [RFC 9110：HTTP Semantics](https://www.rfc-editor.org/rfc/rfc9110.html)：401 與 403 的正式定義
- [MDN：403 Forbidden](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/403)：兩種狀態碼的差異與使用時機
