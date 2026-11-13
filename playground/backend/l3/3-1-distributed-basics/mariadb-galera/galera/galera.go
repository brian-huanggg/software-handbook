// Package galera simulates certification-based, virtually synchronous
// replication: a commit is acknowledged once every node has received and
// certified the write-set, not once every node has applied it.
package galera

import "errors"

var (
	// ErrConflict mirrors MariaDB ERROR 1213: the transaction lost certification.
	ErrConflict = errors.New("deadlock found when trying to get lock; try restarting transaction")
	// ErrNoQuorum mirrors ERROR 1047: the node is outside the primary component.
	ErrNoQuorum = errors.New("WSREP has not yet prepared node for application use")
	ErrNodeDown = errors.New("node is down")
)

type entry struct {
	seqno  uint64
	writes map[string]string
}

type Node struct {
	Name    string
	up      bool
	data    map[string]string
	applied uint64  // seqno of the last write-set applied here
	queue   []entry // received and certified, not yet applied
}

type Cluster struct {
	nodes []*Node
	seqno uint64
	index map[string]uint64 // key -> seqno of the last committed writer
	log   []entry           // stands in for the gcache used by IST
}

func New(names ...string) *Cluster {
	c := &Cluster{index: map[string]uint64{}}
	for _, name := range names {
		c.nodes = append(c.nodes, &Node{Name: name, up: true, data: map[string]string{}})
	}
	return c
}

// Certify is the deterministic check every node runs on a write-set.
// It fails when another transaction committed a write to one of the same
// keys after this transaction's snapshot was taken.
func Certify(index map[string]uint64, snapshot uint64, writes map[string]string) error {
	for key := range writes {
		if index[key] > snapshot {
			return ErrConflict
		}
	}
	return nil
}

// hasQuorum is simplified: it counts against the configured size, while
// real Galera counts against the last primary component.
func (c *Cluster) hasQuorum() bool {
	up := 0
	for _, n := range c.nodes {
		if n.up {
			up++
		}
	}
	return up*2 > len(c.nodes)
}

type Tx struct {
	c        *Cluster
	origin   int
	snapshot uint64
	writes   map[string]string
}

// Begin starts a transaction on one node. Its snapshot is whatever that
// node has applied so far, which may be behind the cluster.
func (c *Cluster) Begin(node int) *Tx {
	return &Tx{c: c, origin: node, snapshot: c.nodes[node].applied, writes: map[string]string{}}
}

func (tx *Tx) Set(key, val string) { tx.writes[key] = val }

func (tx *Tx) Commit() error {
	c := tx.c
	origin := c.nodes[tx.origin]
	if !origin.up {
		return ErrNodeDown
	}
	if !c.hasQuorum() {
		return ErrNoQuorum
	}
	if err := Certify(c.index, tx.snapshot, tx.writes); err != nil {
		return err
	}
	c.seqno++
	e := entry{seqno: c.seqno, writes: tx.writes}
	for key := range tx.writes {
		c.index[key] = e.seqno
	}
	c.log = append(c.log, e)
	// Every live node receives the write-set before the client sees success.
	for _, n := range c.nodes {
		if n.up {
			n.queue = append(n.queue, e)
		}
	}
	// Only the origin applies it now. The others apply asynchronously.
	origin.Apply()
	return nil
}

// Apply drains the node's queue, standing in for its applier threads.
func (n *Node) Apply() {
	for _, e := range n.queue {
		for k, v := range e.writes {
			n.data[k] = v
		}
		n.applied = e.seqno
	}
	n.queue = nil
}

// Node returns a node by index.
func (c *Cluster) Node(i int) *Node { return c.nodes[i] }

// Read returns what the node has applied. It can be stale.
func (c *Cluster) Read(node int, key string) (string, bool) {
	v, ok := c.nodes[node].data[key]
	return v, ok
}

// ReadSynced applies everything already certified before reading, like
// SET SESSION wsrep_sync_wait=1.
func (c *Cluster) ReadSynced(node int, key string) (string, bool) {
	c.nodes[node].Apply()
	return c.Read(node, key)
}

func (c *Cluster) Kill(node int) { c.nodes[node].up = false }

// Rejoin brings a node back and replays the write-sets it missed,
// which is what IST does when the donor still has them in its gcache.
func (c *Cluster) Rejoin(node int) {
	n := c.nodes[node]
	n.up = true
	for _, e := range c.log {
		if e.seqno > n.applied {
			n.queue = append(n.queue, e)
		}
	}
	n.Apply()
}
