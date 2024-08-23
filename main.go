package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	unknownFilename = "unknown-filename"
	etagFilePattern = "etag-%s.txt"
	tmpDirPattern   = "autoprint-*"
)

var dryRun, force, verbose bool

func main() {
	flag.Usage = func() {
		fmt.Printf("usage: %s [options] <url>\n\n", os.Args[0])
		fmt.Println("options:")
		flag.PrintDefaults()
	}

	flag.BoolVar(&dryRun, "n", false, "don't print anything")
	flag.BoolVar(&force, "f", false, "ignore etag to force download")
	flag.BoolVar(&verbose, "v", false, "enable verbose logging")
	flag.Parse()

	// discard all log messages unless in verbose mode.
	if !verbose {
		log.SetOutput(io.Discard)
	}

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	url := args[0]
	os.Exit(innerMain(url, dryRun, force))
}

func innerMain(url string, dryRun, force bool) int {
	etagFn := getEtagFilename(etagFilePattern, url)
	etagPath := filepath.Join(os.TempDir(), etagFn)
	etagPrev, err := readLastEtag(etagPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "readLastModified: %v\n", err)
		return 1
	}

	log.Printf("previous etag: fn=%#v etag=%#v\n", etagFn, etagPrev)

	if force {
		log.Println("ignoring previous etag")
		etagPrev = ""
	}

	body, fn, etagNext, err := doRequest(url, etagPrev)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doRequest: %v\n", err)
		return 1
	}

	// if there's no response body (but also no error), there's nothing else to
	// do. the remote file was not modified since the last time we fetched it.
	if body == nil {
		fmt.Println("Not modified.")
		return 0
	}

	log.Printf("fetched: url=%#v, fn=%#v, etag=%v\n", url, fn, etagNext)

	defer func() {
		err := body.Close()
		if err != nil {
			// log error but otherwise ignore it.
			log.Printf("error: rc.Close: %v\n", err)
		}
	}()

	path, cb, err := writeOutput(body, fn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "writeOutput: %v\n", err)
		return 1
	}

	log.Printf("wrote to file: path=%v\n", path)

	defer func() {
		err := cb()
		if err != nil {
			// log error but otherwise ignore it.
			log.Printf("error: cb (from writeOutput): %v\n", err)
		}
	}()

	fmt.Println("Printing...")

	cmd := []string{"lp", "-s", "-o sides=two-sided-long-edge", "--", path}
	if dryRun {
		fmt.Printf("Would run: %v\n", strings.Join(cmd, " "))

	} else {
		log.Printf("running: %v\n", strings.Join(cmd, " "))
		cmd := exec.Command(cmd[0], cmd[1:]...)
		out, err := cmd.Output()
		if err != nil {
			exitErr, ok := err.(*exec.ExitError)
			if ok {
				fmt.Fprintf(os.Stderr, "cmd.Run: %s\n", exitErr.Stderr)
				return exitErr.ExitCode()
			} else {
				fmt.Fprintf(os.Stderr, "cmd.Run: %v\n", err)
				return 1
			}
		}

		// should be no output if there was no error code, because we set -s.
		// something's probably wrong if we get any, so at least log it.
		if len(out) > 0 {
			log.Printf("output: %s\n", out)
		}
	}

	err = writeLastEtag(etagPath, etagNext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "writeLastModified: %v\n", err)
		return 1
	}

	log.Printf("wrote etag: fn=%#v, etag=%#v\n", etagPath, etagNext)

	return 0
}

// getEtagFilename returns the path to the etag file for the given URL.
func getEtagFilename(pattern, url string) string {
	hash := sha256.Sum256([]byte(url))
	str := hex.EncodeToString(hash[:])
	return fmt.Sprintf(pattern, str)
}

// readLastEtag returns the etag from the given path, or an empty string if the
// path doesn't exist.
func readLastEtag(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("os.ReadFile: %w", err)
	}

	return string(b), nil
}

// writeLastEtag writes the given etag to the given path.
func writeLastEtag(path string, etag string) error {
	err := os.WriteFile(path, []byte(etag), 0644)
	if err != nil {
		return fmt.Errorf("os.WriteFile: %w", err)
	}

	return nil
}

// doRequest fetches the given URL, with the given ETag. If the server responds
// with StatusNotModified, then returns nothing. Othewise, returns the response
// body and the ETag and suggested filename extracted from the response headers.
//
// See: https://www.rfc-editor.org/rfc/rfc2616#section-14.19
func doRequest(url string, etag string) (io.ReadCloser, string, string, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", "", fmt.Errorf("http.NewRequest: %w", err)
	}

	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, "", "", fmt.Errorf("client.Do: %w", err)
	}

	if res.StatusCode == http.StatusNotModified {
		return nil, "", "", nil
	}

	cd := res.Header.Get("Content-Disposition")
	fn, err := extractFilename(cd)
	if err != nil {
		return nil, "", "", fmt.Errorf("extractFilename: %w", err)
	}

	etagNext := res.Header.Get("ETag")
	return res.Body, fn, etagNext, nil
}

// extractFilename returns the suggested filename extracted from the given
// Content-Disposition header.
// See: https://www.rfc-editor.org/rfc/rfc2183#section-2.3
func extractFilename(disp string) (string, error) {
	if disp == "" {
		return "", nil
	}

	_, params, err := mime.ParseMediaType(disp)
	if err != nil {
		return "", fmt.Errorf("mime.ParseMediaType: %w", err)
	}

	fn := params["filename"]
	if fn == "" {
		return "", nil
	}

	return fn, nil
}

// writeOutput writes the contents of the given reader to a temporary file with
// the given filename, and returns the full path to that file and a cleanup
// function to delete the file later.
func writeOutput(rc io.Reader, fn string) (string, func() error, error) {
	dir, err := os.MkdirTemp("", tmpDirPattern)
	if err != nil {
		return "", nil, fmt.Errorf("os.MkdirTemp: %w", err)
	}
	cb := func() error {
		return os.RemoveAll(dir)
	}

	if fn == "" {
		fn = unknownFilename
	}

	path := filepath.Join(dir, fn)
	file, err := os.Create(path)
	if err != nil {
		return path, cb, fmt.Errorf("os.Create: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, rc)
	return path, cb, err
}
