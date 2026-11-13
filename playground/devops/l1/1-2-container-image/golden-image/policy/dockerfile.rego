package main

import rego.v1

# Only images under this prefix count as golden images.
golden_prefix := "registry.internal.example/golden/"

# Names given with `FROM ... AS name`; later stages may build on these.
stage_names contains name if {
	some instruction in input
	instruction.Cmd == "from"
	count(instruction.Value) == 3
	lower(instruction.Value[1]) == "as"
	name := lower(instruction.Value[2])
}

deny contains msg if {
	some instruction in input
	instruction.Cmd == "from"
	ref := instruction.Value[0]
	not lower(ref) in stage_names
	not startswith(ref, golden_prefix)
	msg := sprintf("FROM %s is not a golden image (must start with %s)", [ref, golden_prefix])
}

deny contains msg if {
	some instruction in input
	instruction.Cmd == "from"
	ref := instruction.Value[0]
	startswith(ref, golden_prefix)
	not contains(ref, "@sha256:")
	msg := sprintf("FROM %s is not pinned by digest (add @sha256:...)", [ref])
}

# `FROM ${BASE}` is resolved at build time, so the policy cannot see the value.
deny contains msg if {
	some instruction in input
	instruction.Cmd == "from"
	ref := instruction.Value[0]
	contains(ref, "$")
	msg := sprintf("FROM %s uses a variable; write the image reference literally", [ref])
}

# `COPY --from=<image>` pulls an image without going through FROM.
deny contains msg if {
	some instruction in input
	instruction.Cmd == "copy"
	some flag in instruction.Flags
	startswith(flag, "--from=")
	source := lower(trim_prefix(flag, "--from="))
	not source in stage_names
	not regex.match(`^[0-9]+$`, source)
	msg := sprintf("COPY %s pulls an image outside FROM; use a golden image stage instead", [flag])
}
