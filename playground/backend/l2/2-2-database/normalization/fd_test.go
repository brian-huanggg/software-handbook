package normalization

import "testing"

// linksFlat 是「短網址服務 v2」第一版把擁有者資料塞進 links 的樣子。
var linksFlat = []Row{
	{"link_id": "1", "owner_id": "7", "owner_email": "ann@example.com", "plan": "pro", "daily_quota": "10000"},
	{"link_id": "2", "owner_id": "7", "owner_email": "ann@example.com", "plan": "pro", "daily_quota": "10000"},
	{"link_id": "3", "owner_id": "8", "owner_email": "bob@example.com", "plan": "free", "daily_quota": "100"},
}

func TestHoldsFD(t *testing.T) {
	tests := []struct {
		name string
		fd   FD
		want bool
	}{
		{"主鍵決定一切", FD{[]string{"link_id"}, []string{"owner_id", "owner_email"}}, true},
		{"擁有者決定 email（遞移相依）", FD{[]string{"owner_id"}, []string{"owner_email"}}, true},
		{"方案決定配額（遞移相依）", FD{[]string{"plan"}, []string{"daily_quota"}}, true},
		{"擁有者不決定連結", FD{[]string{"owner_id"}, []string{"link_id"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HoldsFD(linksFlat, tt.fd); got != tt.want {
				t.Fatalf("HoldsFD(%s) = %v, want %v", tt.fd, got, tt.want)
			}
		})
	}
}

// 更新異常：只改了其中一列的 email，同一個擁有者就有兩個 email，相依被破壞。
func TestUpdateAnomalyBreaksFD(t *testing.T) {
	rows := make([]Row, len(linksFlat))
	for i, r := range linksFlat {
		cp := Row{}
		for k, v := range r {
			cp[k] = v
		}
		rows[i] = cp
	}
	rows[0]["owner_email"] = "ann@new.example.com"

	if HoldsFD(rows, FD{[]string{"owner_id"}, []string{"owner_email"}}) {
		t.Fatal("同一個 owner_id 出現兩個 email，相依不該成立")
	}
}

func TestProjectSeparatesColumns(t *testing.T) {
	rows := []Row{
		{"a": "x", "b": "yz", "c": "1"},
		{"a": "xy", "b": "z", "c": "2"},
	}
	// 若 project 直接把值接在一起，兩列的 (a,b) 會被當成同一組，結果會是 false。
	if !HoldsFD(rows, FD{[]string{"a", "b"}, []string{"c"}}) {
		t.Fatal("(a,b) 的值不同，不該被視為同一組")
	}
}
