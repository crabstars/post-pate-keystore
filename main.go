package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"

	dbrepo "github.com/crabstars/post-pate-keystore/db_repo"
)

const datbase = "./keystore.db"

var db *sql.DB

func main() {
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

	// TODO: check for api key
	mux := http.NewServeMux()
	mux.HandleFunc("GET /key/{userId}", GetKey)

	if err = http.ListenAndServe("localhost:8080", mux); err != nil {
		log.Fatal(err)
	}

	// err = dbrepo.InsertUserAndKey(db, "e2UserDaiel", "bla", "iv key")
	// if sqliteErr, ok := err.(sqlite3.Error); ok {
	// 	if sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
	// 		log.Fatal("User already exists")
	// 	}
	// }
	// if err != nil {
	// 	log.Fatal(err)
	// }
}

func GetKey(w http.ResponseWriter, r *http.Request) {
	userId := r.PathValue("userId")

	exists, err := dbrepo.UserExists(db, userId)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("User exists: ", exists)
	fmt.Fprintf(w, "User exists: %t", exists)
}
