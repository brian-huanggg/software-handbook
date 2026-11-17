# 022 Todo List

用動態清單實作待辦事項的增刪查改。

## 需求

- 每個項目有 `title` 與 `done` 兩個欄位。
- 支援以下操作，索引超出範圍時回傳錯誤，不讓程式崩潰：

  | 操作 | 說明 |
  |---|---|
  | `add(title)` | 加到尾端 |
  | `insert(index, title)` | 插入到指定位置 |
  | `remove(index)` | 刪除指定位置 |
  | `mark_done(index)` | 標記完成 |
  | `find(title) -> index` | 線性搜尋，找不到回傳 -1 |
  | `print_all()` | 印出索引、狀態、標題 |

## 操作順序

```
add "Buy milk"
add "Write report"
add "Call mom"
insert 1 "Pay bills"
mark_done 2
remove 0
remove 10
find "Call mom"
```

## 範例輸出

```
remove 10 -> error: index 10 out of range (size 3)
find "Call mom" -> 2
0 [ ] Pay bills
1 [x] Write report
2 [ ] Call mom
```

## 進階

- `remove_done()`：刪除所有已完成的項目。注意邊走訪邊刪除時的索引問題。

## 練習重點

- 動態清單的新增、插入、刪除
- 邊界檢查
- 清單元素是 struct 或物件
