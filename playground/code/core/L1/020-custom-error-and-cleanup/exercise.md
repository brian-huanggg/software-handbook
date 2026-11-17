# 020 Custom Error and Cleanup

自訂帶有資料的錯誤型別，並確保每次操作結束都會執行清理。

## 需求

- 帳戶初始餘額為 0，餘額用整數。
- `deposit(amount)` 與 `withdraw(amount)`：
  - `amount <= 0` 時回傳 `ErrInvalidAmount`。
  - 提款超過餘額時回傳 `InsufficientFundsError`，帶有 `balance` 與 `requested` 兩個欄位。
- 呼叫端從 `InsufficientFundsError` 取出欄位，印出還差多少。
- 每個操作不論成功或失敗，結束時都要印出 `[op] done`。用 `defer`、`finally` 或語言中對應的機制實作，不要在每個 return 前各寫一次。

## 操作順序

`deposit 500`、`withdraw 200`、`withdraw 800`、`withdraw -5`

## 範例輸出

```
deposit 500 -> balance 500
[deposit] done
withdraw 200 -> balance 300
[withdraw] done
withdraw 800 -> insufficient funds: balance 300, requested 800, short 500
[withdraw] done
withdraw -5 -> invalid amount
[withdraw] done
```

## 練習重點

- 自訂錯誤型別
- 從錯誤取出結構化資料
- `defer` / `finally` 保證清理執行
