package main

import (
	"database/sql"
	"encoding/json"
	"github.com/DanielHons/change-tracker/migrate"
	"github.com/DanielHons/go-jwt-exchange/jwt_exchange"
	"log"
	"net/http"
	"os"
	"time"
)

const envVarTokenHeaderIn = "TOKEN_HEADER_IN"
const envVarApiToken = "NOTIFIER_API_TOKEN"
const envVarDbConnStr = "DATABASE_CONNECTION_STRING"
const envVarMigrationFilePathKey = "MIGRATION_FILES"

type Validatable interface {
	IsValid() bool
}

type StageNotification struct {
	Component string `json:"component"`
	Stage     string `json:"stage"`
	Version   string `json:"version"`
	Sha       string `json:"sha"`
}

type MultiStageNotification struct {
	Component string   `json:"component"`
	Stages    []string `json:"stages"`
	Version   string   `json:"version"`
	Sha       string   `json:"sha"`
}

func (n StageNotification) IsValid() bool {
	return len(n.Component) > 0 && len(n.Stage) > 0 && len(n.Sha) > 0
}

func (n MultiStageNotification) IsValid() bool {
	return len(n.Component) > 0 && len(n.Stages) > 0 && len(n.Sha) > 0
}

type DiffRequest struct {
	Base    string
	Compare string
}

func ExtractDiffRequest(r *http.Request) DiffRequest {
	query := r.URL.Query()
	return DiffRequest{
		Base:    query.Get("base"),
		Compare: query.Get("compare"),
	}
}

func (dr DiffRequest) IsValid() bool {
	return dr.Base != "" && dr.Compare != ""
}

func main() {
	// Migrate schema
	migrate.MigrateToNewest(&migrate.Config{
		MigrationConfig: migrate.MigrationConfig{
			MigrationsTable:  "ct_schema_migration",
			StatementTimeout: 0,
		},
		MigrationFilesAbsolutePath: os.Getenv(envVarMigrationFilePathKey),
		ConnectionStr:              os.Getenv(envVarDbConnStr),
	})

	// Run proxy
	jwksCache := jwt_exchange.JwksCache{
		JwksUrl:             os.Getenv("JWKS_URL"),
		JwksRefreshInterval: 24 * time.Hour,
	}
	jwksCache.ReloadJwks()

	http.HandleFunc("/overview/api/diff", secured(overviewApi(), &jwksCache))
	http.HandleFunc("/notify/single", receiveSingleStageNotification())
	http.HandleFunc("/notify/multi", receiveMultiStageNotification())
	log.Fatal(http.ListenAndServe(os.Getenv("BIND_ADDRESS"), nil))
}

func overviewApi() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
		} else {
			dr := ExtractDiffRequest(r)
			if !dr.IsValid() {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad Request"))
			} else {
				getDiffResponse(w, dr)
			}

		}
	}
}

func getDiffResponse(w http.ResponseWriter, dr DiffRequest) {
	diff, err := getDiff(dr.Base, dr.Compare)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Internal Error"))
	} else {
		w.Write([]byte(diff))
	}
}

func secured(handler func(w http.ResponseWriter, r *http.Request), validator jwt_exchange.TokenValidator) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get(os.Getenv(envVarTokenHeaderIn))
		validate, err := validator.Validate(token)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			err := validate.Valid()
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
			} else {
				handler(w, r)
			}

		}
	}
}

func receiveSingleStageNotification() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		receiveNotification(w, r, processSingleStageNotification)
	}
}

func receiveMultiStageNotification() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		receiveNotification(w, r, processMultiStageNotification)
	}
}

func processSingleStageNotification(w http.ResponseWriter, r *http.Request) {
	var p StageNotification
	// Try to decode the request body into the struct. If there is an error,
	// respond to the client with the error message and a 400 status code.
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if p.IsValid() {
		insert(p)
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid Argument"))
	}
}

func processMultiStageNotification(w http.ResponseWriter, r *http.Request) {
	var p MultiStageNotification
	// Try to decode the request body into the struct. If there is an error,
	// respond to the client with the error message and a 400 status code.
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if p.IsValid() {
		for _, n := range expand(p) {
			insert(n)

		}
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid Argument"))
	}
}

// Notification calls are always POST with the static token required
func receiveNotification(w http.ResponseWriter, r *http.Request, notificationHandler func(w http.ResponseWriter, r *http.Request)) {
	if r.Header.Get(os.Getenv(envVarTokenHeaderIn)) == os.Getenv(envVarApiToken) {
		if r.Method == http.MethodPost {
			notificationHandler(w, r)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}
}

func expand(m MultiStageNotification) []StageNotification {
	var notifications []StageNotification
	for _, stage := range m.Stages {
		notifications = append(notifications, StageNotification{
			Component: m.Component,
			Stage:     stage,
			Version:   m.Version,
			Sha:       m.Sha,
		})
	}
	return notifications
}

func insert(n StageNotification) {
	sqlStatement := `
INSERT INTO stages_reached_latest_versions (component_name, stage_name, version, git_sha)
VALUES ($1, $2, $3, $4)`

	db, err := sql.Open("postgres", os.Getenv(envVarDbConnStr))
	defer db.Close()
	if err != nil {
		log.Fatal("Could connect to database", err)
	}
	row := db.QueryRow(sqlStatement, n.Component, n.Stage, n.Version, n.Sha)
	print(row)
}

func getDiff(base string, compare string) (string, error) {
	sqlStatement := "select json_agg(sub) from (select * from diff($1,$2)) sub;"

	db, err := sql.Open("postgres", os.Getenv(envVarDbConnStr))
	defer db.Close()
	if err != nil {
		log.Fatal("Could connect to database", err)
	}
	row := db.QueryRow(sqlStatement, base, compare)
	var json string
	err = row.Scan(&json)
	if err != nil {
		return "[]", nil
	}
	return json, err

}
