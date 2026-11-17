# 007 FizzBuzz Extended

FizzBuzz 加上第三條規則。

## 需求

- 從 1 印到 35，每個數字一行。
- 3 的倍數印 `Fizz`，5 的倍數印 `Buzz`，7 的倍數印 `Bazz`。
- 同時符合多條規則時，依 Fizz、Buzz、Bazz 的順序串接，例如 15 印 `FizzBuzz`、21 印 `FizzBazz`、35 印 `BuzzBazz`。
- 都不符合時印出數字本身。

## 範例輸出（1–15）

```
1
2
Fizz
4
Buzz
Fizz
Bazz
8
Fizz
Buzz
11
Fizz
13
Bazz
FizzBuzz
```

## 進階

- 把規則改成資料（除數與字詞的清單），新增第四條規則時不需要新增 `if`。

## 練習重點

- `for` 迴圈
- `%` 與條件組合
- 避免為每種組合各寫一個分支
