// Command containeraudit checks the three places this article is about:
// the layers of a saved image, the Dockerfile that built it, and the settings
// of a running container. It exits 1 when it finds anything, so a CI job can
// fail the build.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"handbook/playground/golang/cyber/container-image-attack-surface/dockerlint"
	"handbook/playground/golang/cyber/container-image-attack-surface/imagescan"
	"handbook/playground/golang/cyber/container-image-attack-surface/runaudit"
)

const usage = `usage:
  containeraudit image <image.tar>       scan layers of a "docker save" archive
  containeraudit dockerfile <Dockerfile> lint a Dockerfile
  containeraudit run <inspect.json|->    audit "docker inspect" output ("-" reads stdin)`

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	var (
		found int
		err   error
	)
	switch args[0] {
	case "image":
		found, err = scanImage(args[1], stdout)
	case "dockerfile":
		found, err = lintDockerfile(args[1], stdout)
	case "run":
		found, err = auditRun(args[1], stdin, stdout)
	default:
		fmt.Fprintln(stderr, usage)
		return 2
	}
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	if found > 0 {
		return 1
	}
	fmt.Fprintln(stdout, "ok: nothing found")
	return 0
}

func scanImage(file string, out io.Writer) (int, error) {
	findings, err := imagescan.Scan(func() (io.ReadCloser, error) { return os.Open(file) })
	if err != nil {
		return 0, err
	}
	for _, f := range findings {
		state := "still in the final image"
		if f.DeletedIn > 0 {
			// The file is hidden from the running container, not removed from the image.
			state = fmt.Sprintf("deleted in layer %d, but layer %d still stores it", f.DeletedIn, f.Layer)
		}
		fmt.Fprintf(out, "layer %d  %s  [%s]  %s\n", f.Layer, f.Path, strings.Join(f.Rules, ", "), state)
	}
	return len(findings), nil
}

func lintDockerfile(file string, out io.Writer) (int, error) {
	f, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	issues, err := dockerlint.Lint(f)
	for _, i := range issues {
		fmt.Fprintf(out, "%s:%d  %s  %s\n", file, i.Line, i.Rule, i.Detail)
	}
	return len(issues), err
}

func auditRun(file string, stdin io.Reader, out io.Writer) (int, error) {
	in := stdin
	if file != "-" {
		f, err := os.Open(file)
		if err != nil {
			return 0, err
		}
		defer f.Close()
		in = f
	}
	issues, err := runaudit.Audit(in)
	for _, i := range issues {
		fmt.Fprintf(out, "%-8s %s  %s  %s\n", i.Severity, i.Container, i.Rule, i.Detail)
	}
	return len(issues), err
}
