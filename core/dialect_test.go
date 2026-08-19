package core

import (
	"testing"
)

func TestDialectPlaceholdersAndQuotes(t *testing.T) {
	my := &MySQLDialect{}
	if my.Placeholder(1) != "?" || my.Placeholder(2) != "?" {
		t.Errorf("MySQL placeholder expected '?', got %q", my.Placeholder(1))
	}
	if my.QuoteIdentifier("users") != "`users`" {
		t.Errorf("MySQL quote expected `users`, got %q", my.QuoteIdentifier("users"))
	}
	if !my.SupportsLastInsertID() {
		t.Error("MySQL expected to support LastInsertId")
	}

	pg := &PostgresDialect{}
	if pg.Placeholder(1) != "$1" || pg.Placeholder(2) != "$2" {
		t.Errorf("Postgres placeholder expected '$1', got %q", pg.Placeholder(1))
	}
	if pg.QuoteIdentifier("users") != `"users"` {
		t.Errorf("Postgres quote expected \"users\", got %q", pg.QuoteIdentifier("users"))
	}
	if pg.SupportsLastInsertID() {
		t.Error("Postgres expected to not support LastInsertId")
	}

	sq := &SQLiteDialect{}
	if sq.Placeholder(1) != "?" {
		t.Errorf("SQLite placeholder expected '?', got %q", sq.Placeholder(1))
	}
	if !sq.SupportsLastInsertID() {
		t.Error("SQLite expected to support LastInsertId")
	}
}

func TestPostgresQueryBuilderPlaceholderIndex(t *testing.T) {
	qb := &QueryBuilder{
		table:   "users",
		dialect: &PostgresDialect{},
	}
	qb.Where("status", "=", "active").Where("age", ">", 18)

	sqlStr, args := qb.buildSelect()
	expectedSQL := "SELECT * FROM users WHERE status = $1 AND age > $2"
	if sqlStr != expectedSQL {
		t.Errorf("expected %q, got %q", expectedSQL, sqlStr)
	}
	if len(args) != 2 || args[0] != "active" || args[1] != 18 {
		t.Errorf("unexpected args: %v", args)
	}
}


func TestSQLiteInMemoryIntegrationCRUD(t *testing.T) {
	db, err := OpenDBWithDialect(&SQLiteDialect{}, ":memory:")
	if err != nil {
		t.Fatalf("could not open in-memory sqlite db: %v", err)
	}
	defer db.Conn.Close()

	// 1. Create table
	_, err = db.Exec(`CREATE TABLE products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		price REAL NOT NULL,
		status TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatalf("could not create test table: %v", err)
	}

	// 2. Test Insert via QueryBuilder
	id1, err := db.Table("products").Insert(map[string]interface{}{
		"name":   "MacBook Air M2",
		"price":  1199.99,
		"status": "active",
	})
	if err != nil || id1 != 1 {
		t.Fatalf("expected insert id 1, got id=%d, err=%v", id1, err)
	}

	id2, err := db.Table("products").Insert(map[string]interface{}{
		"name":   "Magic Mouse",
		"price":  79.99,
		"status": "active",
	})
	if err != nil || id2 != 2 {
		t.Fatalf("expected insert id 2, got id=%d, err=%v", id2, err)
	}

	// 3. Test Count
	count, err := db.Table("products").Count()
	if err != nil || count != 2 {
		t.Fatalf("expected count 2, got %d, err=%v", count, err)
	}

	// 4. Test First
	row, err := db.Table("products").Where("id", "=", 1).First()
	if err != nil || row == nil {
		t.Fatalf("expected row 1, got nil, err=%v", err)
	}
	if row["name"] != "MacBook Air M2" {
		t.Errorf("expected name 'MacBook Air M2', got %v", row["name"])
	}

	// 5. Test Update
	affected, err := db.Table("products").Where("id", "=", 2).Update(map[string]interface{}{
		"price": 69.99,
	})
	if err != nil || affected != 1 {
		t.Fatalf("expected 1 row affected, got %d, err=%v", affected, err)
	}

	// 6. Test Delete
	deleted, err := db.Table("products").Where("id", "=", 2).Delete()
	if err != nil || deleted != 1 {
		t.Fatalf("expected 1 row deleted, got %d, err=%v", deleted, err)
	}

	newCount, _ := db.Table("products").Count()
	if newCount != 1 {
		t.Fatalf("expected count 1 after delete, got %d", newCount)
	}
}

type testProductEntity struct {
	ID    int64   `db:"id"`
	Name  string  `db:"name"`
	Price float64 `db:"price"`
}

func (testProductEntity) TableName() string { return "test_products" }

func TestGenericRepositoryWithSQLite(t *testing.T) {
	db, err := OpenDBWithDialect(&SQLiteDialect{}, ":memory:")
	if err != nil {
		t.Fatalf("could not open in-memory sqlite db: %v", err)
	}
	defer db.Conn.Close()

	_, err = db.Exec(`CREATE TABLE test_products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		price REAL NOT NULL
	)`)
	if err != nil {
		t.Fatalf("could not create test table: %v", err)
	}

	repo := NewRepository[testProductEntity](db)

	// 1. Create
	newProd := &testProductEntity{Name: "Golang Book", Price: 39.99}
	id, err := repo.Create(newProd)
	if err != nil || id != 1 {
		t.Fatalf("expected create id=1, got id=%d, err=%v", id, err)
	}

	// 2. Find
	found, err := repo.Find(1)
	if err != nil || found == nil {
		t.Fatalf("expected to find product 1, got nil, err=%v", err)
	}
	if found.Name != "Golang Book" || found.Price != 39.99 {
		t.Errorf("unexpected product data: %+v", found)
	}

	// 3. Update
	found.Price = 29.99
	affected, err := repo.Update(found)
	if err != nil || affected != 1 {
		t.Fatalf("expected 1 row updated, got %d, err=%v", affected, err)
	}

	// 4. All
	all, err := repo.All()
	if err != nil || len(all) != 1 {
		t.Fatalf("expected 1 product in All(), got %d, err=%v", len(all), err)
	}
	if all[0].Price != 29.99 {
		t.Errorf("expected updated price 29.99, got %f", all[0].Price)
	}

	// 5. Where
	items, err := repo.Where("price", "<", 30).Get()
	if err != nil || len(items) != 1 {
		t.Fatalf("expected 1 product in Where(), got %d, err=%v", len(items), err)
	}

	// 6. Delete
	delCount, err := repo.Delete(1)
	if err != nil || delCount != 1 {
		t.Fatalf("expected 1 row deleted, got %d, err=%v", delCount, err)
	}
}

