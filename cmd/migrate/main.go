package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"

	"gopkg.in/telebot.v4/internal/config"
)

func main() {
	var dir string
	flag.StringVar(&dir, "dir", "db/migrations", "path to migration directory")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatalf("usage: go run ./cmd/migrate -- <up|down|status|reset|version>")
	}

	command := flag.Arg(0)
	args := flag.Args()[1:]

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	dsn := cfg.DatabaseURL
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		log.Fatal("DATABASE_URL (or POSTGRES_* vars) is required")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal(err)
	}

	if err := goose.Run(command, db, dir, args...); err != nil {
		log.Fatal(err)
	}

	fmt.Println("migration command completed:", command)
}
