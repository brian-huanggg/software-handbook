---
title: 密碼雜湊：bcrypt、argon2、salt 與 pepper
description: 知道為什麼密碼不能用 SHA-256 存，並能用 argon2id 加 salt 與 pepper 寫出可驗證、可升級的密碼儲存。
sidebar:
  order: 2
---

## 先想想

短網址服務 v1 加上了帳號功能。你把 `users` 表的 `password_hash` 欄位用 `sha256(密碼)` 填好，心想：「SHA-256 是密碼學等級的雜湊，資料庫就算外洩也沒人能還原密碼。」

三個月後資料庫備份被誤放到公開的 bucket。資安同事說：「如果你的使用者有人用 `123456`，我五分鐘內就能在這份資料裡找到他。」

雜湊不是不可逆嗎？為什麼他還找得到？如果換成更長的 SHA-512，情況會改善嗎？

<details>
<summary>提示</summary>

攻擊者不用「還原」雜湊。想想他可以把哪些字串拿去算一次，再拿結果跟你的欄位比對。

</details>

## 是什麼

**密碼雜湊（password hashing）** 是用刻意設計成「慢」且「吃記憶體」的雜湊函式，把密碼轉成只能驗證、不能還原的字串後再存進資料庫。

它跟一般的雜湊函式（SHA-256、MD5）不同類：一般雜湊追求快，密碼雜湊追求貴。常見的選擇是 **argon2id** 與 **bcrypt**；**salt** 與 **pepper** 是它們搭配的兩種額外輸入。

## 為什麼存在

SHA-256 在一張現代 GPU 上每秒能算數十億次（量級估算，依硬體而異）。人類選的密碼很集中，攻擊者拿「常見密碼清單加變形規則」逐一計算雜湊，再比對你的欄位，就是所謂的暴力破解（brute force）。快，就等於幫攻擊者省錢。

更糟的是，沒有 salt 時，同一個密碼永遠得到同一個雜湊。攻擊者可以事先算好一大張對照表（**rainbow table**），拿到資料庫後只要查表；而且資料庫裡 5,000 個人用了 `123456`，破一個等於破 5,000 個。

2012 年 LinkedIn 外洩約 650 萬筆未加 salt 的 SHA-1 密碼雜湊，之後短時間內大量被破解（見延伸閱讀）。

密碼雜湊的做法是反過來：讓每次計算都慢到「使用者登入不受影響，但攻擊者每秒只能試幾十到幾百次」，並讓每個使用者的雜湊都不一樣。

## 怎麼運作

### 慢，而且可調：成本參數

**key stretching（金鑰延展）** 指把雜湊重複很多輪，讓一次計算變貴。bcrypt 用 **cost** 參數，每加 1，工作量加倍；argon2 用三個參數：記憶體用量 `m`、迭代次數 `t`、平行度 `p`。

argon2 比 bcrypt 多了「吃記憶體」這一項：GPU 與專用硬體擅長大量平行算小東西，但每個運算單元都要配 19 MiB 以上記憶體時，平行度就上不去。

三個常見選擇的定位：

- **argon2id**：2015 年密碼雜湊競賽的優勝者 argon2 的混合版本，規格是 RFC 9106。新專案的第一選擇。
- **bcrypt**：1999 年的設計，所有語言都有實作。限制是只吃前 72 bytes 的密碼。
- **scrypt、PBKDF2**：前者也吃記憶體；後者只吃時間，但是 FIPS 合規情境下常被要求。

### salt：讓每個雜湊都不一樣

**salt** 是每個使用者各自隨機產生的位元組（至少 16 bytes），在雜湊之前混進密碼。它不是祕密，直接跟雜湊存在一起。

它的作用只有一個：讓「同一個密碼」在不同使用者身上得到不同雜湊，使預先算好的表格失效，也讓攻擊者必須對每個使用者分別暴力破解。

這就是為什麼文章最前面那個 `sha256(密碼)` 不夠：沒有 salt、也不慢。

### pepper：存在資料庫以外的祕密

**pepper** 是全站共用的祕密值，放在環境變數或祕密管理服務裡，不跟雜湊存在同一個地方。

