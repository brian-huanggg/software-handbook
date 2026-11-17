// - 公式：`F = C × 9 / 5 + 32`、`K = C + 273.15`。
// - 公式中的數字（`9`、`5`、`32`、`273.15`）宣告成具名常數。
// - 從 -40°C 到 100°C，每 20 度印一列，小數取 1 位（克氏取 2 位）。

package main

import "fmt"

const (
	fahrenheitPerCelsius = 9.0 / 5.0 // 攝氏升 1 度，華氏升多少度
	fahrenheitOffset     = 32.0      // 0°C 對應 32°F
	kelvinOffset         = 273.15    // 0°C 對應 273.15 K
)

func celsius_to_fahrenheit(c float64) float64 {
	return c*fahrenheitPerCelsius + fahrenheitOffset
}

func celsius_to_kelvin(c float64) float64 {
	return c + kelvinOffset
}

func main() {
	fmt.Printf("%5s %8s %9s\n", "C", "F", "K")

	// 在迴圈裡用 int 跑 -40 到 100，
	for c := -40; c <= 100; c += 20 {
		cf := float64(c)

		// %xd = 整數 寬度x 向右靠齊
		// %a.b.f = 浮點數 寬度a, 小數點b
		fmt.Printf("%5d %8.1f %9.2f\n", c, celsius_to_fahrenheit(cf), celsius_to_kelvin(cf))
	}
}
