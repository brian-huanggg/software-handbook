package password

// NewLimited wraps h so that at most n Hash or Verify calls run at once;
// the rest wait for a slot. Each Argon2id call allocates memoryKiB of RAM,
// so without a cap a burst of logins can exhaust memory. Pick n around the
// number of CPUs: more only adds memory pressure, not throughput.
func NewLimited(h Hasher, n int) Hasher {
	if n <= 0 {
		panic("password: NewLimited needs n > 0")
	}
	return &limited{h: h, slots: make(chan struct{}, n)}
}

type limited struct {
	h     Hasher
	slots chan struct{}
}

func (l *limited) Hash(password string) (string, error) {
	l.slots <- struct{}{}
	defer func() { <-l.slots }()
	return l.h.Hash(password)
}

func (l *limited) Verify(password, encoded string) (bool, error) {
	l.slots <- struct{}{}
	defer func() { <-l.slots }()
	return l.h.Verify(password, encoded)
}

// NeedsRehash only parses a string, so it does not take a slot.
func (l *limited) NeedsRehash(encoded string) bool { return l.h.NeedsRehash(encoded) }
