# 025 Inventory

L1 綜合題：用 A–E 章的所有概念實作庫存管理。

## 需求

- 庫存用 map 儲存：key 是商品名稱，value 是包含 `qty` 與 `price`（整數，新台幣元）的 struct 或物件。
- 指令寫死成一個清單，每筆指令是一個 struct 或物件（不要解析字串，字串處理是 L2 的內容）。
- 支援的指令：

  | 指令 | 說明 |
  |---|---|
  | `add(name, qty, price)` | 新商品就建立；已存在就增加數量，價格不變 |
  | `remove(name, qty)` | 商品不存在或數量不足時回傳錯誤 |
  | `report()` | 依名稱排序，印出每項小計與總價值；數量低於 15 標示 `LOW` |

- `qty <= 0` 的指令回傳錯誤。
- 錯誤不中斷後續指令。

## 指令清單

```
add apple 10 15
add banana 20 8
remove apple 3
remove banana 25
remove cherry 1
add apple 5 15
add grape 0 30
report
```

## 範例輸出

```
remove banana 25 -> error: insufficient stock for banana: have 20, want 25
remove cherry 1 -> error: unknown item: cherry
add grape 0 -> error: qty must be > 0, got 0

Item      Qty  Price  Subtotal
apple      12     15       180  LOW
banana     20      8       160
------------------------------
Total                      340
```

## 練習重點

- 型別與常數（A）
- 控制流與提早 return（B）
- 函式拆分（C）
- 自訂錯誤與錯誤處理（D）
- map 與 list（E）
