package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

const (
	defaultTorPath       = "/usr/bin/tor"
	defaultWebtunnelPath = "/usr/local/bin/webtunnel"

	defaultBootstrapTimeout = 120 * time.Second
	bridgeConnectTimeout    = 30 * time.Second
)

type torCheckResponse struct {
	IsTor bool   `json:"IsTor"`
	IP    string `json:"IP"`
}

type bridgeEntry struct {
	bridge  string
	comment string
}

type bridgeResult struct {
	index  int
	label  string
	bridge string
	port   int
	ok     bool
	ip     string
	output string
	err    error
}

type runConfig struct {
	jsonOnly bool
}

func jsonModeRequested() bool {
	for _, arg := range os.Args[1:] {
		if arg == "--json" || arg == "-json" {
			return true
		}
	}
	if value := strings.TrimSpace(os.Getenv("JSON_OUTPUT")); value != "" {
		return value == "1" || value == "true" || value == "yes" || value == "on"
	}
	return false
}

func main() {
	if err := run(); err != nil {
		if jsonModeRequested() {
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := runConfig{jsonOnly: false}
	flag.BoolVar(&cfg.jsonOnly, "json", false, "emit machine-readable JSON output only")
	flag.Parse()

	if value := strings.TrimSpace(os.Getenv("JSON_OUTPUT")); value != "" {
		if value == "1" || value == "true" || value == "yes" || value == "on" {
			cfg.jsonOnly = true
		}
	}

	if cfg.jsonOnly {
		return runJSON()
	}

	if value := strings.TrimSpace(os.Getenv("CONNECTION_STRING")); value != "" {
		result := checkBridge(value, "env", 1080)
		printResult(result)
		if !result.ok {
			return errors.New("bridge validation failed")
		}
		return nil
	}

	path, err := inputFile()
	if err != nil {
		return err
	}
	return runFile(path)
}

func runJSON() error {
	if value := strings.TrimSpace(os.Getenv("CONNECTION_STRING")); value != "" {
		result := checkBridge(value, "env", 1080)
		if err := emitJSON(result.toJSON()); err != nil {
			return err
		}
		if !result.ok {
			return errors.New("bridge validation failed")
		}
		return nil
	}

	path, err := inputFile()
	if err != nil {
		return err
	}
	results, err := checkFile(path)
	if err != nil {
		return err
	}
	payload := make([]map[string]any, len(results))
	failed := false
	for i, result := range results {
		payload[i] = result.toJSON()
		if !result.ok {
			failed = true
		}
	}
	if err := emitJSON(payload); err != nil {
		return err
	}
	if failed {
		return errors.New("bridge validation failed")
	}
	return nil
}

func inputFile() (string, error) {
	if value := strings.TrimSpace(os.Getenv("CONNECTION_STRINGS_FILE")); value != "" {
		return value, nil
	}
	if value := strings.TrimSpace(os.Getenv("INPUT_FILE")); value != "" {
		return value, nil
	}
	if flag.NArg() > 0 {
		return flag.Arg(0), nil
	}
	return "", errors.New("provide CONNECTION_STRING or INPUT_FILE/CONNECTION_STRINGS_FILE or a file path as a CLI argument")
}

// checkFile validates every bridge in the file in parallel, each with its own
// Tor instance and SOCKS port. Results keep the order of the file.
func checkFile(path string) ([]bridgeResult, error) {
	entries, err := parseBridgeFile(path)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no bridge entries found in %s", path)
	}

	results := make([]bridgeResult, len(entries))
	var wg sync.WaitGroup
	wg.Add(len(entries))
	for i, entry := range entries {
		go func(idx int, e bridgeEntry) {
			defer wg.Done()
			results[idx] = checkBridge(e.bridge, e.comment, 1080+idx)
			results[idx].index = idx
		}(i, entry)
	}
	wg.Wait()
	return results, nil
}

func (r bridgeResult) toJSON() map[string]any {
	out := map[string]any{
		"ok":     r.ok,
		"label":  r.label,
		"bridge": r.bridge,
	}
	if r.err != nil {
		out["error"] = r.err.Error()
		return out
	}
	out["isTor"] = r.ok
	out["ip"] = r.ip
	return out
}

func emitJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func printResult(result bridgeResult) {
	switch {
	case result.err != nil:
		fmt.Printf("❌ %s failed: %v\n", result.label, result.err)
	case result.ok && result.ip == "":
		fmt.Printf("✅ Tor connection is true for %s\n", result.label)
	case result.ok:
		fmt.Printf("✅ Tor connection is true for %s # Check Tor API-Response: {\"IsTor\":true,\"IP\":\"%s\"}\n", result.label, result.ip)
	default:
		fmt.Printf("❌ Tor connection is false for %s # Check Tor API-Response: %s\n", result.label, result.output)
	}
}

func runFile(path string) error {
	results, err := checkFile(path)
	if err != nil {
		return err
	}

	failed := false
	for idx, result := range results {
		if idx > 0 {
			fmt.Println()
		}
		printResult(result)
		if !result.ok {
			failed = true
		}
	}
	if failed {
		return errors.New("bridge validation failed")
	}
	return nil
}

func parseBridgeFile(path string) ([]bridgeEntry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	entries := make([]bridgeEntry, 0, len(lines))
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		comment := ""
		if idx := strings.Index(line, "#"); idx >= 0 {
			comment = strings.TrimSpace(line[idx+1:])
			line = strings.TrimSpace(line[:idx])
		}
		if line == "" {
			continue
		}

		entries = append(entries, bridgeEntry{bridge: line, comment: comment})
	}

	return entries, nil
}

