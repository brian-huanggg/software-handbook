// Package exercise holds the starter code for the weighted round robin exercise.
package exercise

// Server is one upstream entry. Current is the running score the
// algorithm keeps between calls.
type Server struct {
	Name    string
	Weight  int
	Current int
}

// Next picks the server for the next request using smooth weighted round
// robin: add every server's Weight to its Current, pick the server with the
// highest Current, then subtract the sum of all weights from the winner.
// It returns nil when servers is empty.
func Next(servers []*Server) *Server {
	// TODO: implement.
	return nil
}