salt 防的是「預先計算」；pepper 防的是「只有資料庫外洩」。攻擊者只拿到資料表時，沒有 pepper 就連一次正確的猜測都驗證不了。

本文的做法是先用 `HMAC-SHA256(pepper, 密碼)` 處理密碼，再交給 argon2id：

```go title="argon2.go"
func applyPepper(password string, pepper []byte) []byte {
	if len(pepper) == 0 {
		return []byte(password)
	}
	mac := hmac.New(sha256.New, pepper)
	mac.Write([]byte(password))
	return mac.Sum(nil)
}
```

HMAC 輸出固定 32 bytes，順便避開 bcrypt 的 72 bytes 限制問題（見下方常見坑）。

### 一個字串裝下所有東西

雜湊結果通常存成一個自描述的字串。bcrypt 長這樣：`$2a$10$` 後面接 salt 與雜湊。argon2 採用 PHC 格式：

```text
$argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>
```

驗證時從字串裡讀出參數與 salt，用同樣的參數重算，再比對結果。比對要用固定時間（constant time）的函式，避免從比對耗時推測出雜湊的前幾個 byte：

```go title="argon2.go"
got := argon2.IDKey(applyPepper(password, pepper), salt, t, m, p, uint32(len(want)))
return subtle.ConstantTimeCompare(got, want) == 1, nil
```

參數寫在字串裡有個好處：之後想把 `t` 從 2 調到 3，舊使用者仍能用舊參數驗證，登入成功的當下再重算一次、存回新雜湊，就完成了無痛升級。

### 參數怎麼選

沒有放諸四海皆準的數字。做法是在你的正式環境硬體上量測，讓一次雜湊落在大約 100 到 500 毫秒之間；OWASP 的最低建議之一是 argon2id 的 `m=19456`（19 MiB）、`t=2`、`p=1`，範例採用這組。bcrypt 則建議 cost 至少 10。

回到一開始的問題：攻擊者不需要還原雜湊，他把常見密碼逐一算過 SHA-256，再跟你的欄位比對就行；SHA-256 快，沒有 salt，所以一次計算能同時比對全表。換成 SHA-512 一樣快，一樣沒有 salt，沒有改善。正確做法是 argon2id 加每人一個 salt（可再加 pepper），讓每次猜測又慢又得各自重算。

## 取捨與替代方案

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| argon2id | 抗 GPU、可調記憶體與時間，有 RFC | 吃記憶體，高併發登入要估算總量 | 新專案預設 |
| bcrypt | 成熟、各語言都有、參數只有 cost | 72 bytes 上限，不吃記憶體 | 既有系統、語言生態只有 bcrypt |
| scrypt | 吃記憶體，歷史長 | 三個參數較難調 | 已在使用時維持 |
| PBKDF2 | 標準、FIPS 認可 | 只吃時間，對 GPU 較弱 | 合規要求 |
| SHA-256 加 salt | 快，實作簡單 | 快就是缺點，不適合存密碼 | 不要用 |

登入流程的記憶體成本要實際估算：每次驗證用 19 MiB，同時 100 個登入就是約 1.9 GiB。需要時用限流（rate limiting）或 `golang.org/x/sync/semaphore` 限制同時進行的雜湊數量。

pepper 也有代價：pepper 一旦遺失，所有人的密碼都驗證不了；要輪替 pepper 還得替雜湊加上 pepper 版本號，並在使用者登入時逐步遷移。

### 什麼時候不該用

- **不需要自己存密碼時**：用 OAuth 2.0 或 OpenID Connect 讓 Google、GitHub 之類的服務做認證，你就沒有密碼可以外洩。
- **用 pepper 取代 salt 或慢雜湊時**：pepper 只是額外一層。沒有 salt 與成本參數，只加 pepper 仍是快雜湊。
- **存 API key、token 等高熵隨機值時**：這類值本身就無法被猜，用 SHA-256 即可，不需要 argon2 的成本，否則每次 API 請求都得多花幾百毫秒。

## 常見坑與故障案例

### 坑 1：用 SHA-256、MD5 存密碼

症狀：資料庫外洩後，短時間內大量帳號被還原成明文。

