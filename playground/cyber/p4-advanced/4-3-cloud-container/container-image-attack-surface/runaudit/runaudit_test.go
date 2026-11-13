package runaudit

import (
	"os"
	"slices"
	"strings"
	"testing"
)

func rules(t *testing.T, json string) []string {
	t.Helper()
	issues, err := Audit(strings.NewReader(json))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, i := range issues {
		out = append(out, i.Rule)
	}
	return out
}

func TestPrivilegedLogAgent(t *testing.T) {
	f, err := os.Open("../testdata/inspect-privileged.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	issues, err := Audit(f)
	if err != nil {
		t.Fatal(err)
	}
	sev := map[string]string{}
	for _, i := range issues {
		sev[i.Rule] = i.Severity
	}
	for rule, want := range map[string]string{"privileged": "critical", "docker-sock": "critical", "host-pid": "high"} {
		if sev[rule] != want {
			t.Errorf("%s: severity %q, want %q (all: %v)", rule, sev[rule], want, sev)
		}
	}
	if _, noisy := sev["dangerous-capability"]; noisy {
		t.Error("privileged already implies every capability; do not list them one by one")
	}
}

func TestHardenedContainerIsClean(t *testing.T) {
	data, err := os.ReadFile("../testdata/inspect-hardened.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := rules(t, string(data)); len(got) != 0 {
		t.Errorf("hardened container flagged: %v", got)
	}
}

func TestCapabilityNamesAreNormalised(t *testing.T) {
	const sample = `[{"Name":"/x","Config":{"User":"65532"},"HostConfig":{
		"CapAdd":["CAP_SYS_ADMIN","net_admin","CHOWN"],"CapDrop":["ALL"],
		"ReadonlyRootfs":true,"SecurityOpt":["no-new-privileges:true"]}}]`
	got := rules(t, sample)
	if want := []string{"dangerous-capability", "dangerous-capability"}; !slices.Equal(got, want) {
		t.Errorf("rules = %v, want %v (CHOWN is harmless)", got, want)
	}
}

func TestSensitiveHostPathButNotDockerSock(t *testing.T) {
	const sample = `[{"Name":"/x","Config":{"User":"65532"},"HostConfig":{
		"CapDrop":["ALL"],"ReadonlyRootfs":true,"SecurityOpt":["no-new-privileges"]},
		"Mounts":[{"Source":"/etc","Destination":"/host-etc"},{"Source":"/srv/data","Destination":"/data"}]}]`
	got := rules(t, sample)
	if want := []string{"sensitive-host-path"}; !slices.Equal(got, want) {
		t.Errorf("rules = %v, want %v", got, want)
	}
}

func TestBadJSON(t *testing.T) {
	if _, err := Audit(strings.NewReader("not json")); err == nil {
		t.Error("want an error for input that is not docker inspect output")
	}
}