// torProgress tracks what a Tor instance reports on its log while bootstrapping.
type torProgress struct {
	mu       sync.Mutex
	percent  int
	phase    string
	lastWarn string
}

var bootstrapRe = regexp.MustCompile(`Bootstrapped (\d+)% \(([^)]*)\)`)

// watch consumes Tor's log and closes done once bootstrap reaches 100%.
func (p *torProgress) watch(log io.Reader, done chan<- struct{}) {
	scanner := bufio.NewScanner(log)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	finished := false
	for scanner.Scan() {
		line := scanner.Text()
		if m := bootstrapRe.FindStringSubmatch(line); m != nil {
			percent, _ := strconv.Atoi(m[1])
			p.mu.Lock()
			p.percent, p.phase = percent, m[2]
			p.mu.Unlock()
			if percent >= 100 && !finished {
				finished = true
				close(done)
			}
			continue
		}
		for _, level := range []string{"[warn] ", "[err] "} {
			if idx := strings.Index(line, level); idx >= 0 && !strings.Contains(line, "conflux") {
				p.mu.Lock()
				p.lastWarn = line[idx+len(level):]
				p.mu.Unlock()
			}
		}
	}
}

func (p *torProgress) String() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	s := fmt.Sprintf("bootstrap at %d%%", p.percent)
	if p.phase != "" {
		s += fmt.Sprintf(" (%s)", p.phase)
	}
	if p.lastWarn != "" {
		s += fmt.Sprintf(", last tor warning: %s", p.lastWarn)
	}
	return s
}

func (p *torProgress) connected() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.percent >= 10
}

