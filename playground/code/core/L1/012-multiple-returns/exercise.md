# 012 Multiple Returns

讓一個函式一次回傳多個值。

## 需求

### Part 1：統計

- `stats(nums) -> (min, max, avg, ok)`：一次走訪就算出最小值、最大值、平均。
- 清單為空時 `ok` 為 `false`，其他值無意義。
- 你的語言沒有多回傳值時，改用 tuple、record、struct 或物件，並在 `notes.md` 記錄。

### Part 2：商與餘數

- `div_mod(a, b) -> (quotient, remainder)`。
- 用你的語言內建的 `/` 與 `%` 實作，不要自己修正正負號。

## 範例

| 輸入 | 輸出 |
|---|---|
| `stats([3, 9, -2, 7, 5])` | `min=-2 max=9 avg=4.40` |
| `stats([])` | `empty list` |
| `div_mod(7, 2)` | `3, 1` |
| `div_mod(-7, 2)` | 依語言而定 |

## 觀察

- `div_mod(-7, 2)` 在不同語言可能是 `-3, -1` 或 `-4, 1`。你的語言是哪一種？

## 練習重點

- 多回傳值、tuple 或 struct
- 用旗標表示「沒有結果」
- 整數除法的取整方向
