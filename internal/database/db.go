package database

import (
	"fmt"
	"strings"

	"github.com/go-pg/pg/v9"
)

func callStoredProcedure(db *pg.DB, name string, args ...interface{}) (res pg.Result, err error) {
	return db.Exec(constructQuery("call", name, len(args)), args...)
}

func callFunction(db *pg.DB, name string, model interface{}, args ...interface{}) (res pg.Result, err error) {
	return db.QueryOne(model, constructQuery("SELECT", name, len(args)), args...)
}

func constructQuery(verb string, callableName string, nArgs int) string {
	var b strings.Builder
	query := fmt.Sprintf("%s %s(", verb, callableName)
	b.WriteString(query)
	for i := 1; i <= nArgs; i++ {
		b.WriteString("?")
		if i != nArgs {
			b.WriteString(",")
		}
	}
	b.WriteString(");")
	return b.String()
}

// Connect connects to the database and checks the connection.
func Connect(addr string, user string, password string, dbName string) (*pg.DB, error) {
	db := pg.Connect(&pg.Options{
		User:     user,
		Password: password,
		Database: dbName,
		Addr: addr,
	})

	_, err := db.Exec("SELECT 1")
	if err != nil {
		return nil, err
	}

	return db, nil
}
