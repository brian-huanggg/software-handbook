package main

import rego.v1

# Digests of golden image builds that were pulled after a vulnerability was found.
revoked_digests := {"sha256:3333333333333333333333333333333333333333333333333333333333333333"}

# Exercise: make this rule reject any FROM whose digest is in revoked_digests.
deny contains msg if {
	false
	msg := ""
}
