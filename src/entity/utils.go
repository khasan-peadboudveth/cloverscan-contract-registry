package entity

import (
	"fmt"
	"github.com/clover-network/cloverscan-contract-registry/src/config"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jinzhu/inflection"
	log "github.com/sirupsen/logrus"
	"strings"
)

func NewConnection(config *config.Config) *pg.DB {
	/* disable plural names */
	orm.SetTableNameInflector(inflection.Singular)
	/* create connection */
	db := pg.Connect(&pg.Options{
		ApplicationName: "sidecar-manager",
		Addr:            config.DatabaseAddress,
		Database:        config.DatabaseDb,
		User:            config.DatabaseUser,
		Password:        config.DatabasePassword,
	})
	_, err := db.Exec("SELECT 1")
	if err != nil {
		log.Fatalf("unable to check PostgreSQL connection", err)
	}
	return db
}

func ApplyDatabaseMigrations(config *config.Config) {
	log.Infof("going to apply database migrations")
	m, err := migrate.New("file://src/entity/migrations", fmt.Sprintf("postgresql://%s/%s?user=%s&password=%s&sslmode=disable", config.DatabaseAddress, config.DatabaseDb, config.DatabaseUser, config.DatabasePassword))
	if err != nil {
		log.Fatalf("can't init migration", err)
	}
	err = m.Up()
	if err != nil && strings.Compare(err.Error(), "no change") != 0 {
		log.Fatalf("unable to apply migrations", err)
	}
	_, _ = m.Close()
	log.Infof("all database migrations has been successfully applied")
}
