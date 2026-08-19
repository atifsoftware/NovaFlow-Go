// Command novaflow-cli mirrors NovaFlow PHP's cli.php: system health
// checks, route listing, database migrations, and controller/model
// scaffolding — run with `go run ./cli <command>`.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"novaflow/app/middleware"
	"novaflow/config"
	"novaflow/core"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	switch os.Args[1] {
	case "--health", "health":
		cmdHealth()
	case "--routes", "routes":
		cmdRoutes()
	case "migrate":
		cmdMigrate()
	case "migrate:rollback", "migrate-rollback", "rollback":
		cmdMigrateRollback()
	case "make:migration":
		requireArg(2, "migration name")
		cmdMakeMigration(os.Args[2])
	case "make:controller":
		requireArg(2, "controller name")
		cmdMakeController(os.Args[2])
	case "make:model":
		requireArg(2, "model name")
		cmdMakeModel(os.Args[2])
	default:
		printHelp()
	}
}

func requireArg(idx int, label string) {
	if len(os.Args) <= idx {
		fmt.Printf("Missing %s. Usage: novaflow-cli %s <Name>\n", label, os.Args[1])
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`NovaFlow CLI

Usage:
  go run ./cli --health              System health check (env + database)
  go run ./cli --routes              List all registered routes
  go run ./cli migrate               Run pending SQL migrations in database/migrations
  go run ./cli migrate:rollback      Rollback the last batch of migrations
  go run ./cli make:migration Name   Create a new migration file with timestamp
  go run ./cli make:controller Name  Scaffold app/controllers/name_controller.go
  go run ./cli make:model Name       Scaffold app/models/name.go`)
}

func cmdHealth() {
	app := core.NewApp(".env", "")
	fmt.Println("NovaFlow Health Check")
	fmt.Println("---------------------")
	fmt.Printf("APP_ENV:      %s\n", app.Config.Get("APP_ENV", "local"))
	fmt.Printf("APP_PORT:     %s\n", app.Config.Get("APP_PORT", "8080"))
	if app.DB != nil {
		if err := app.DB.Conn.Ping(); err != nil {
			fmt.Println("Database:     FAILED -", err)
			os.Exit(1)
		}
		fmt.Println("Database:     OK")
	} else {
		fmt.Println("Database:     not configured (set DB_HOST in .env)")
	}
	fmt.Println("Status:       healthy")
}

func cmdRoutes() {
	app := core.NewApp(".env", "app/views")
	config.RegisterRoutes(app, middleware.NewKernel(app))
	routes := app.Router.Routes()
	sort.Slice(routes, func(i, j int) bool { return routes[i].Path < routes[j].Path })
	fmt.Printf("%-8s %s\n", "METHOD", "PATH")
	fmt.Println(strings.Repeat("-", 40))
	for _, rt := range routes {
		fmt.Printf("%-8s %s\n", rt.Method, rt.Path)
	}
}

