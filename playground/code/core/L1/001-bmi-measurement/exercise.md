# 001 BMI Measurement

輸入體重與身高，計算 BMI 並判斷體位分類。

## 需求

- 公式：`BMI = 體重(kg) / 身高(m)²`。
- 身高大於 3 時視為公分，先換算成公尺。
- 體重或身高小於等於 0 時回傳錯誤，印出 `Invalid input: <原因>`。
- 分類標準（WHO），門檻值宣告成具名常數：

  | BMI | 分類 |
  |---|---|
  | < 18.5 | Underweight |
  | 18.5–24.9 | Healthy |
  | 25.0–29.9 | Overweight |
  | ≥ 30.0 | Obesity |

- 本題把 WHO 的 Obesity Class 1–3 合併成一類 `Obesity`。
- 計算過程不四捨五入，只在輸出時取 2 位小數。

## 範例

| 體重 | 身高 | 輸出 |
|---|---|---|
| 62 | 1.7 | `Your BMI is 21.45, you are Healthy` |
| 64.3 | 173 | `Your BMI is 21.48, you are Healthy` |
| 72 | 1.7 | `Your BMI is 24.91, you are Healthy` |
| 90 | 1.7 | `Your BMI is 31.14, you are Obesity` |
| 0 | 1.7 | `Invalid input: weight must be > 0` |

## 練習重點

- 具名常數
- 函式拆分：單位換算、驗證、計算、分類
- 錯誤回傳
- 格式化輸出：小數位數
