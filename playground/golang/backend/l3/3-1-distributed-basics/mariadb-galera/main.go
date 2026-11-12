// Command mariadb-galera contrasts async replication with
// certification-based replication using in-memory simulations.
package main

import (
	"errors"
	"fmt"

	"handbook/playground/golang/backend/mariadb-galera/async"
	"handbook/playground/golang/backend/mariadb-galera/galera"
)

func main() {
	demoAsync()
	fmt.Println()
	demoGalera()
}

func demoAsync() {
	fmt.Println("== standard: async replication ==")
	c := async.New()
	for _, code := range []string{"a1", "b2", "c3", "d4", "e5"} {
		c.Write(code, "https://example.com/"+code)
	}
	c.Ship(2) // the replica is 3 writes behind when the primary dies
	fmt.Printf("replica lag before crash: %d writes\n", c.Lag())
	fmt.Printf("lost after failover: %d writes the clients were told succeeded\n", c.FailOver())
}

func demoGalera() {
	fmt.Println("== galera: certification-based replication ==")
	c := galera.New("db1", "db2", "db3")

	tx := c.Begin(0)
	tx.Set("a1", "https://example.com/a1")
	if err := tx.Commit(); err != nil {
		fmt.Println("commit failed:", err)
		return
	}
	_, ok := c.Read(1, "a1")
	fmt.Printf("plain read on db2 right after commit: found=%v\n", ok)
	_, ok = c.ReadSynced(1, "a1")
	fmt.Printf("read on db2 with wsrep_sync_wait:     found=%v\n", ok)

	// Two nodes update the same hot row from the same snapshot.
	x, y := c.Begin(0), c.Begin(1)
	x.Set("a1", "from-db1")
	y.Set("a1", "from-db2")
	fmt.Println("first writer: ", x.Commit())
	err := y.Commit()
	fmt.Println("second writer:", err, "| retry it:", errors.Is(err, galera.ErrConflict))

	c.Kill(1)
	c.Kill(2)
	tx = c.Begin(0)
	tx.Set("b2", "https://example.com/b2")
	fmt.Println("only db1 left:", tx.Commit())
}
