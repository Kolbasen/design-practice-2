package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/Kolbasen/design-practice-2/cmd/datastore"
	"github.com/Kolbasen/design-practice-2/httptools"
	"github.com/Kolbasen/design-practice-2/signal"
)


var port = flag.Int("p", 8000, "port")
var path = flag.String("d", ".db", "db path")
var segmentSize = flag.Int("s", 10*MB, "segment size")
const teamName = "kfcteam"
const MB = 1024 * 1024

type Response struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RequestPayload struct {
	Value string `json:"value"`
}

func main() {
	flag.Parse()

	err := os.MkdirAll(*path, os.ModePerm)
	if err != nil {
		log.Fatalf(err)
		return
	}

	db, err := datastore.NewDb(*path, int64(*segmentSize))
	if err != nil {
		log.Fatalf(err)
		return
	}

	defer db.Close()
	_ = db.Put("key", teamName)

	router := mux.NewRouter()

	router.HandleFunc("/db/{key}", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("content-type", "application/json")

		vars := mux.Vars(r)
		key := vars["key"]

		var body RequestPayload
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		err = db.Put(key, body.Value)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		rw.WriteHeader(http.StatusOK)
	}).Methods("POST")

	router.HandleFunc("/db/{key}", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("content-type", "application/json")

		vars := mux.Vars(r)
		key := vars["key"]
		value, err := db.Get(key)

		if err != nil {
			rw.WriteHeader(http.StatusNotFound)
			return
		} 

		rw.WriteHeader(http.StatusOK)

		res := Response{key, value}
		err := json.NewEncoder(rw).Encode(&res)

		if err != nil {
			log.Printf(err)
		}
	}).Methods("GET")

	h := new(http.ServeMux)
	h.Handle("/", router)

	server := httptools.CreateServer(*port, h)
	server.Start()
	signal.WaitForTerminationSignal()
}