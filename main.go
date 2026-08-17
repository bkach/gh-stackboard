package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

//go:embed web/*
var webFiles embed.FS

type apiResponse struct {
	Viewer      viewer  `json:"viewer"`
	Repository  string  `json:"repository"`
	RefreshedAt string  `json:"refreshedAt"`
	Source      string  `json:"source"`
	Warning     string  `json:"warning,omitempty"`
	Syncing     bool    `json:"syncing,omitempty"`
	Stacks      []stack `json:"stacks"`
}

type repositoryFlags []string

func (flags *repositoryFlags) String() string { return strings.Join(*flags, ",") }
func (flags *repositoryFlags) Set(value string) error {
	*flags = append(*flags, value)
	return nil
}

func main() {
	port := flag.Int("port", 4387, "local port to listen on (0 chooses an available port)")
	noOpen := flag.Bool("no-open", false, "do not open the browser automatically")
	var repositories repositoryFlags
	flag.Var(&repositories, "repo", "GitHub repository in OWNER/REPO format; may be repeated (defaults to the current repository)")
	useMock := flag.Bool("mock", false, "use built-in mock data instead of GitHub")
	flag.Parse()

	assets, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatal(err)
	}

	var cache struct {
		sync.Mutex
		value      apiResponse
		expiresAt  time.Time
		hasValue   bool
		refreshing bool
	}
	provider := func() (apiResponse, error) {
		if *useMock {
			return apiResponse{
				Viewer:      viewer{Login: "octocat", Name: "Octo Cat"},
				Repository:  "acme/all repositories",
				RefreshedAt: time.Now().Format(time.RFC3339),
				Source:      "mock",
				Stacks:      mockStacks,
			}, nil
		}
		cache.Lock()
		if time.Now().Before(cache.expiresAt) {
			response := cache.value
			cache.Unlock()
			return response, nil
		}
		if cache.hasValue {
			if !cache.refreshing {
				cache.refreshing = true
				go func() {
					response, err := loadGitHub(repositories)
					cache.Lock()
					defer cache.Unlock()
					cache.refreshing = false
					cache.expiresAt = time.Now().Add(time.Minute)
					if err == nil {
						cache.value = response
					} else {
						cache.value.Warning = "The latest GitHub refresh failed; showing the previous snapshot."
					}
				}()
			}
			response := cache.value
			response.Syncing = true
			cache.Unlock()
			return response, nil
		}
		cache.Unlock()

		response, err := loadGitHub(repositories)
		if err != nil {
			return apiResponse{}, err
		}
		cache.Lock()
		cache.value = response
		cache.expiresAt = time.Now().Add(time.Minute)
		cache.hasValue = true
		cache.Unlock()
		return response, nil
	}
	if !*useMock {
		fmt.Println("Fetching GitHub pull requests…")
		if _, err := provider(); err != nil {
			log.Fatal(err)
		}
	}

	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServer(http.FS(assets)))
	mux.HandleFunc("GET /api/stacks", func(w http.ResponseWriter, _ *http.Request) {
		response, err := provider()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(response)
	})

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		log.Fatalf("start Stackboard: %v", err)
	}
	url := "http://" + listener.Addr().String()
	mode := "GitHub"
	if *useMock {
		mode = "mock data"
	}
	fmt.Printf("Stackboard is running at %s (%s)\nPress Ctrl+C to stop.\n", url, mode)

	if !*noOpen {
		go openBrowser(url)
	}

	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second}
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func openBrowser(url string) {
	time.Sleep(150 * time.Millisecond)
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", url)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		command = exec.Command("xdg-open", url)
	}
	_ = command.Start()
}
