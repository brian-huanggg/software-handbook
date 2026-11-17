# 013 Value vs Reference

觀察函式參數是傳值還是傳參考。

## 需求

寫以下四個函式，每個都在 `main` 呼叫，並在呼叫前後印出變數：

| 函式 | 做的事 |
|---|---|
| `double_value(n)` | 把整數參數乘以 2 |
| `double_all(list)` | 把清單每個元素乘以 2 |
| `append_item(list, x)` | 在清單尾端加入一個元素 |
| `rename(person, name)` | 修改 person 物件（或 struct）的 `name` 欄位 |

**先預測，再執行。** 執行前把預測填進下表，執行後填入實際結果：

| 函式 | 預測：呼叫端會看到改變嗎？ | 實際 |
|---|---|---|
| `double_value` | | |
| `double_all` | | |
| `append_item` | | |
| `rename` | | |

### 修正

讓四個函式的修改都能被呼叫端看到。依你的語言，選擇回傳新值、傳指標或傳參考。

## 練習重點

- 傳值與傳參考
- 可變（mutable）與不可變（immutable）
- 函式的副作用
