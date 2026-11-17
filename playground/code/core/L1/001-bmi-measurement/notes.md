# 001 BMI Measurement：跨語言對照

對照同一個 BMI 程式在五種語言中的寫法。欄位順序與資料夾一致：Python、TypeScript、Go、Java、C++。

## 對照表

| 概念 | Python | TypeScript | Go | Java | C++ |
|---|---|---|---|---|---|
| 執行方式 | `python3 main.py` | `node main.ts` | `go run .` | `java Main.java` | `clang++ -std=c++17 main.cpp -o main && ./main` |
| 入口 | `if __name__ == "__main__":` | 頂層呼叫 `main()` | `func main()` | `public static void main(String[] args)` | `int main()` |
| 常數 | 本練習直接寫數值 | `const CM_THRESHOLD = 3` | `const cmThreshold = 3` | `static final double CM_THRESHOLD = 3;` | `constexpr double cmThreshold = 3;` |
| 函式宣告 | `def to_meters(height: float) -> float:` | `function toMeters(height: number): number` | `func toMeters(h float64) float64` | `static double toMeters(double h)` | `double toMeters(double h)` |
| 型別 | 型別註記，執行時不檢查 | 靜態型別，編譯後移除 | 靜態，必須宣告 | 靜態，必須宣告 | 靜態，必須宣告 |
| 錯誤 | `raise ValueError(...)` | `throw new Error(...)` | `return errors.New(...)` | `throw new IllegalArgumentException(...)` | `throw std::invalid_argument(...)` |
| 錯誤處理 | `try` / `except` | `try` / `catch` | `if err != nil` | `try` / `catch` | `try` / `catch` |
| 分支 | `if` / `elif` / `else` | `if` / `else if` | `switch { case ...: }` | `if (...) return ...;` | `if (...) return ...;` |
| 小數格式化 | `f"{bmi:.2f}"` | `bmi.toFixed(2)` | `fmt.Printf("%.2f", bmi)` | `System.out.printf("%.2f", bmi)` | `std::fixed << std::setprecision(2)` |

## 補充說明

### 函式的位置

Python、TypeScript、Go、C++ 的函式可以直接寫在檔案頂層。Java 沒有獨立的函式，必須寫成 class 內的 `static` 方法。

### 錯誤處理

Go 沒有例外機制。`validate` 回傳 `error`，`getBmi` 回傳 `(float64, error)`，呼叫端用 `if err != nil` 檢查。其他四種語言用 `throw` 丟出例外，由 `try` / `catch` 接住。

### 編譯與直譯

| 語言 | 方式 |
|---|---|
| Python | 直譯，直接執行 |
| TypeScript | 先移除型別再執行（`node main.ts` 需要 Node.js 22.18 以上） |
| Go | 編譯後執行，`go run .` 會自動完成 |
| Java | `java Main.java` 會在記憶體中編譯後執行（JDK 11 以上） |
| C++ | 必須先編譯成執行檔，再執行 |

## 容易踩的坑

- Go：`go run .` 需要先有 `go.mod`，用 `go mod init <module>` 建立。
- Java：`public class` 的名稱必須等於檔名，所以 `Main.java` 內要寫 `public class Main`。
- C++：`main` 的回傳型別必須是 `int`，`void main()` 不合法。
- C++：函式必須先宣告才能使用，所以 `main` 放在最後。
- C++：只做檢查、不回傳值的函式要宣告為 `void`，宣告成 `double` 卻沒有 `return` 是錯誤。
- Java / Go：`%2f` 的 `2` 是最小寬度，不是小數位數，要寫 `%.2f`。
- Python：`f"{bmi: .2f}"` 的空格表示正數前面補一個空白，要寫 `f"{bmi:.2f}"`。
