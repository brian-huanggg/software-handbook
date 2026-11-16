package shortener

import (
	"errors"
	"testing"
)

// 「先想想」的兩個案例：CreateNaive 的兩個判斷，真與假各走過一次，分支覆蓋 100%。
func TestCreateNaiveBranches(t *testing.T) {
	s := NewStore()

	code, err := s.CreateNaive("https://a.com", "")
	if err != nil || code != "1" {
		t.Fatalf("自動短碼 = %q, %v，期望 \"1\", nil", code, err)
	}

	_, err = s.CreateNaive("https://b.com", "1")
	if !errors.Is(err, ErrCodeTaken) {
		t.Fatalf("自訂短碼衝突 = %v，期望 ErrCodeTaken", err)
	}
}

// Create 的四條基本路徑：V(G) = 3 個判斷 + 1 = 4。
func TestCreateBasisPaths(t *testing.T) {
	t.Run("自訂短碼、沒被佔用", func(t *testing.T) {
		s := NewStore()
		code, err := s.Create("https://a.com", "go")
		if err != nil || code != "go" {
			t.Fatalf("= %q, %v，期望 \"go\", nil", code, err)
		}
	})
	t.Run("自訂短碼、已被佔用", func(t *testing.T) {
		s := NewStore()
		if _, err := s.Create("https://a.com", "go"); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Create("https://b.com", "go"); !errors.Is(err, ErrCodeTaken) {
			t.Fatalf("= %v，期望 ErrCodeTaken", err)
		}
	})
	t.Run("自動短碼、迴圈 0 次", func(t *testing.T) {
		s := NewStore()
		code, err := s.Create("https://a.com", "")
		if err != nil || code != "1" {
			t.Fatalf("= %q, %v，期望 \"1\", nil", code, err)
		}
	})
	t.Run("自動短碼、迴圈至少 1 次", func(t *testing.T) {
		s := NewStore()
		if _, err := s.Create("https://a.com", "1"); err != nil {
			t.Fatal(err)
		}
		code, err := s.Create("https://b.com", "")
		if err != nil || code != "2" {
			t.Fatalf("= %q, %v，期望跳過被佔走的 \"1\"，得到 \"2\", nil", code, err)
		}
	})
}

// 第四條路徑換到 CreateNaive 上：這就是「先想想」裡的事故。
func TestCreateNaiveAutoCollision(t *testing.T) {
	s := NewStore()
	if _, err := s.CreateNaive("https://a.com", "1"); err != nil {
		t.Fatal(err)
	}
	_, err := s.CreateNaive("https://b.com", "")
	if !errors.Is(err, ErrCodeTaken) {
		t.Fatalf("= %v；CreateNaive 的這條路徑本來就會回 ErrCodeTaken，這個測試記錄的是缺陷本身", err)
	}
}
