package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	dbrepo "github.com/crabstars/post-pate-keystore/db_repo"
)

const datbase = "./keystore.db"

var (
	apiKey string
	db     *sql.DB

	// Prometheus metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)
	dbConnectionsOpen = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_open",
			Help: "Number of open database connections",
		},
	)
	healthStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "app_health_status",
			Help: "Current health status of the application",
		},
		[]string{"status"},
	)
)

func main() {
	godotenv.Load(".env")
	apiKey = os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatal("API_KEY not set")
	}

	_, databaseExistsErr := os.Stat(datbase)

	var err error
	db, err = sql.Open("sqlite3", "./keystore.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if os.IsNotExist(databaseExistsErr) {
		dbrepo.CreateInitialSchema(db)
	}
	db.SetMaxOpenConns(10)
	go monitorDBStats(db)
	go monitorHealth()

	mux := http.NewServeMux()

	mux.Handle("/metrics", AuthMiddleware(promhttp.Handler().ServeHTTP))
	mux.HandleFunc("GET /user/{userId}/exists", instrumentHandler("GET", "UserExists", AuthMiddleware(GetUserExists)))
	mux.HandleFunc("GET /user/{userId}/entry", instrumentHandler("GET", "UserEntry", AuthMiddleware(GetUserEntry)))
	mux.HandleFunc("POST /user/{userId}/entry", instrumentHandler("POST", "UserEntry", AuthMiddleware(AddUserEntry)))
	mux.HandleFunc("DELETE /user/{userId}/entry", instrumentHandler("DELETE", "UserEntry", AuthMiddleware(DeleteUserEntry)))
	if err = http.ListenAndServe("localhost:8081", mux); err != nil {
		log.Fatal(err)
	}
}
func monitorDBStats(db *sql.DB) {
	for {
		stats := db.Stats()
		dbConnectionsOpen.Set(float64(stats.OpenConnections))
		time.Sleep(5 * time.Second)
	}
}

func monitorHealth() {
	for {
		HealthCheck()
		time.Sleep(5 * time.Second)
	}
}
func instrumentHandler(method, endpoint string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(method, endpoint, fmt.Sprintf("%d", rw.statusCode)).Inc()
		httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func LogMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		fmt.Println("Start Log Middleware ")
		next.ServeHTTP(w, r)
		fmt.Println("Goodbye from Log Middleware: ")
	}
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		receivedKey := r.Header.Get("X-API-KEY")
		if receivedKey != apiKey {
			log.Println("Wrong api key")
			http.Error(w, "Wrong api key", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

type HealthStatus struct {
	Status      string `json:"status"`
	DBStatus    string `json:"db_status"`
	Connections int    `json:"db_connections"`
	Uptime      string `json:"uptime"`
	Version     string `json:"version"`
}

var startTime = time.Now()

func HealthCheck() {
	status := HealthStatus{
		Status:   "healthy",
		DBStatus: "up",
	}

	err := db.Ping()
	if err != nil {
		status.Status = "unhealthy"
		status.DBStatus = "down"
	}

	healthStatus.With(prometheus.Labels{"status": status.Status}).Set(1)
}

func GetUserEntry(w http.ResponseWriter, r *http.Request) {
	userId := r.PathValue("userId")

	user, err := dbrepo.GetUserEntry(db, userId)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func GetUserExists(w http.ResponseWriter, r *http.Request) {
	userId := r.PathValue("userId")

	exists, err := dbrepo.UserExists(db, userId)
	if err != nil {
		log.Fatal(err)
	}
	err = json.NewEncoder(w).Encode(exists)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func AddUserEntry(w http.ResponseWriter, r *http.Request) {
	userId := r.PathValue("userId")
	var user dbrepo.UserEntry
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if userId != user.UserId {
		http.Error(w, "User ID in URL does not match user ID in body", http.StatusBadRequest)
		return
	}

	exists, err := dbrepo.UserExists(db, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if exists {
		http.Error(w, "User already exists", http.StatusBadRequest)
		return
	}

	err = dbrepo.InsertUserAndKey(db, user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func DeleteUserEntry(w http.ResponseWriter, r *http.Request) {
	userId := r.PathValue("userId")
	count, err := dbrepo.DeleteUser(db, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if count == 0 {
		http.NotFound(w, r)
		return
	}
}
