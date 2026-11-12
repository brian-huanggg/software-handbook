package pkchoice

import (
	"bytes"
	"testing"
	"time"
)

func TestV7Layout(t *testing.T) {
	now := time.UnixMilli(1_700_000_000_123)
	u, err := NewV7(now)
	if err != nil {
		t.Fatal(err)
	}
	if got := u[6] >> 4; got != 7 {
		t.Errorf("version = %d, want 7", got)
	}
	if got := u[8] >> 6; got != 0b10 {
		t.Errorf("variant bits = %b, want 10", got)
	}
	if !u.Time().Equal(now) {
		t.Errorf("Time() = %v, want %v", u.Time(), now)
	}
}

func TestV7SortsByTime(t *testing.T) {
	base := time.UnixMilli(1_700_000_000_000)
	var prev UUID
	for i := range 1000 {
		u, err := NewV7(base.Add(time.Duration(i) * time.Millisecond))
		if err != nil {
			t.Fatal(err)
		}
		if i > 0 && bytes.Compare(prev[:], u[:]) >= 0 {
			t.Fatalf("id %d (%s) does not sort after previous (%s)", i, u, prev)
		}
		prev = u
	}
}

func TestSnowflakeIncreasesAndIsUnique(t *testing.T) {
	// 凍結時間，逼出「同一毫秒用序號區分」的路徑。
	frozen := Epoch.Add(time.Hour)
	sf, err := NewSnowflake(7, func() time.Time { return frozen })
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int64]bool{}
	var prev int64
	for i := range maxSequence + 1 {
		id, err := sf.Next()
		if err != nil {
			t.Fatal(err)
		}
		if seen[id] || id <= prev {
			t.Fatalf("id %d at i=%d is duplicated or not increasing", id, i)
		}
		seen[id], prev = true, id
	}
}

func TestSnowflakeClockBackwards(t *testing.T) {
	now := Epoch.Add(time.Hour)
	sf, _ := NewSnowflake(1, func() time.Time { return now })
	if _, err := sf.Next(); err != nil {
		t.Fatal(err)
	}
	now = now.Add(-time.Second)
	if _, err := sf.Next(); err != ErrClockBackwards {
		t.Fatalf("err = %v, want ErrClockBackwards", err)
	}
}

func TestSnowflakeWorkerIDsDoNotCollide(t *testing.T) {
	frozen := Epoch.Add(time.Hour)
	clock := func() time.Time { return frozen }
	a, _ := NewSnowflake(1, clock)
	b, _ := NewSnowflake(2, clock)
	ida, _ := a.Next()
	idb, _ := b.Next()
	if ida == idb {
		t.Fatal("different workers produced the same id in the same millisecond")
	}
}
