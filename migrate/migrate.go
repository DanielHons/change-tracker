package migrate

import (
	"database/sql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"log"
)

type MigrationConfig postgres.Config

type Config struct {
	MigrationConfig            MigrationConfig
	ConnectionStr              string
	MigrationFilesAbsolutePath string
}

func MigrateToNewest(config *Config) {
	db, err := sql.Open("postgres", config.ConnectionStr)
	defer db.Close()
	if err != nil {
		log.Fatal("Could connect to database", err)
	}

	driver, err := postgres.WithInstance(db, (*postgres.Config)(&config.MigrationConfig))
	if err != nil {
		log.Fatal("Error using database", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://"+config.MigrationFilesAbsolutePath,
		config.ConnectionStr, driver)
	if err != nil {
		log.Fatal("Could not initialize schema migrations", err)
	}
	err = m.Up()
	if err != nil && err.Error() != "no change" {
		log.Fatal("Could not apply schema migrations", err)
	}
}
