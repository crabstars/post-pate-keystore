package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"

	dbrepo "github.com/crabstars/post-pate-keystore/db_repo"
)

const datbase = "./keystore.db"

var (
	apiKey string
	db     *sql.DB
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

	mux := http.NewServeMux()

	mux.HandleFunc("GET /user/{userId}/exists", AuthMiddleware(GetUserExists))
	mux.HandleFunc("GET /user/{userId}/entry", AuthMiddleware(GetUserEntry))
	mux.HandleFunc("POST /user/{userId}/entry", AuthMiddleware(AddUserEntry))
	mux.HandleFunc("DELETE /user/{userId}/entry", AuthMiddleware(DeleteUserEntry))

	if err = http.ListenAndServe("localhost:8081", mux); err != nil {
		log.Fatal(err)
	}
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
			http.Error(w, "Wrong api key", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
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
	fmt.Fprintf(w, "User exists: %t", exists)
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
	count, err := dbrepo.DelteUser(db, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if count == 0 {
		http.NotFound(w, r)
		return
	}
}
