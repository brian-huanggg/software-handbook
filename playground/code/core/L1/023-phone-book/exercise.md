# 023 Phone Book

用 map 實作電話簿的增刪查改。

## 需求

- key 是姓名，value 是電話號碼字串。
- 支援以下操作：

  | 操作 | 說明 |
  |---|---|
  | `add(name, phone)` | 已存在時回傳錯誤，不覆蓋 |
  | `lookup(name)` | 區分「不存在」與「存在但電話是空字串」 |
  | `update(name, phone)` | 不存在時回傳錯誤 |
  | `delete(name)` | 不存在時回傳錯誤 |
  | `list_all()` | 依姓名排序後印出 |

## 操作順序

```
add "Alice" "0912-345-678"
add "Bob" ""
add "Alice" "0900-000-000"
lookup "Bob"
lookup "Carol"
update "Bob" "0922-111-222"
update "Carol" "0933-333-333"
delete "Alice"
add "Dave" "0955-555-555"
list_all
```

## 範例輸出

```
add Alice -> error: Alice already exists
lookup Bob -> "" (exists)
lookup Carol -> not found
update Carol -> error: Carol not found
Bob: 0922-111-222
Dave: 0955-555-555
```

## 觀察

- 不排序直接走訪 map，多跑幾次輸出順序一樣嗎？你的語言是否保證順序？

## 練習重點

- map 的新增、查詢、修改、刪除
- 判斷 key 是否存在
- map 的走訪順序