// bootstrapTimeout is how long a bridge may take to fully bootstrap Tor.
// Slow bridges need well over 20s just to load relay descriptors.
func bootstrapTimeout() time.Duration {
	value := strings.TrimSpace(os.Getenv("BOOTSTRAP_TIMEOUT"))
	if value == "" {
		return defaultBootstrapTimeout
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if d, err := time.ParseDuration(value); err == nil && d > 0 {
		return d
	}
	return defaultBootstrapTimeout
}

// checkBridge starts a dedicated Tor instance for one bridge, waits until it
// has fully bootstrapped and then asks check.torproject.org through it.
func checkBridge(bridgeValue, label string, socksPort int) bridgeResult {
	bridge := strings.TrimSpace(bridgeValue)
	comment := strings.TrimSpace(label)
	if idx := strings.Index(bridge, "#"); idx >= 0 {
		comment = strings.TrimSpace(bridge[idx+1:])
		bridge = strings.TrimSpace(bridge[:idx])
	}
	if comment == "" {
		comment = "single bridge"
	}

	result := bridgeResult{label: comment, bridge: bridge, port: socksPort}
	fail := func(err error) bridgeResult {
		result.err = err
		result.output = err.Error()
		return result
	}

	if bridge == "" {
		return fail(errors.New("empty bridge string"))
	}

	if err := os.MkdirAll("/run/tor", 0o700); err != nil {
		return fail(fmt.Errorf("create /run/tor: %w", err))
	}
	if err := os.Chmod("/run/tor", 0o700); err != nil {
		return fail(fmt.Errorf("set /run/tor permissions: %w", err))
	}

	workDir, err := os.MkdirTemp("", "webtunnel-")
	if err != nil {
		return fail(fmt.Errorf("create temp dir: %w", err))
	}
	defer os.RemoveAll(workDir)

	configPath := filepath.Join(workDir, "torrc")
	config := strings.Join([]string{
		"UseBridges 1",
		fmt.Sprintf("SocksPort %d", socksPort),
		fmt.Sprintf("DataDirectory %s", workDir),
		fmt.Sprintf("PIDFile %s/tor.pid", workDir),
		"CookieAuthentication 0",
		"Log notice stdout",
		"ClientTransportPlugin webtunnel exec /usr/local/bin/webtunnel",
		fmt.Sprintf("Bridge %s", bridge),
		"",
	}, "\n")

	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		return fail(fmt.Errorf("write tor config: %w", err))
	}

	cmd := exec.Command(defaultTorPath, "-f", configPath)
	torLog, err := cmd.StdoutPipe()
	if err != nil {
		return fail(fmt.Errorf("attach tor log: %w", err))
	}
	if err := cmd.Start(); err != nil {
		return fail(fmt.Errorf("start tor: %w", err))
	}

	progress := &torProgress{}
	bootstrapped := make(chan struct{})
	exited := make(chan struct{})
	go func() {
		progress.watch(torLog, bootstrapped)
		close(exited)
	}()
	defer func() {
		_ = cmd.Process.Kill()
		<-exited
		_ = cmd.Wait()
	}()

	timeout := bootstrapTimeout()
	overall := time.After(timeout)
	connect := time.After(bridgeConnectTimeout)
wait:
	for {
		select {
		case <-bootstrapped:
			break wait
		case <-exited:
			return fail(fmt.Errorf("tor exited before bootstrap completed: %s", progress))
		case <-connect:
			if !progress.connected() {
				return fail(fmt.Errorf("could not connect to bridge within %s: %s", bridgeConnectTimeout, progress))
			}
		case <-overall:
			return fail(fmt.Errorf("tor bootstrap timed out after %s: %s", timeout, progress))
		}
	}

	output, err := fetchTorCheckViaHTTP(socksPort)
	if err != nil {
		// a fresh circuit can still be flaky right after bootstrap
		output, err = fetchTorCheckViaHTTP(socksPort)
	}
	if err != nil {
		return fail(err)
	}

	resp := torCheckResponse{}
	if err := json.Unmarshal(output, &resp); err != nil {
		return fail(fmt.Errorf("invalid tor response: %s", strings.TrimSpace(string(output))))
	}

	result.ok = resp.IsTor
	result.ip = resp.IP
	result.output = strings.TrimSpace(string(output))
	return result
}

func fetchTorCheckViaHTTP(socksPort int) ([]byte, error) {
	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%d", socksPort), nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("create SOCKS5 dialer: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
		},
	}

	resp, err := client.Get("https://check.torproject.org/api/ip")
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	return body, nil
}

func init() {
	if _, err := os.Stat(defaultTorPath); err != nil {
		fmt.Fprintf(os.Stderr, "tor binary not found at %s\n", defaultTorPath)
		os.Exit(1)
	}
	if _, err := os.Stat(defaultWebtunnelPath); err != nil {
		fmt.Fprintf(os.Stderr, "webtunnel binary not found at %s; please build it into the image\n", defaultWebtunnelPath)
		os.Exit(1)
	}
}
