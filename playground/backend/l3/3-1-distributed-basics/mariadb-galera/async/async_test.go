package async

import "testing"

func TestFailOverLosesUnshippedWrites(t *testing.T) {
	c := New()
	for _, k := range []string{"a", "b", "c", "d", "e"} {
		c.Write(k, "u")
	}
	c.Ship(2)

	if got := c.Lag(); got != 3 {
		t.Fatalf("Lag() = %d, want 3", got)
	}
	if _, ok := c.ReadReplica("e"); ok {
		t.Fatal("replica should not have write e yet")
	}

	if lost := c.FailOver(); lost != 3 {
		t.Fatalf("FailOver() lost %d writes, want 3", lost)
	}
	if _, ok := c.ReadPrimary("e"); ok {
		t.Fatal("write e was acknowledged but should be gone after failover")
	}
	if _, ok := c.ReadPrimary("b"); !ok {
		t.Fatal("write b was shipped and should survive failover")
	}
}
