package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Kolbasen/design-practice-2/httptools"
	"github.com/Kolbasen/design-practice-2/signal"
)

type Server struct {
	URL     string
	IsAlive bool
}

var (
	port       = flag.Int("port", 8090, "load balancer port")
	timeoutSec = flag.Int("timeout-sec", 3, "request timeout time in seconds")
	https      = flag.Bool("https", false, "whether backends support HTTPs")

	traceEnabled = flag.Bool("trace", false, "whether to include tracing information into responses")
)

var (
	timeout     = time.Duration(*timeoutSec) * time.Second
	serversPool = []Server{
		{
			URL:     "server1:8080",
			IsAlive: true,
		},
		{
			URL:     "server2:8080",
			IsAlive: true,
		},
		{
			URL:     "server3:8080",
			IsAlive: true,
		},
	}
)

func getForwardServer(servers []Server, r *http.Request) (*Server, error) {
	aliveServers := []Server{}
	for _, server := range servers {
		if server.IsAlive == true {
			aliveServers = append(aliveServers, server)
		}
	}
	if len(aliveServers) == 0 {
		return nil, errors.New("No healthy servers")
	}
	if len(aliveServers) == 1 {
		return &aliveServers[0], nil
	}
	serverIndex := hash(r.URL.Path) % len(aliveServers)
	server := servers[serverIndex]
	return &server, nil
}

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

func health(dst string) bool {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s://%s/health", scheme(), dst), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	if resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func forward(dst string, rw http.ResponseWriter, r *http.Request) error {
	ctx, _ := context.WithTimeout(r.Context(), timeout)
	fwdRequest := r.Clone(ctx)
	fwdRequest.RequestURI = ""
	fwdRequest.URL.Host = dst
	fwdRequest.URL.Scheme = scheme()
	fwdRequest.Host = dst

	resp, err := http.DefaultClient.Do(fwdRequest)
	if err == nil {
		for k, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(k, value)
			}
		}
		if *traceEnabled {
			rw.Header().Set("lb-from", dst)
		}
		log.Println("fwd", resp.StatusCode, resp.Request.URL)
		rw.WriteHeader(resp.StatusCode)
		defer resp.Body.Close()
		_, err := io.Copy(rw, resp.Body)
		if err != nil {
			log.Printf("Failed to write response: %s", err)
		}
		return nil
	} else {
		log.Printf("Failed to get response from %s: %s", dst, err)
		rw.WriteHeader(http.StatusServiceUnavailable)
		return err
	}
}

func main() {
	flag.Parse()

	// TODO: Використовуйте дані про стан сервреа, щоб підтримувати список тих серверів, яким можна відправляти ззапит.
	for _, server := range serversPool {
		serverURL := server.URL
		go func() {
			for range time.Tick(10 * time.Second) {
				isAlive := health(serverURL)
				log.Println(serverURL, isAlive)
				server.IsAlive = isAlive
			}
		}()
	}

	frontend := httptools.CreateServer(*port, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		forwardServer, err := getForwardServer(serversPool, r)
		if err != nil {
			log.Printf("Failed to redirect request: %s", err)
		}
		forward(forwardServer.URL, rw, r)
	}))

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *traceEnabled)
	frontend.Start()
	signal.WaitForTerminationSignal()
}

func hash(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32())
}
