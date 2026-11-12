package main

import rego.v1

golden := "registry.internal.example/golden/go-static@sha256:2222222222222222222222222222222222222222222222222222222222222222"

test_golden_with_digest_passes if {
	count(deny) == 0 with input as [{"Cmd": "from", "Value": [golden]}]
}

test_stage_reference_passes if {
	count(deny) == 0 with input as [
		{"Cmd": "from", "Value": [golden, "AS", "build"]},
		{"Cmd": "from", "Value": ["build"]},
	]
}

test_public_image_denied if {
	count(deny) == 1 with input as [{"Cmd": "from", "Value": ["golang:latest"]}]
}

test_golden_without_digest_denied if {
	count(deny) == 1 with input as [{"Cmd": "from", "Value": ["registry.internal.example/golden/go-static:1.4"]}]
}

test_variable_denied if {
	count(deny) >= 1 with input as [{"Cmd": "from", "Value": ["${BASE}"]}]
}

test_copy_from_image_denied if {
	count(deny) == 1 with input as [
		{"Cmd": "from", "Value": [golden]},
		{"Cmd": "copy", "Flags": ["--from=docker.io/library/busybox:latest"], "Value": ["/bin/busybox", "/bin/busybox"]},
	]
}

test_copy_from_stage_passes if {
	count(deny) == 0 with input as [
		{"Cmd": "from", "Value": [golden, "AS", "build"]},
		{"Cmd": "copy", "Flags": ["--from=build"], "Value": ["/out/app", "/app"]},
	]
}
