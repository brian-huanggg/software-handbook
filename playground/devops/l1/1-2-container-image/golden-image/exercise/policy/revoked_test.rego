package main

import rego.v1

revoked := "registry.internal.example/golden/go-static@sha256:3333333333333333333333333333333333333333333333333333333333333333"

ok := "registry.internal.example/golden/go-static@sha256:2222222222222222222222222222222222222222222222222222222222222222"

test_revoked_digest_denied if {
	count(deny) == 1 with input as [{"Cmd": "from", "Value": [revoked]}]
}

test_other_digest_passes if {
	count(deny) == 0 with input as [{"Cmd": "from", "Value": [ok]}]
}
