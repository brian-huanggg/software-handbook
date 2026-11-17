# 011 Discount Calculator

把計價流程拆成多個小函式，再由 `main` 組合。

## 需求

- 金額一律為整數（新台幣元）。
- 每個函式只做一件事：
  - `apply_member_discount(price, is_member) -> int`：會員打九折，無條件捨去。
  - `apply_coupon(price, coupon) -> int`：扣掉折價券金額，結果不得小於 0。
  - `shipping_fee(price) -> int`：滿 500 免運，否則運費 60。
  - `final_price(price, is_member, coupon) -> int`：依序套用以上三步。
- 折扣率、免運門檻、運費宣告成常數。

## 範例

| 原價 | 會員 | 折價券 | 結果 |
|---|---|---|---|
| 1000 | 是 | 50 | 850 |
| 300 | 否 | 0 | 360 |
| 550 | 是 | 0 | 555 |
| 40 | 否 | 50 | 60 |

## 練習重點

- 函式參數與回傳值
- 單一職責
- 純函式：相同輸入永遠得到相同輸出
