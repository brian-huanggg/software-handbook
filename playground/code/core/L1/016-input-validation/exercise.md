# 016 Input Validation

驗證註冊資料，回傳清楚的錯誤訊息。

## 需求

- `validate_user(username, age, password)`，規則：
  - `username` 長度 3–20。
  - `age` 在 0–150 之間。
  - `password` 長度至少 8。
- 遇到第一個不符合的規則就回傳錯誤，訊息要說明哪個欄位、為什麼不合法。
- 全部通過時回傳「沒有錯誤」（依語言是 `nil`、`null`、不丟例外等）。
- `main` 依序驗證範例中的每一筆資料，錯誤不會中斷後續驗證。

## 範例

| username | age | password | 輸出 |
|---|---|---|---|
| `alice` | 30 | `secret123` | `ok` |
| `al` | 30 | `secret123` | `username must be 3-20 characters, got 2` |
| `bob` | 200 | `secret123` | `age must be between 0 and 150, got 200` |
| `carol` | 25 | `short` | `password must be at least 8 characters` |

## 進階

- 改成一次回傳所有錯誤，而不是只回傳第一個。

## 練習重點

- 回傳錯誤或丟出例外
- 錯誤訊息包含足夠的上下文
- 密碼不出現在錯誤訊息中