func cmdMigrate() {
	app := core.NewApp(".env", "")
	if app.DB == nil {
		fmt.Println("No database configured (set DB_HOST in .env)")
		os.Exit(1)
	}

	// Create table migrations. Add batch column to support rollbacks.
	if _, err := app.DB.Exec(`CREATE TABLE IF NOT EXISTS migrations (
		id INT AUTO_INCREMENT PRIMARY KEY,
		migration VARCHAR(255) NOT NULL UNIQUE,
		batch INT NOT NULL DEFAULT 1,
		run_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		fmt.Println("Could not create migrations table:", err)
		os.Exit(1)
	}

	// Ensure batch column exists for older tables
	_, _ = app.DB.Exec("ALTER TABLE migrations ADD COLUMN batch INT NOT NULL DEFAULT 1")

	applied := map[string]int{}
	rows, err := app.DB.Raw("SELECT migration, batch FROM migrations")
	if err != nil {
		fmt.Println("Could not read migrations table:", err)
		os.Exit(1)
	}
	for _, row := range rows {
		if name, ok := row["migration"].(string); ok {
			applied[name] = int(core.ToInt64(row["batch"]))
		}
	}

	files, _ := filepath.Glob("database/migrations/*.sql")
	sort.Strings(files)

	// Determine next batch number
	nextBatch := 1
	batchRow, err := app.DB.Table("migrations").Select("MAX(batch) as max_batch").First()
	if err == nil && batchRow != nil && batchRow["max_batch"] != nil {
		nextBatch = int(core.ToInt64(batchRow["max_batch"])) + 1
	}

	ran := 0
	for _, f := range files {
		name := filepath.Base(f)
		if _, ok := applied[name]; ok {
			continue
		}
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			fmt.Println("Could not read", f, ":", err)
			os.Exit(1)
		}

		upSQL, _ := parseMigrationSQL(string(sqlBytes))
		if err := execSQLStatements(app.DB, upSQL); err != nil {
			fmt.Printf("Migration %s failed: %v\n", name, err)
			os.Exit(1)
		}

		if _, err := app.DB.Table("migrations").Insert(map[string]interface{}{
			"migration": name,
			"batch":     nextBatch,
		}); err != nil {
			fmt.Println("Could not record migration:", err)
			os.Exit(1)
		}
		fmt.Println("Migrated:", name)
		ran++
	}
	if ran == 0 {
		fmt.Println("Nothing to migrate.")
	}
}

func cmdMigrateRollback() {
	app := core.NewApp(".env", "")
	if app.DB == nil {
		fmt.Println("No database configured (set DB_HOST in .env)")
		os.Exit(1)
	}

	// Check max batch
	batchRow, err := app.DB.Table("migrations").Select("MAX(batch) as max_batch").First()
	if err != nil || batchRow == nil || batchRow["max_batch"] == nil {
		fmt.Println("No migrations found to rollback.")
		return
	}
	maxBatch := core.ToInt64(batchRow["max_batch"])
	if maxBatch <= 0 {
		fmt.Println("No migrations found to rollback.")
		return
	}

	// Fetch migrations in reverse order
	rows, err := app.DB.Table("migrations").Where("batch", "=", maxBatch).OrderBy("id", "DESC").Get()
	if err != nil || len(rows) == 0 {
		fmt.Println("No migrations found for batch", maxBatch)
		return
	}

	fmt.Printf("Rolling back batch %d...\n", maxBatch)
	for _, row := range rows {
		name := row["migration"].(string)
		f := filepath.Join("database", "migrations", name)
		if _, err := os.Stat(f); os.IsNotExist(err) {
			fmt.Printf("Warning: migration file %s not found, skipping SQL execution but deleting db record.\n", f)
		} else {
			sqlBytes, err := os.ReadFile(f)
			if err != nil {
				fmt.Printf("Error reading migration file %s: %v\n", f, err)
				os.Exit(1)
			}
			_, downSQL := parseMigrationSQL(string(sqlBytes))
			if strings.TrimSpace(downSQL) != "" {
				fmt.Println("Rolling back:", name)
				if err := execSQLStatements(app.DB, downSQL); err != nil {
					fmt.Printf("Rollback failed for %s: %v\n", name, err)
					os.Exit(1)
				}
			} else {
				fmt.Printf("No DOWN section found in %s, skipped SQL execution.\n", name)
			}
		}

		if _, err := app.DB.Table("migrations").Where("migration", "=", name).Delete(); err != nil {
			fmt.Printf("Could not delete migration record %s: %v\n", name, err)
			os.Exit(1)
		}
		fmt.Println("Rolled back successfully:", name)
	}
}

func cmdMakeMigration(name string) {
	timestamp := time.Now().Format("20060102150405")
	slug := toSnake(name)
	filename := fmt.Sprintf("%s_%s.sql", timestamp, slug)
	path := filepath.Join("database", "migrations", filename)

	if err := os.MkdirAll("database/migrations", 0755); err != nil {
		fmt.Println("Could not create migrations directory:", err)
		os.Exit(1)
	}

	tpl := `-- UP
-- Write your UP migration SQL statements here


-- DOWN
-- Write your DOWN migration SQL statements here
`
	if err := os.WriteFile(path, []byte(tpl), 0644); err != nil {
		fmt.Println("Could not write migration file:", err)
		os.Exit(1)
	}
	fmt.Println("Created:", path)
}

func cmdMakeController(name string) {
	base := titleCase(strings.ToLower(name))
	fileName := toSnake(name) + "_controller.go"
	path := filepath.Join("app", "controllers", fileName)
	if _, err := os.Stat(path); err == nil {
		fmt.Println("Already exists:", path)
		os.Exit(1)
	}

	tpl := `package controllers

import "novaflow/core"

type ` + base + `Controller struct{}

func New` + base + `Controller() *` + base + `Controller { return &` + base + `Controller{} }

func (ctl *` + base + `Controller) Index(c *core.Context) {
	c.JSONSuccess(map[string]string{"message": "` + base + ` index"})
}
`
	if err := os.WriteFile(path, []byte(tpl), 0o644); err != nil {
		fmt.Println("Could not write file:", err)
		os.Exit(1)
	}
	fmt.Println("Created:", path)
}

func cmdMakeModel(name string) {
	base := titleCase(strings.ToLower(name))
	fileName := toSnake(name) + ".go"
	path := filepath.Join("app", "models", fileName)
	if _, err := os.Stat(path); err == nil {
		fmt.Println("Already exists:", path)
		os.Exit(1)
	}

	table := toSnake(name) + "s"
	tpl := `package models

type ` + base + ` struct {
	ID int64 ` + "`db:\"id\"`" + `
}

func (` + base + `) TableName() string { return "` + table + `" }
`
	if err := os.WriteFile(path, []byte(tpl), 0o644); err != nil {
		fmt.Println("Could not write file:", err)
		os.Exit(1)
	}
	fmt.Println("Created:", path)
}

// titleCase capitalizes the first letter of each word in s.
// Replaces deprecated strings.Title() which mishandles Unicode.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, "")
}

func toSnake(s string) string {
	var out strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				out.WriteByte('_')
			}
			out.WriteRune(r - 'A' + 'a')
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func parseMigrationSQL(sqlContent string) (upSQL string, downSQL string) {
	lines := strings.Split(sqlContent, "\n")
	var upLines, downLines []string
	currentMode := "up"

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		upperTrimmed := strings.ToUpper(trimmed)
		if strings.HasPrefix(upperTrimmed, "--") {
			commentContent := strings.TrimSpace(strings.TrimPrefix(trimmed, "--"))
			commentUpper := strings.ToUpper(commentContent)
			if commentUpper == "UP" || commentUpper == "+MIGRATE UP" {
				currentMode = "up"
				continue
			} else if commentUpper == "DOWN" || commentUpper == "+MIGRATE DOWN" {
				currentMode = "down"
				continue
			}
		}
		if currentMode == "up" {
			upLines = append(upLines, line)
		} else {
			downLines = append(downLines, line)
		}
	}

	upSQL = strings.Join(upLines, "\n")
	downSQL = strings.Join(downLines, "\n")
	return
}

func execSQLStatements(db *core.DB, sqlContent string) error {
	for _, stmt := range strings.Split(sqlContent, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
