// Package dockerlint flags Dockerfile lines that weaken the image supply chain:
// base images that can change under you, secrets copied into layers, and
// downloads that run without any verification.
package dockerlint

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

// Issue is one finding on one Dockerfile line.
type Issue struct {
	Line   int
	Rule   string
	Detail string
}

var (
	secretName   = regexp.MustCompile(`(?i)(token|secret|passw|api_?key|private_?key)`)
	secretSource = regexp.MustCompile(`(^|/)(\.env|[^/]+\.env|\.npmrc|\.netrc|id_(rsa|ed25519|ecdsa)|[^/]+\.pem)$`)
	pipeToShell  = regexp.MustCompile(`(curl|wget)\b[^|]*\|\s*(sudo\s+)?(ba)?sh\b`)
)

type instruction struct {
	line int
	op   string
	args string
}

// Lint reads a Dockerfile and returns its findings in line order.
func Lint(r io.Reader) ([]Issue, error) {
	ins, err := parse(r)
	if err != nil {
		return nil, err
	}
	var out []Issue
	stages := map[string]bool{}
	finalUser := ""
	finalLine := 0

	for _, in := range ins {
		add := func(rule, detail string) { out = append(out, Issue{in.line, rule, detail}) }
		fields := strings.Fields(in.args)
		switch in.op {
		case "FROM":
			finalUser, finalLine = "", in.line
			fields = withoutFlags(fields)
			if len(fields) == 0 {
				continue
			}
			image := fields[0]
			// "FROM build" refers to an earlier stage, not to a registry image.
			if image != "scratch" && !stages[strings.ToLower(image)] && !strings.Contains(image, "$") && !strings.Contains(image, "@sha256:") {
				add("unpinned-base", "tag can move to a different image; pin "+image+" by @sha256 digest")
			}
			if len(fields) >= 3 && strings.EqualFold(fields[1], "AS") {
				stages[strings.ToLower(fields[2])] = true
			}
		case "USER":
			finalUser = in.args
		case "ENV", "ARG":
			for _, f := range fields {
				name, _, hasValue := strings.Cut(f, "=")
				if secretName.MatchString(name) {
					add("secret-in-"+strings.ToLower(in.op), name+" is stored in image metadata and visible in `docker history`")
				}
				if !hasValue {
					break // legacy "ENV KEY value" form: only the first word is a name
				}
			}
		case "COPY", "ADD":
			args := withoutFlags(fields)
			for _, src := range args[:max(len(args)-1, 0)] {
				if secretSource.MatchString(src) {
					add("secret-copied", src+" ends up in a layer; deleting it later does not remove it")
				}
				if in.op == "ADD" && (strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://")) {
					add("remote-add", "downloads "+src+" without checking a checksum")
				}
			}
		case "RUN":
			if pipeToShell.MatchString(in.args) {
				add("pipe-to-shell", "runs a downloaded script without verifying it")
			}
		}
	}
	if finalLine > 0 && isRoot(finalUser) {
		out = append(out, Issue{finalLine, "runs-as-root", "the final stage never switches to a non-root USER"})
	}
	return out, nil
}

func isRoot(user string) bool {
	return user == "" || user == "root" || user == "0" || strings.HasPrefix(user, "root:") || strings.HasPrefix(user, "0:")
}

func withoutFlags(fields []string) []string {
	var out []string
	for _, f := range fields {
		if !strings.HasPrefix(f, "--") {
			out = append(out, f)
		}
	}
	return out
}

// parse joins backslash continuations and drops comments and blank lines.
func parse(r io.Reader) ([]instruction, error) {
	var out []instruction
	sc := bufio.NewScanner(r)
	var cur strings.Builder
	start, n := 0, 0
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		op, args, _ := strings.Cut(strings.TrimSpace(cur.String()), " ")
		out = append(out, instruction{start, strings.ToUpper(op), strings.TrimSpace(args)})
		cur.Reset()
	}
	for sc.Scan() {
		n++
		text := strings.TrimSpace(sc.Text())
		if cur.Len() == 0 && (text == "" || strings.HasPrefix(text, "#")) {
			continue
		}
		if cur.Len() == 0 {
			start = n
		}
		if strings.HasSuffix(text, `\`) {
			cur.WriteString(strings.TrimSuffix(text, `\`) + " ")
			continue
		}
		cur.WriteString(text)
		flush()
	}
	flush()
	return out, sc.Err()
}
