// Package async simulates MariaDB standard replication: one primary,
// one replica, and a commit that does not wait for the replica.
package async

type write struct{ key, val string }

// Cluster is not safe for concurrent use; it exists to show data movement.
type Cluster struct {
	primary map[string]string
	replica map[string]string
	pending []write // committed on the primary, not yet on the replica
}

func New() *Cluster {
	return &Cluster{
		primary: map[string]string{},
		replica: map[string]string{},
	}
}

// Write commits on the primary and returns at once. The client is told
// "success" before the replica has seen the write.
func (c *Cluster) Write(key, val string) {
	c.primary[key] = val
	c.pending = append(c.pending, write{key, val})
}

// Ship delivers up to n pending writes, standing in for the replica's
// IO and SQL threads catching up.
func (c *Cluster) Ship(n int) {
	n = min(n, len(c.pending))
	for _, w := range c.pending[:n] {
		c.replica[w.key] = w.val
	}
	c.pending = c.pending[n:]
}

// Lag is the number of acknowledged writes the replica does not have.
func (c *Cluster) Lag() int { return len(c.pending) }

func (c *Cluster) ReadPrimary(key string) (string, bool) {
	v, ok := c.primary[key]
	return v, ok
}

func (c *Cluster) ReadReplica(key string) (string, bool) {
	v, ok := c.replica[key]
	return v, ok
}

// FailOver loses the primary and promotes the replica. It returns how
// many acknowledged writes disappeared with the old primary.
func (c *Cluster) FailOver() (lost int) {
	lost = len(c.pending)
	c.primary = c.replica
	c.replica = map[string]string{}
	c.pending = nil
	return lost
}
