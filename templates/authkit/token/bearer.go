package token

import (
	"net/http"
	"strings"

	"github.com/brian-huanggg/software-handbook/templates/authkit/authn"
)

// Bearer authenticates requests carrying "Authorization: Bearer <access jwt>".
type Bearer struct{ access *Access }

var _ authn.Authenticator = (*Bearer)(nil)

func NewBearer(a *Access) *Bearer { return &Bearer{access: a} }

// Challenge is the WWW-Authenticate value sent with a 401 (RFC 6750).
func (b *Bearer) Challenge() string { return `Bearer realm="authkit"` }

func (b *Bearer) Authenticate(r *http.Request) (authn.Principal, error) {
	scheme, tok, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || tok == "" || strings.ContainsAny(tok, " \t") {
		return authn.Principal{}, authn.ErrUnauthenticated
	}
	return b.access.Verify(tok)
}
