# 015 Function Values

把函式當成值：存進 map、當參數傳遞、從函式回傳。

## 需求

### Part 1：計算機

- 建立一個 map，key 是運算子字串（`+`、`-`、`*`、`/`），value 是對應的函式。
- `calculate(op, a, b)`：從 map 取出函式並呼叫。找不到運算子時印出 `unknown operator`。

### Part 2：高階函式

- `apply_to_all(nums, fn) -> list`：對每個元素套用 `fn`，回傳新清單，不修改原清單。
- 分別傳入「乘以 2」與「平方」兩個函式呼叫它。

### Part 3：Closure

- `make_counter() -> function`：每次呼叫回傳的函式，計數加 1 並回傳目前的值。
- 建立兩個 counter，確認它們各自計數。

## 範例

```
calculate("+", 6, 3) = 9
calculate("/", 6, 3) = 2
calculate("%", 6, 3) -> unknown operator

apply_to_all([1, 2, 3], double) = [2, 4, 6]
apply_to_all([1, 2, 3], square) = [1, 4, 9]

a() = 1
a() = 2
b() = 1
a() = 3
```

## 練習重點

- 函式型別
- 高階函式
- Closure 捕捉外部變數
