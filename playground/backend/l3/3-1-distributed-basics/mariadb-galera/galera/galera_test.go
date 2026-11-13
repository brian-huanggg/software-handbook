package galera

import (
	"errors"
	"testing"
)

func commit(t *testing.T, c *Cluster, node int, key, val string) {
	t.Helper()
	tx := c.Begin(node)
	tx.Set(key, val)
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() on node %d: %v", node, err)
	}
}

func TestCommitSurvivesOriginCrash(t *testing.T) {
	c := New("db1", "db2", "db3")
	commit(t, c, 0, "abc", "https://example.com")
	c.Kill(0)

	// RPO = 0: the write-set already reached db2 before the client saw success.
	got, ok := c.ReadSynced(1, "abc")
	if !ok || got != "https://example.com" {
		t.Fatalf("ReadSynced(db2) = %q, %v; want the committed value", got, ok)
	}
}

func TestReadWithoutSyncWaitCanBeStale(t *testing.T) {
	c := New("db1", "db2", "db3")
	commit(t, c, 0, "abc", "https://example.com")

	if _, ok := c.Read(1, "abc"); ok {
		t.Fatal("db2 has not applied the write-set yet; a plain read should miss")
	}
	if _, ok := c.ReadSynced(1, "abc"); !ok {
		t.Fatal("a synced read should see the write")
	}
}

func TestConcurrentWritesToSameKeyConflict(t *testing.T) {
	c := New("db1", "db2", "db3")
	commit(t, c, 0, "abc", "v0")
	c.Node(1).Apply()

	a, b := c.Begin(0), c.Begin(1)
	a.Set("abc", "from-db1")
	b.Set("abc", "from-db2")

	if err := a.Commit(); err != nil {
		t.Fatalf("first commit: %v", err)
	}
	if err := b.Commit(); !errors.Is(err, ErrConflict) {
		t.Fatalf("second commit = %v, want ErrConflict", err)
	}
}

func TestDifferentKeysDoNotConflict(t *testing.T) {
	c := New("db1", "db2", "db3")
	a, b := c.Begin(0), c.Begin(1)
	a.Set("abc", "1")
	b.Set("xyz", "2")

	if err := a.Commit(); err != nil {
		t.Fatalf("first commit: %v", err)
	}
	if err := b.Commit(); err != nil {
		t.Fatalf("second commit: %v", err)
	}
}

func TestLosingQuorumRefusesWrites(t *testing.T) {
	c := New("db1", "db2", "db3")
	c.Kill(1)
	c.Kill(2)

	tx := c.Begin(0)
	tx.Set("abc", "1")
	if err := tx.Commit(); !errors.Is(err, ErrNoQuorum) {
		t.Fatalf("Commit() = %v, want ErrNoQuorum", err)
	}
}

func TestTwoNodeClusterHasNoToleranceForOneFailure(t *testing.T) {
	c := New("db1", "db2")
	c.Kill(1)

	tx := c.Begin(0)
	tx.Set("abc", "1")
	if err := tx.Commit(); !errors.Is(err, ErrNoQuorum) {
		t.Fatalf("Commit() = %v, want ErrNoQuorum", err)
	}
}

func TestRejoinCatchesUp(t *testing.T) {
	c := New("db1", "db2", "db3")
	c.Kill(2)
	commit(t, c, 0, "abc", "1")
	commit(t, c, 0, "xyz", "2")

	c.Rejoin(2)
	for _, k := range []string{"abc", "xyz"} {
		if _, ok := c.Read(2, k); !ok {
			t.Fatalf("db3 is missing %q after rejoin", k)
		}
	}
}
