# 019 Error Wrapping

錯誤經過多層函式時，每層補上上下文，呼叫端仍能判斷根本原因。

## 需求

- 資料寫死在 map：`{1: "Alice", 2: "Bob"}`。
- 三層呼叫：`main` → `load_user(id)` → `find_record(id)`。
- `find_record`：
  - `id <= 0` 時回傳 `ErrInvalidID`。
  - 找不到時回傳 `ErrNotFound`。
- `load_user` 不自己處理錯誤，只包上一層上下文再往上傳，例如 `load user 3: find record 3: record not found`。
- `main` 檢查根本原因：
  - `ErrNotFound`：印出 `user not found`。
  - `ErrInvalidID`：印出 `invalid input`。
  - 兩者都要附上完整的錯誤鏈。

## 範例輸出

```
id 1 -> Alice
id 3 -> user not found (load user 3: find record 3: record not found)
id -1 -> invalid input (load user -1: find record -1: invalid id)
```

## 練習重點

- 錯誤包裝（wrapping）與錯誤鏈
- 包裝後仍能判斷根本原因
- 中間層只補上下文，不吞掉錯誤
