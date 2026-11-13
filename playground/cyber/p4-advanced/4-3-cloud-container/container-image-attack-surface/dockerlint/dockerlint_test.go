package dockerlint

import (
	"os"
	"slices"
	"strings"
	"testing"
)

func lint(t *testing.T, dockerfile string) []string {
	t.Helper()
	issues, err := Lint(strings.NewReader(dockerfile))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, i := range issues {
		out = append(out, i.Rule)
	}
	return out
}

func TestLabDockerfiles(t *testing.T) {
	leaky, err := os.ReadFile("../lab/Dockerfile.leaky")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"unpinned-base", "secret-copied", "runs-as-root"}
	if got := lint(t, string(leaky)); !slices.Equal(got, want) {
		t.Errorf("leaky: rules = %v, want %v", got, want)
	}

	fixed, err := os.ReadFile("../lab/Dockerfile.fixed")
	if err != nil {
		t.Fatal(err)
	}
	if got := lint(t, string(fixed)); len(got) != 0 {
		t.Errorf("fixed: want no findings, got %v", got)
	}
}

func TestMultiStageOnlyChecksTheFinalStageUser(t *testing.T) {
	const df = `
FROM golang:1.26@sha256:aaaa AS build
RUN go build -o /out/app .

FROM build AS test
RUN go test ./...

FROM gcr.io/distroless/static@sha256:bbbb
COPY --from=build /out/app /app
USER nonroot
ENTRYPOINT ["/app"]
`
	if got := lint(t, df); len(got) != 0 {
		t.Errorf("want no findings, got %v", got)
	}
}

func TestSecretsInArgAndEnv(t *testing.T) {
	const df = `FROM scratch
ARG NPM_TOKEN
ENV REGION=eu DB_PASSWORD=hunter2
ENV API_KEY legacy-form-value
USER 1000
`
	want := []string{"secret-in-arg", "secret-in-env", "secret-in-env"}
	if got := lint(t, df); !slices.Equal(got, want) {
		t.Errorf("rules = %v, want %v", got, want)
	}
}

func TestDownloadsWithoutVerification(t *testing.T) {
	const df = `FROM scratch
ADD https://example.test/tool.tar.gz /opt/
RUN curl -fsSL https://example.test/install.sh \
    | sh
USER 1000
`
	want := []string{"remote-add", "pipe-to-shell"}
	if got := lint(t, df); !slices.Equal(got, want) {
		t.Errorf("rules = %v, want %v", got, want)
	}
}
