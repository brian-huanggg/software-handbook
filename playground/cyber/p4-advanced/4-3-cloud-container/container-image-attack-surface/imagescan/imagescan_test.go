package imagescan

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"slices"
	"testing"
)

type file struct{ name, body string }

// layer builds one layer tarball, optionally gzip-compressed.
func layer(t *testing.T, compress bool, files ...file) []byte {
	t.Helper()
	var buf bytes.Buffer
	var w io.Writer = &buf
	var gz *gzip.Writer
	if compress {
		gz = gzip.NewWriter(&buf)
		w = gz
	}
	tw := tar.NewWriter(w)
	for _, f := range files {
		hdr := &tar.Header{Name: f.name, Mode: 0o644, Size: int64(len(f.body)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(f.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if gz != nil {
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return buf.Bytes()
}

// image assembles a docker-save style archive: manifest.json plus one blob per layer.
func image(t *testing.T, layers ...[]byte) Opener {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	put := func(name string, data []byte) {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	var names []string
	for i, l := range layers {
		name := "blobs/sha256/layer" + string(rune('a'+i))
		names = append(names, name)
		put(name, l)
	}
	manifest, err := json.Marshal([]map[string]any{{"Layers": names}})
	if err != nil {
		t.Fatal(err)
	}
	put("manifest.json", manifest) // last, like `docker save`: the scan must not assume it comes first
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(buf.Bytes())), nil }
}

const token = "ADMIN_TOKEN=sho-demo-0123456789abcdef\n"

func TestDeletedSecretIsStillFound(t *testing.T) {
	open := image(t,
		layer(t, true, file{"etc/os-release", "NAME=Alpine\n"}),
		layer(t, true, file{"app/.env", token}),
		layer(t, true, file{"app/.wh..env", ""}),
	)
	got, err := Scan(open)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 finding, got %+v", got)
	}
	f := got[0]
	if f.Layer != 2 || f.Path != "/app/.env" || f.DeletedIn != 3 {
		t.Errorf("got %+v, want layer 2, /app/.env, deleted in 3", f)
	}
	if !slices.Equal(f.Rules, []string{"secret-file", "secret-assignment"}) {
		t.Errorf("rules = %v", f.Rules)
	}
}

func TestCleanImageHasNoFindings(t *testing.T) {
	open := image(t,
		layer(t, true, file{"etc/os-release", "NAME=Alpine\n"}),
		layer(t, false, file{"app/shortener", "\x7fELF\x00\x00binary"}),
	)
	got, err := Scan(open)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("want no findings, got %+v", got)
	}
}

func TestDirectoryWhiteoutHidesFilesBelow(t *testing.T) {
	open := image(t,
		layer(t, false, file{"app/conf/.env", token}),
		layer(t, false, file{".wh.app", ""}),
	)
	got, err := Scan(open)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].DeletedIn != 2 {
		t.Errorf("got %+v, want one finding deleted in layer 2", got)
	}
}

func TestFileAddedAgainAfterDeleteIsPresent(t *testing.T) {
	open := image(t,
		layer(t, false, file{"app/.env", token}),
		layer(t, false, file{"app/.wh..env", ""}),
		layer(t, false, file{"app/.env", token}),
	)
	got, err := Scan(open)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].DeletedIn != 2 || got[1].DeletedIn != 0 {
		t.Errorf("got %+v, want first deleted in 2, second still present", got)
	}
}

func TestContentRulesCatchSecretsInInnocentNames(t *testing.T) {
	awsKey := "AKIA" + "ABCDEFGHIJKLMNOP"
	open := image(t, layer(t, false,
		file{"srv/notes.txt", "key = " + awsKey + "\n"},
		file{"srv/backup.dat", "-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n"},
	))
	got, err := Scan(open)
	if err != nil {
		t.Fatal(err)
	}
	rules := map[string][]string{}
	for _, f := range got {
		rules[f.Path] = f.Rules
	}
	if !slices.Contains(rules["/srv/notes.txt"], "aws-access-key") {
		t.Errorf("notes.txt rules = %v", rules["/srv/notes.txt"])
	}
	if !slices.Contains(rules["/srv/backup.dat"], "private-key") {
		t.Errorf("backup.dat rules = %v", rules["/srv/backup.dat"])
	}
}

func TestNotAnImageArchive(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	open := func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(buf.Bytes())), nil }
	if _, err := Scan(open); err == nil {
		t.Error("want an error for an archive without manifest.json")
	}
}
