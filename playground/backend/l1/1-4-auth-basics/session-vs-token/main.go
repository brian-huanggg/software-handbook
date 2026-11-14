// Command session-vs-token logs a user out under each approach and shows
// which credentials still work afterwards.
package main

import (
	"fmt"
	"log"
	"time"

	"handbook/playground/backend/session-vs-token/authn"
)

func main() {
	key := []byte("demo-only-secret-demo-only-secret")

	// 1. Server-side session: delete the record, the login ends at once.
	s := authn.NewSession(24 * time.Hour)
	id, err := s.Login("42")
	if err != nil {
		log.Fatal(err)
	}
	_ = s.Logout(id)
	_, err = s.Authenticate(id)
	fmt.Println("session, after logout:      ", verdict(err))

	// 2. Stateless token: nothing to delete, so a copied token keeps working.
	t := authn.NewToken(key, 15*time.Minute)
	tok, err := t.Login("42")
	if err != nil {
		log.Fatal(err)
	}
	_ = t.Logout(tok)
	_, err = t.Authenticate(tok)
	fmt.Println("token, after logout:        ", verdict(err))

	// 3. Short access token + refresh token: the refresh side is revoked,
	// the access side lives until it expires.
	h := authn.NewHybrid(key, 5*time.Minute, 24*time.Hour)
	p, err := h.Login("42")
	if err != nil {
		log.Fatal(err)
	}
	_ = h.Logout(p.Refresh)
	_, err = h.Authenticate(p.Access)
	fmt.Println("hybrid access, after logout:", verdict(err))
	_, err = h.Refresh(p.Refresh)
	fmt.Println("hybrid refresh, after logout:", verdict(err))
}

func verdict(err error) string {
	if err != nil {
		return "rejected"
	}
	return "still accepted"
}
