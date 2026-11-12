// Package imagescan finds secrets that sit in the layers of an image saved
// with `docker save`. It reads every layer on its own, because a file removed
// by a later layer is still stored in the earlier one.
package imagescan

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
)

// Finding is one file that looks like a secret.
type Finding struct {
	Layer     int      // 1-based layer that stores the file
	Path      string   // absolute path inside the image
	Rules     []string // which rules matched
	DeletedIn int      // 1-based layer that whiteouts the file; 0 means it is still in the final image
}

// Opener returns a fresh reader over the image tarball. The scan reads the
// archive twice (manifest first, then layers), so one io.Reader is not enough.
type Opener func() (io.ReadCloser, error)

const maxContentBytes = 1 << 20 // do not read more than 1 MiB of a single file

type pathRule struct {
	name string
	re   *regexp.Regexp
}

var pathRules = []pathRule{
	{"secret-file", regexp.MustCompile(`(^|/)(\.env|[^/]+\.env|\.npmrc|\.netrc|id_(rsa|ed25519|ecdsa)|\.aws/credentials)$`)},
}

var contentRules = []pathRule{
	{"private-key", regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`)},
	{"aws-access-key", regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)},
	{"secret-assignment", regexp.MustCompile(`(?i)(token|secret|passw(or)?d|api[_-]?key)\w*\s*[=:]\s*["']?[\w+/=.-]{12,}`)},
}

type layerResult struct {
	findings  []Finding
	whiteouts []string
}

// Scan returns every secret-looking file in every layer, oldest layer first.
func Scan(open Opener) ([]Finding, error) {
	layers, err := layerPaths(open)
	if err != nil {
		return nil, err
	}
	index := make(map[string]int, len(layers))
	for i, l := range layers {
		index[l] = i
	}
	results := make([]layerResult, len(layers))
	seen := 0

	rc, err := open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	tr := tar.NewReader(rc)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		i, ok := index[path.Clean(hdr.Name)]
		if !ok {
			continue
		}
		if results[i], err = scanLayer(tr); err != nil {
			return nil, fmt.Errorf("layer %s: %w", hdr.Name, err)
		}
		seen++
	}
	if seen != len(layers) {
		return nil, fmt.Errorf("manifest lists %d layers but the archive has %d", len(layers), seen)
	}
	return merge(results), nil
}

// merge replays the layers in order so that a whiteout only marks findings
// from lower layers, and a file added again later counts as present again.
func merge(results []layerResult) []Finding {
	var out []Finding
	for i, res := range results {
		for _, w := range res.whiteouts {
			for j := range out {
				if out[j].DeletedIn == 0 && covers(w, out[j].Path) {
					out[j].DeletedIn = i + 1
				}
			}
		}
		for _, f := range res.findings {
			f.Layer = i + 1
			out = append(out, f)
		}
	}
	return out
}

func covers(deleted, p string) bool {
	return p == deleted || strings.HasPrefix(p, deleted+"/")
}

func layerPaths(open Opener) ([]string, error) {
	rc, err := open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	tr := tar.NewReader(rc)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("manifest.json not found: is this a `docker save` archive?")
		}
		if err != nil {
			return nil, err
		}
		if path.Clean(hdr.Name) != "manifest.json" {
			continue
		}
		var manifest []struct{ Layers []string }
		if err := json.NewDecoder(tr).Decode(&manifest); err != nil {
			return nil, fmt.Errorf("manifest.json: %w", err)
		}
		if len(manifest) == 0 {
			return nil, fmt.Errorf("manifest.json lists no image")
		}
		layers := make([]string, len(manifest[0].Layers))
		for i, l := range manifest[0].Layers {
			layers[i] = path.Clean(l)
		}
		return layers, nil
	}
}

func scanLayer(r io.Reader) (layerResult, error) {
	var res layerResult
	br := bufio.NewReader(r)
	r = br
	if magic, err := br.Peek(2); err == nil && magic[0] == 0x1f && magic[1] == 0x8b {
		gz, err := gzip.NewReader(br)
		if err != nil {
			return res, err
		}
		defer gz.Close()
		r = gz
	}
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return res, nil
		}
		if err != nil {
			return res, err
		}
		p := path.Clean("/" + hdr.Name)
		dir, base := path.Split(p)
		if strings.HasPrefix(base, ".wh.") {
			// ".wh..wh..opq" only hides the directory's older contents; it carries no file name.
			if base != ".wh..wh..opq" {
				res.whiteouts = append(res.whiteouts, path.Join(dir, strings.TrimPrefix(base, ".wh.")))
			}
			continue
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		rules, err := match(p, tr)
		if err != nil {
			return res, err
		}
		if len(rules) > 0 {
			res.findings = append(res.findings, Finding{Path: p, Rules: rules})
		}
	}
}

func match(p string, content io.Reader) ([]string, error) {
	var rules []string
	for _, r := range pathRules {
		if r.re.MatchString(p) {
			rules = append(rules, r.name)
		}
	}
	data, err := io.ReadAll(io.LimitReader(content, maxContentBytes))
	if err != nil {
		return nil, err
	}
	if bytes.IndexByte(data[:min(len(data), 512)], 0) >= 0 {
		return rules, nil // binary file: regexes over machine code only produce noise
	}
	for _, r := range contentRules {
		if r.re.Match(data) {
			rules = append(rules, r.name)
		}
	}
	return rules, nil
}
