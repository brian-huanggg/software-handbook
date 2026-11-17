# 018 Error Kinds

解析 `key=value` 設定行，讓呼叫端依錯誤種類做不同處理。

## 需求

- `parse_line(line) -> (key, value, error)`，可能的錯誤：

  | 錯誤 | 條件 |
  |---|---|
  | `ErrSkip` | 空行，或以 `#` 開頭的註解 |
  | `ErrMissingSeparator` | 沒有 `=` |
  | `ErrEmptyKey` | `=` 前面是空的 |

- key 與 value 去掉前後空白。
- 錯誤要能被程式判斷種類，不能只靠比對錯誤訊息字串。
- 呼叫端：`ErrSkip` 直接略過，不印任何東西；其他錯誤印出行號與原因後繼續。

## 輸入

```
host=localhost
port=8080

# comment
timeout
=30
name = demo
```

## 範例輸出

```
host: localhost
port: 8080
line 5: missing '='
line 6: empty key
name: demo
```

## 練習重點

- 預先定義的錯誤值或例外類別
- 依錯誤種類分支
- 不是所有錯誤都要印出來
