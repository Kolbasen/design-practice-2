package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Kolbasen/design-practice-2/httptools"
	"github.com/Kolbasen/design-practice-2/signal"
)

var port = flag.Int("port", 8080, "server port")

const dbUrl = "http://db:8000"
const confResponseDelaySec = "CONF_RESPONSE_DELAY_SEC"
const confHealthFailure = "CONF_HEALTH_FAILURE"

func main() {
	url := fmt.Sprintf("%s/db/teamkfc", dbUrl)
	body := fmt.Sprintf(`{"value":"%s"}`, time.Now().Format("01-01-2001"))
	body += "\n"
	req, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}

	h := new(http.ServeMux)

	h.HandleFunc("/health", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("content-type", "text/plain")
		if failConfig := os.Getenv(confHealthFailure); failConfig == "true" {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte("FAILURE"))
		} else {
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte("OK"))
		}
	})

	report := make(Report)

	h.HandleFunc("/api/v1/some-data", func(rw http.ResponseWriter, r *http.Request) {
		url := fmt.Sprintf("%s/db/%s", dbUrl, r.FormValue("key"))
		res, err := http.Get(url)
		if err != nil {
			rw.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		rw.Header().Set("content-type", "application/json")
		rw.WriteHeader(res.StatusCode)
		defer res.Body.Close()
		_, err = io.Copy(rw, res.Body)
		if err != nil {
			log.Printf(err)
		}
	})

	h.HandleFunc("/api/v1/another-data", func(rw http.ResponseWriter, r *http.Request) {
		respDelayString := os.Getenv(confResponseDelaySec)
		if delaySec, parseErr := strconv.Atoi(respDelayString); parseErr == nil && delaySec > 0 && delaySec < 300 {
			time.Sleep(time.Duration(delaySec) * time.Second)
		}

		report.Process(r)

		rw.Header().Set("content-type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(rw).Encode([]string{
			"1", "2",
		})
	})

	h.HandleFunc("/api/v1/last-data", func(rw http.ResponseWriter, r *http.Request) {
		respDelayString := os.Getenv(confResponseDelaySec)
		if delaySec, parseErr := strconv.Atoi(respDelayString); parseErr == nil && delaySec > 0 && delaySec < 300 {
			time.Sleep(time.Duration(delaySec) * time.Second)
		}

		report.Process(r)

		rw.Header().Set("content-type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(rw).Encode([]string{
			"1", "2",
		})
	})

	h.Handle("/report", report)

	server := httptools.CreateServer(*port, h)
	server.Start()
	signal.WaitForTerminationSignal()
}
