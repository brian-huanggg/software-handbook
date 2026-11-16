//go:build exercise

package shortener

import "testing"

// TestMCDC 檢查 targetCases 裡，每個原子條件都有一組獨立影響對。
func TestMCDC(t *testing.T) {
	for i, name := range conditionNames {
		a, b, ok := independencePair(targetCases, i)
		if !ok {
			t.Errorf("條件 %s 沒有獨立影響對：找不到只差這個條件、結果又不同的兩個案例", name)
			continue
		}
		t.Logf("條件 %s：%q 與 %q", name, a.name, b.name)
	}
}
