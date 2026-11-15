package consistency

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var errBoom = errors.New("boom")

func setup(t *testing.T) (*Service, *DirStore) {
	t.Helper()
	store := &DirStore{Root: t.TempDir()}
	return &Service{Store: store, DB: NewDB()}, store
}

// assertNoDangling is the invariant the whole design protects: every key a
// record points at exists in the object store.
func assertNoDangling(t *testing.T, s *Service) {
	t.Helper()
	for _, key := range s.DB.Avatars() {
		if !s.Store.Exists(key) {
			t.Fatalf("dangling record: %s is referenced but missing", key)
		}
	}
}

func upload(t *testing.T, s *Service, user, body string) string {
	t.Helper()
	key, err := s.ReplaceAvatar(user, strings.NewReader(body))
	if err != nil {
		t.Fatalf("ReplaceAvatar: %v", err)
	}
	return key
}

func TestReplaceQueuesOldKeyThenSweepDeletesIt(t *testing.T) {
	s, store := setup(t)
	first := upload(t, s, "u1", "v1")
	second := upload(t, s, "u1", "v2")

	if first == second {
		t.Fatal("keys must be unique per upload")
	}
	if got := s.DB.PendingDeletions(); len(got) != 1 || got[0] != first {
		t.Fatalf("pending = %v, want [%s]", got, first)
	}
	// Until the sweeper runs, the old object still exists: harmless.
	if !store.Exists(first) {
		t.Fatal("old object should survive until the sweep")
	}
	assertNoDangling(t, s)

	if err := s.Sweep(); err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if store.Exists(first) || len(s.DB.PendingDeletions()) != 0 {
		t.Fatal("sweep should remove the object and its row")
	}
	assertNoDangling(t, s)
}

func TestCrashAfterPutBeforeCommit(t *testing.T) {
	s, store := setup(t)
	old := upload(t, s, "u1", "v1")

	// Commit fails and cleanup fails too: the process effectively died.
	s.DB.FailCommit = func() error { return errBoom }
	store.FailDelete = func(string) error { return errBoom }

	if _, err := s.ReplaceAvatar("u1", strings.NewReader("v2")); err == nil {
		t.Fatal("want error")
	}

	if got := s.DB.Avatar("u1"); got != old {
		t.Fatalf("pointer moved to %s; want it to stay %s", got, old)
	}
	assertNoDangling(t, s)
	if len(s.DB.PendingDeletions()) != 0 {
		t.Fatal("rolled-back transaction must not queue deletions")
	}

	// The new object is an orphan. Within the grace period it is left alone;
	// after the grace period the scan reports exactly it.
	now := time.Now()
	if got, _ := s.Orphans(now, time.Hour); len(got) != 0 {
		t.Fatalf("fresh object reported as orphan: %v", got)
	}
	got, err := s.Orphans(now.Add(2*time.Hour), time.Hour)
	if err != nil || len(got) != 1 {
		t.Fatalf("orphans = %v, %v; want exactly one", got, err)
	}
}

func TestCommitFailureCleansUpNewObject(t *testing.T) {
	s, store := setup(t)
	s.DB.FailCommit = func() error { return errBoom }

	if _, err := s.ReplaceAvatar("u1", strings.NewReader("v1")); err == nil {
		t.Fatal("want error")
	}
	objs, _ := store.List()
	if len(objs) != 0 {
		t.Fatalf("leftover objects: %v", objs)
	}
}

func TestPutFailureTouchesNothing(t *testing.T) {
	s, store := setup(t)
	old := upload(t, s, "u1", "v1")
	store.FailPut = func(string) error { return errBoom }

	if _, err := s.ReplaceAvatar("u1", strings.NewReader("v2")); err == nil {
		t.Fatal("want error")
	}
	if s.DB.Avatar("u1") != old || len(s.DB.PendingDeletions()) != 0 {
		t.Fatal("database must not change when the upload failed")
	}
	assertNoDangling(t, s)
}

func TestSweepCrashMidDeleteThenRerun(t *testing.T) {
	s, store := setup(t)
	a := upload(t, s, "u1", "v1")
	upload(t, s, "u1", "v2")
	upload(t, s, "u1", "v3")

	// First run: deleting the first queued key fails.
	store.FailDelete = func(key string) error {
		if key == a {
			return errBoom
		}
		return nil
	}
	if err := s.Sweep(); err == nil {
		t.Fatal("want error from first sweep")
	}
	if got := s.DB.PendingDeletions(); len(got) != 1 || got[0] != a {
		t.Fatalf("pending after failed sweep = %v, want [%s]", got, a)
	}
	assertNoDangling(t, s)

	// Rerun with the fault gone: no residue.
	store.FailDelete = nil
	if err := s.Sweep(); err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if len(s.DB.PendingDeletions()) != 0 || store.Exists(a) {
		t.Fatal("rerun should leave nothing behind")
	}
}

func TestSweepIsIdempotentWhenObjectAlreadyGone(t *testing.T) {
	s, store := setup(t)
	first := upload(t, s, "u1", "v1")
	upload(t, s, "u1", "v2")

	// A previous sweep deleted the object but died before removing the row.
	if err := store.Delete(first); err != nil {
		t.Fatal(err)
	}
	if err := s.Sweep(); err != nil {
		t.Fatalf("Sweep must treat a missing object as done: %v", err)
	}
	if len(s.DB.PendingDeletions()) != 0 {
		t.Fatal("row should be removed")
	}
}

func TestPutNeverOverwrites(t *testing.T) {
	s, store := setup(t)
	if err := store.Put("k", strings.NewReader("a")); err != nil {
		t.Fatal(err)
	}
	if err := store.Put("k", strings.NewReader("b")); err == nil {
		t.Fatal("second Put to the same key must fail")
	}
	_ = s
}
