package main

import (
	"fmt"
	"flag"
	"database/sql"
	_ "github.com/lib/pq"
	"os"
	"github.com/dm03514/anomaly-detection-ip/ipsets"
	"context"
	"github.com/lib/pq"
)

func main() {
	fmt.Println("HELLLO")
	var dbConnectionString = flag.String("db-connection-string", "", "")
	var netsetFile = flag.String("netset-file", "", "")
	var netsetName = flag.String("netset-name", "", "")
	flag.Parse()

	db, err := sql.Open("postgres", *dbConnectionString)
	if err != nil {
		panic(err)
	}

	if err = db.Ping(); err != nil {
		panic(err)
	}

	f, err := os.Open(*netsetFile)
	if err != nil {
		panic(err)
	}

	netset, err := ipsets.NewNetset(f)
	if err != nil {
		panic(err)
	}

	m, err := netset.Metadata()
	if err != nil {
		panic(err)
	}

	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		panic(err)
	}
	data, err := m.JSON()
	if err != nil {
		panic(err)
	}
	if _, execErr := tx.Exec(
			"INSERT INTO provider (pname, metadata) VALUES ($1, $2) ON CONFLICT (pname) DO UPDATE SET metadata=$2",
			*netsetName, data); execErr != nil {
		_ = tx.Rollback()
		panic(execErr)
	}

	if _, execErr := tx.Exec("DELETE FROM ipsets WHERE provider=$1", *netsetName); execErr != nil {
		_ = tx.Rollback()
		panic(execErr)
	}

	// loop through each of the rows and add them, we could make this a generator
	// in order to bound memory in the future
	stmt, err := tx.Prepare(pq.CopyIn("ipsets", "address", "provider"))
	for _, cidr := range netset.CIDRS() {
		if _, err = stmt.Exec(cidr, *netsetName); err != nil {
			panic(err)
		}
	}

	if _, err = stmt.Exec(); err != nil {
		panic(err)
	}

	if err = stmt.Close(); err != nil {
		panic(err)
	}

	if err := tx.Commit(); err != nil {
		panic(err)
	}


}
