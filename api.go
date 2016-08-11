package zonewatcher

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type Server struct {
	Addr      string
	DbFile    string
	db        *bolt.DB
	Observers []Observer
}

type ObserverSummary struct {
	Total    int `json:"total"`
	Finished int `json:"finished"`
	Canceled int `json:"canceled"`
	Waiting  int `json:"waiting"`
}

func (s *Server) Start() {
	if s.Addr == "" {
		s.Addr = ":9000"
	}

	if s.DbFile == "" {
		s.DbFile = fmt.Sprintf("zonewatcher-%d.db", timestamp())
	}

	db, err := bolt.Open(s.DbFile, 0600, &bolt.Options{Timeout: 1 * time.Second})

	if err != nil {
		panic(err)
	}

	s.db = db

	r := mux.NewRouter()
	r.HandleFunc("/observe", s.CreateObserverHandler).Methods("POST")
	r.HandleFunc("/status", s.GetObserverStatusHandler).Methods("GET")

	loggedRouter := handlers.LoggingHandler(os.Stdout, r)

	http.ListenAndServe(s.Addr, loggedRouter)
}

func (s *Server) CreateObserverHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("New observer")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	var o Observer
	if err := json.Unmarshal(body, &o); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	go o.Watch(s.db)
	s.Observers = append(s.Observers, o)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(o); err != nil {
		panic(err)
	}
}

func (s *Server) GetObserverStatusHandler(w http.ResponseWriter, r *http.Request) {
	summary := ObserverSummary{Total: len(s.Observers)}
	for _, o := range s.Observers {
		if o.Status == STATUS_FINISHED {
			summary.Finished++
		}
		if o.Status == STATUS_CANCELED {
			summary.Canceled++
		}

		if o.Status == STATUS_WAITING {
			summary.Waiting++
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(summary); err != nil {
		panic(err)
	}
}