原因：通用雜湊為了快而設計，攻擊者每秒能試的次數太多。真實案例是 2012 年的 LinkedIn 事件：約 650 萬筆未加 salt 的 SHA-1 雜湊外洩後，絕大多數密碼很快被破解（見延伸閱讀）。

避免：改用 argon2id 或 bcrypt；既有使用者在下次登入成功時，用明文密碼重算並替換欄位。

### 坑 2：全站共用一個 salt

症狀：兩個使用者的 `password_hash` 一模一樣。

原因：把「salt」寫成常數，等於整個站共用一個 salt。攻擊者只要針對這個 salt 算一次表就能套用到所有人，等於沒有 salt。

避免：每次雜湊都用 `crypto/rand` 產生新的 salt。範例的測試 `TestSamePasswordDifferentHash` 就在檢查這件事。

### 坑 3：用 `==` 比對雜湊

症狀：功能正常，但在安全審查中被點名。

原因：`==` 在第一個不同的 byte 就停止，回應時間會洩漏比對進度。在網路上難以利用，但沒有理由留著。

避免：用 `subtle.ConstantTimeCompare`；bcrypt 的 `CompareHashAndPassword` 已內建。

### 坑 4：bcrypt 的 72 bytes 上限

症狀：使用者密碼是 100 個字元的密碼短語，但只打前 72 個字元也能登入。

原因：bcrypt 只使用前 72 bytes。舊版的函式庫會默默截斷；新版的 `golang.org/x/crypto/bcrypt` 會回傳 `ErrPasswordTooLong`。

避免：處理這個錯誤，不要吞掉；或像上面那樣先做 HMAC 預處理。多位元組的中文密碼每個字佔 3 bytes，更容易超過。

### 坑 5：pepper 寫死在程式碼或放進同一個資料庫

症狀：程式碼倉庫或資料庫備份一外洩，pepper 也一起洩漏。

原因：pepper 的價值在於「跟雜湊分開存放」；放在同一處，就沒有任何額外保護。

避免：從環境變數或祕密管理服務讀取，並跟資料庫備份用不同的存取權限（祕密管理會在 2.6 介紹）。

## 小練習

範例放在 `playground/backend/l1/1-4-auth-basics/password-hash/`，執行 `go test ./...` 可以驗證。

### 動手寫

在 `argon2.go` 旁新增 `NeedsRehash(encoded string) bool`：讀出字串裡的 `m`、`t`、`p`，只要其中任何一個與目前常數 `memoryKiB`、`timeCost`、`threads` 不同就回傳 `true`。接著寫一個測試：手動組出 `m=8192,t=1,p=1` 的雜湊字串，確認 `NeedsRehash` 回傳 `true`，而新產生的雜湊回傳 `false`。

<details>
<summary>參考解答</summary>

```go
func NeedsRehash(encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return true
	}
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return true
	}
	return m != memoryKiB || t != timeCost || p != threads
}
```

登入成功後呼叫它：回傳 `true` 就用明文密碼重新 `HashArgon2id` 並更新資料庫，使用者完全無感。格式有問題時回傳 `true` 也合理，因為這代表該重算。

</details>

### 辨識

下面是 AI 產生的帳號密碼模組。它能通過「同密碼驗證成功、錯密碼驗證失敗」的測試，請做 code review，找出 3 個問題。

```go title="accounts.go"
// Salt keeps rainbow tables from working against our stored passwords.
const Salt = "sho.rt-2024-s3cr3t"

// HashPassword returns the hex digest stored in users.password_hash.
func HashPassword(password string) string {
	sum := sha256.Sum256([]byte(Salt + password))
	return hex.EncodeToString(sum[:])
}

// CheckPassword reports whether password matches the stored hash.
func CheckPassword(password, stored string) bool {
	return HashPassword(password) == stored
}
```

完整檔案在 `recognize/accounts.go`（AI 產生，不參與測試）。

<details>
<summary>參考解答</summary>

