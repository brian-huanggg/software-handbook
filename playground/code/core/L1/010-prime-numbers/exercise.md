# 010 Prime Numbers

判斷質數，並用 `break`、`continue` 控制迴圈。

## 需求

- `is_prime(n) -> bool`：
  - `n < 2` 時提早回傳 `false`。
  - 只需要檢查到 `√n`；用 `i * i <= n` 判斷，不呼叫開根號函式。
- 印出 50 以內的所有質數。
- 找出第一個大於 1000 的質數：找到就用 `break` 離開迴圈；遇到偶數用 `continue` 跳過。

## 範例輸出

```
Primes <= 50: 2 3 5 7 11 13 17 19 23 29 31 37 41 43 47
Count: 15
First prime > 1000: 1009
```

## 練習重點

- 提早 return
- `break` 與 `continue`
- 迴圈終止條件
