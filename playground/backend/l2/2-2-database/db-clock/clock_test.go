package dbclock

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"testing"
	"time"
)

func testDB(t *testing.T) (context.Context, *sql.DB) {
	t.Helper()
	db, err := Open(DefaultDSN)
	if err != nil {
		t.Skipf("資料庫未啟動（先執行 make up）：%v", err)
	}
	t.Cleanup(func() { db.Close() })
	ctx := context.Background()
	if err := Reset(ctx, db); err != nil {
		t.Fatal(err)
	}
	return ctx, db
}

var nodes = []Node{
	{Name: "A", Skew: 0},
	{Name: "B", Skew: 90 * time.Second},
	{Name: "C", Skew: -90 * time.Second},
}

// 三台節點對同一條 60 秒的短網址，應用程式時間會給出不同答案。
func TestAppClockDisagrees(t *testing.T) {
	ctx, db := testDB(t)
	// 由時鐘正常的 A 建立，ttl 60 秒，此刻真實時間內都還有效。
	if err := CreateApp(ctx, db, nodes[0], "x", 60*time.Second); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, n := range nodes {
		ok, err := IsActiveApp(ctx, db, n, "x")
		if err != nil {
			t.Fatal(err)
		}
		got[n.Name] = ok
	}
	if !got["A"] || got["B"] {
		t.Fatalf("預期 A 有效、B（快 90 秒）判定過期，實際 %v", got)
	}
}

// 用資料庫時間，節點的時鐘怎麼漂都不影響結果。
func TestDBClockAgrees(t *testing.T) {
	ctx, db := testDB(t)
	if err := CreateDB(ctx, db, "x", 60*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := CreateDB(ctx, db, "gone", -time.Second); err != nil {
		t.Fatal(err)
	}
	// IsActiveDB 沒有任何節點參數：結果只由資料庫決定。
	for code, want := range map[string]bool{"x": true, "gone": false} {
		got, err := IsActiveDB(ctx, db, code)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s：有效=%v，預期 %v", code, got, want)
		}
	}
	if _, err := IsActiveDB(ctx, db, "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("不存在的短碼應回 ErrNotFound，實際 %v", err)
	}
}

func TestClocksInTransaction(t *testing.T) {
	ctx, db := testDB(t)
	cs, err := ReadClocks(ctx, db, 3, 150*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cs[1:] {
		if !c.TxStart.Equal(cs[0].TxStart) {
			t.Errorf("now() 在交易內應該不變：%v vs %v", c.TxStart, cs[0].TxStart)
		}
	}
	if d := cs[2].Wall.Sub(cs[0].Wall); d < 250*time.Millisecond {
		t.Errorf("clock_timestamp() 應隨真實時間前進，只前進了 %v", d)
	}
	if d := cs[2].Statement.Sub(cs[0].Statement); d < 250*time.Millisecond {
		t.Errorf("statement_timestamp() 每條語句都會更新，只前進了 %v", d)
	}
}

func TestTimestampWithoutTZ(t *testing.T) {
	ctx, db := testDB(t)
	// 同一個瞬間寫入；讀的 session 換了時區，timestamp 欄位就差了整整 8 小時。
	plainSame, tzSame, err := StoredAs(ctx, db, "UTC", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(plainSame-tzSame) > 1 {
		t.Fatalf("同時區下兩欄應一致，差 %v 秒", plainSame-tzSame)
	}
	plain, tz, err := StoredAs(ctx, db, "UTC", "Asia/Taipei")
	if err != nil {
		t.Fatal(err)
	}
	if diff := math.Abs(plain - tz); math.Abs(diff-8*3600) > 1 {
		t.Errorf("timestamp 欄位應差 8 小時，實際差 %v 秒", diff)
	}
	if now := float64(time.Now().Unix()); math.Abs(tz-now) > 5 {
		t.Errorf("timestamptz 應等於真實時間，差 %v 秒", tz-now)
	}
}
