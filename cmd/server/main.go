package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// --- SOLARK MODELS ---

const (
	StatusAdPosted        = "AD_POSTED"
	StatusBuyerInterested = "BUYER_INTERESTED"
	StatusAmountAgreed    = "AMOUNT_AGREED"
	StatusChallanResolved = "CHALLANS_RESOLVED"
	StatusNOCIssued       = "NOC_ISSUED"
	StatusClosed          = "CLOSED"
)

type Vehicle struct {
	RegistrationNum string `json:"registration_num"`
	Model           string `json:"model"`
	Make            string `json:"make"`
}

type Ticket struct {
	ID           string    `json:"id"`
	TicketNumber string    `json:"ticket_number"`
	Vehicle      *Vehicle  `json:"vehicle"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// --- SERVER CORE ---

type Server struct {
	router  *mux.Router
	tickets map[string]*Ticket
	mu      sync.RWMutex
}

func NewServer() *Server {
	return &Server{
		router:  mux.NewRouter(),
		tickets: make(map[string]*Ticket),
	}
}

func (s *Server) routes() {
	// Serve the SOLARK Frontend
	s.router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/index.html")
	})

	// API Endpoints
	api := s.router.PathPrefix("/api/v2").Subrouter()
	api.HandleFunc("/tickets", s.createTicket).Methods("POST")
	api.HandleFunc("/tickets/{tn}", s.getTicket).Methods("GET")
	api.HandleFunc("/tickets/{tn}/noc", s.generateNOC).Methods("POST")
	api.HandleFunc("/tickets/{tn}/close", s.closeTicket).Methods("POST")
}

func (s *Server) createTicket(w http.ResponseWriter, r *http.Request) {
	var v Vehicle
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		http.Error(w, "Invalid Request Body", 400)
		return
	}
	
	reg := strings.ToUpper(v.RegistrationNum)
	ticket := &Ticket{
		ID:           uuid.New().String(),
		TicketNumber: reg,
		Vehicle:      &v,
		Status:       StatusAdPosted,
		CreatedAt:    time.Now(),
	}

	s.mu.Lock()
	s.tickets[reg] = ticket
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

func (s *Server) getTicket(w http.ResponseWriter, r *http.Request) {
	tn := strings.ToUpper(mux.Vars(r)["tn"])
	s.mu.RLock()
	t, ok := s.tickets[tn]
	s.mu.RUnlock()
	
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		fmt.Fprintf(w, `{"error":"Ticket not found"}`)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

func (s *Server) generateNOC(w http.ResponseWriter, r *http.Request) {
	tn := strings.ToUpper(mux.Vars(r)["tn"])
	s.mu.Lock()
	if t, ok := s.tickets[tn]; ok {
		t.Status = StatusNOCIssued
	}
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"NOC_GENERATED", "target":"%s"}`, tn)
}

func (s *Server) closeTicket(w http.ResponseWriter, r *http.Request) {
	tn := strings.ToUpper(mux.Vars(r)["tn"])
	s.mu.Lock()
	if t, ok := s.tickets[tn]; ok {
		t.Status = StatusClosed
	}
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"SUCCESS", "message":"Contract Closed"}`)
}

func main() {
	srv := NewServer()
	srv.routes()
	
	// Enable CORS so the Frontend can talk to the Backend
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	})
	
	fmt.Println("-------------------------------------------")
	fmt.Println("SOLARK PROTOCOL (Solark-Labs) is Live")
	fmt.Println("RTO Gateway active on http://localhost:8080")
	fmt.Println("-------------------------------------------")
	
	log.Fatal(http.ListenAndServe(":8080", c.Handler(srv.router)))
}
