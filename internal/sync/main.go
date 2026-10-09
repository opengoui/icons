// Command sync downloads a Lucide release and regenerates the embedded icon
// data of the icons module: svg/*.svg, index.json, name/name_gen.go and
// LICENSE.lucide.
//
// Run it from the module root:
//
//	go run ./internal/sync                  # re-sync the version in index.json
//	go run ./internal/sync -version latest  # upgrade to the newest release
//	go run ./internal/sync -version 1.53.0  # pin an exact release
//	go run ./internal/sync -archive l.tgz   # use a local source tarball
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	releaseAPI = "https://api.github.com/repos/lucide-icons/lucide/releases/latest"
	archiveURL = "https://codeload.github.com/lucide-icons/lucide/tar.gz/refs/tags/"
)

func main() {
	version := flag.String("version", "", `Lucide release tag, or "latest" (default: version recorded in index.json)`)
	archive := flag.String("archive", "", "read this local Lucide .tar.gz instead of downloading")
	root := flag.String("out", ".", "module root to write into")
	flag.Parse()
	if err := run(*root, *version, *archive); err != nil {
		fmt.Fprintln(os.Stderr, "sync:", err)
		os.Exit(1)
	}
}

func run(root, version, archive string) error {
	if version == "" {
		version = recordedVersion(root)
		if version == "" {
			version = "latest"
		}
	}
	client := &http.Client{Timeout: 2 * time.Minute}

	var body io.ReadCloser
	if archive != "" {
		f, err := os.Open(archive)
		if err != nil {
			return err
		}
		body = f
		if version == "latest" {
			return fmt.Errorf("-archive needs an explicit -version (the tag the archive was cut from)")
		}
	} else {
		if version == "latest" {
			v, err := latestVersion(client)
			if err != nil {
				return err
			}
			version = v
		}
		resp, err := get(client, archiveURL+version)
		if err != nil {
			return err
		}
		body = resp.Body
	}
	defer body.Close()

	b, err := readArchive(version, body)
	if err != nil {
		return err
	}
	files, err := b.render()
	if err != nil {
		return err
	}
	if err := write(root, files); err != nil {
		return err
	}
	fmt.Printf("synced Lucide %s: %d icons\n", version, len(b.svgs))
	return nil
}

// recordedVersion returns the version in root/index.json, or "" if absent.
func recordedVersion(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "index.json"))
	if err != nil {
		return ""
	}
	var idx struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(data, &idx) != nil {
		return ""
	}
	return idx.Version
}

func latestVersion(c *http.Client) (string, error) {
	resp, err := get(c, releaseAPI)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var rel struct {
		Tag string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", fmt.Errorf("decode latest release: %w", err)
	}
	if rel.Tag == "" {
		return "", fmt.Errorf("latest release has no tag_name")
	}
	return rel.Tag, nil
}

// get performs a GET, attaching GITHUB_TOKEN when set to avoid API rate limits.
func get(c *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" && strings.HasPrefix(url, "https://api.github.com/") {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return resp, nil
}