1. **`Salt` 是全站共用的常數**：每個使用者的相同密碼會得到相同雜湊，攻擊者針對這一個 salt 預算一張表就能套用全站；而且它寫在原始碼裡，倉庫外洩就等於公開。正確做法：每個使用者每次雜湊都用 `crypto/rand` 產生 16 bytes 以上的隨機 salt，並跟雜湊存在一起。
2. **用 `sha256.Sum256` 當密碼雜湊**：一次運算只要微秒級，攻擊者每秒能試數十億次。註解寫的「防止 rainbow table」只防了預先計算，沒有防暴力破解。正確做法：換成 argon2id 或 bcrypt，並設定成本參數。
3. **`HashPassword(password) == stored` 用 `==` 比對**：在第一個不同的 byte 就回傳，耗時會洩漏比對進度。正確做法：`subtle.ConstantTimeCompare`，或直接用 bcrypt 的 `CompareHashAndPassword`。

這份程式的每個函式都按照註解做事，所以測試會過，這正是它危險的地方：它「能運作」，但不安全。

</details>

## 快閃卡

<details>
<summary>為什麼用 SHA-256 存密碼不夠安全？</summary>

SHA-256 為了快而設計，攻擊者每秒能試數十億個候選密碼，再與外洩的欄位比對。密碼雜湊刻意讓每次計算變慢且吃記憶體，把這個速度壓到每秒幾十到幾百次。

</details>

<details>
<summary>salt 的作用是什麼？它需要保密嗎？</summary>

salt 是每人各自隨機的值，讓相同密碼產生不同雜湊，使預算好的表格失效。它不需要保密，直接跟雜湊存在同一個欄位。

</details>

<details>
<summary>兩個使用者的 password_hash 欄位完全相同，最可能的原因是什麼？</summary>

最可能是沒有 salt，或全站共用同一個 salt，讓相同密碼得到相同雜湊。每次雜湊都該用 crypto/rand 產生新的 salt。

</details>

<details>
<summary>pepper 與 salt 的差別是什麼？</summary>

salt 每人一個、跟雜湊存在一起，防預先計算。pepper 全站共用、存在資料庫以外，防「只有資料庫外洩」的情境，攻擊者沒有它就無法驗證任何猜測。

</details>

<details>
<summary>bcrypt 的 cost 從 10 調到 12，驗證一次的工作量大約變成幾倍？</summary>

4 倍。cost 每加 1，工作量加倍，加 2 就是 2 的 2 次方。

</details>

<details>
<summary>用 bcrypt 存一個 100 bytes 的密碼，新版 Go 的 x/crypto 會發生什麼事？</summary>

GenerateFromPassword 會回傳 bcrypt.ErrPasswordTooLong，因為 bcrypt 只使用前 72 bytes。解法是處理這個錯誤，或先用 HMAC 預處理成固定長度。

</details>

<details>
<summary>雜湊字串裡為什麼要寫入 m、t、p 等參數？</summary>

驗證時才能用原本的參數重算，之後調高參數也不會讓舊雜湊失效。登入成功後再用新參數重算並存回，就能無痛升級。

</details>

<details>
<summary>把 API key 存進資料庫前，該用 argon2id 還是 SHA-256？為什麼？</summary>

用 SHA-256 即可。API key 是高熵的隨機值，無法被猜測，不需要靠慢雜湊提高成本；用 argon2id 只會讓每次 API 請求多花幾百毫秒。

</details>

<details>
<summary>驗證時寫 `HashPassword(pw) == stored`，有什麼問題？</summary>

字串比對在第一個不同的 byte 就停止，耗時會洩漏比對進度。應使用 subtle.ConstantTimeCompare 做固定時間比對。

</details>

## 延伸閱讀

- [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)：argon2id、bcrypt 的建議參數與 pepper 用法
- [RFC 9106：Argon2](https://www.rfc-editor.org/rfc/rfc9106)：argon2 的規格與參數選擇指引
- [A Future-Adaptable Password Scheme](https://www.usenix.org/legacy/events/usenix99/provos/provos.pdf)：bcrypt 原始論文
- [golang.org/x/crypto/argon2](https://pkg.go.dev/golang.org/x/crypto/argon2)：Go 的 argon2 套件文件
- [golang.org/x/crypto/bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt)：Go 的 bcrypt 套件文件
- [2012 LinkedIn hack](https://en.wikipedia.org/wiki/2012_LinkedIn_hack)：未加 salt 的 SHA-1 雜湊外洩事件
