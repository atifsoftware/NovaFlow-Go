package core

import (
	"testing"
)

func TestQueryBuilderClone(t *testing.T) {
	qb := &QueryBuilder{
		table:   "products",
		columns: []string{"id", "name"},
		orderBy: "id DESC",
		limitN:  10,
		offsetN: 5,
		joins:   []string{"LEFT JOIN categories ON products.category_id = categories.id"},
		wheres: []whereClause{
			{boolean: "AND", column: "status", op: "=", value: "active"},
		},
	}

	clone := qb.Clone()

	if clone.table != qb.table {
		t.Error("table was not cloned")
	}
	if len(clone.columns) != len(qb.columns) || clone.columns[0] != qb.columns[0] {
		t.Error("columns were not cloned")
	}
	if clone.orderBy != qb.orderBy {
		t.Error("orderBy was not cloned")
	}
	if clone.limitN != qb.limitN {
		t.Error("limitN was not cloned")
	}
	if clone.offsetN != qb.offsetN {
		t.Error("offsetN was not cloned")
	}
	if len(clone.joins) != len(qb.joins) || clone.joins[0] != qb.joins[0] {
		t.Error("joins were not cloned")
	}
	if len(clone.wheres) != len(qb.wheres) || clone.wheres[0].column != qb.wheres[0].column {
		t.Error("wheres were not cloned")
	}


	clone.Where("price", ">", 100)
	if len(qb.wheres) != 1 {
		t.Error("mutating clone mutated original wheres slice")
	}
}

func TestQueryBuilderJoins(t *testing.T) {
	qb := &QueryBuilder{table: "products"}
	qb.LeftJoin("categories", "products.category_id", "=", "categories.id")
	qb.RightJoin("tags", "products.id", "=", "tags.product_id")
	qb.InnerJoin("orders", "products.id", "=", "orders.product_id")

	if len(qb.joins) != 3 {
		t.Fatalf("expected 3 joins, got %d", len(qb.joins))
	}

	if qb.joins[0] != "LEFT JOIN categories ON products.category_id = categories.id" {
		t.Errorf("unexpected left join: %s", qb.joins[0])
	}
	if qb.joins[1] != "RIGHT JOIN tags ON products.id = tags.product_id" {
		t.Errorf("unexpected right join: %s", qb.joins[1])
	}
	if qb.joins[2] != "INNER JOIN orders ON products.id = orders.product_id" {
		t.Errorf("unexpected inner join: %s", qb.joins[2])
	}
}

func TestQueryBuilderTxAndPluck(t *testing.T) {
	app := NewApp("../.env", "")
	if app.DB == nil {
		t.Skip("No database configured, skipping integration test")
	}

	_, _ = app.DB.Exec("DELETE FROM products")

	tx, err := app.DB.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	productId, err := tx.Table("products").Insert(map[string]interface{}{
		"name":  "Tx Rollback Product",
		"price": 45.99,
	})
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	rowInside, err := tx.Table("products").Where("id", "=", productId).First()
	if err != nil || rowInside == nil {
		t.Fatalf("row should exist inside transaction: %v", err)
	}

	rowOutside, err := app.DB.Table("products").Where("id", "=", productId).First()
	if err != nil {
		t.Fatal(err)
	}
	if rowOutside != nil {
		t.Fatal("row should not exist outside transaction before commit")
	}

	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	rowAfter, err := app.DB.Table("products").Where("id", "=", productId).First()
	if err != nil {
		t.Fatal(err)
	}
	if rowAfter != nil {
		t.Fatal("row should not exist after rollback")
	}

	_, err = app.DB.Table("products").Insert(map[string]interface{}{
		"name":  "Product A",
		"price": 10.00,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.DB.Table("products").Insert(map[string]interface{}{
		"name":  "Product B",
		"price": 20.00,
	})
	if err != nil {
		t.Fatal(err)
	}

	names, err := app.DB.Table("products").OrderBy("name", "ASC").Pluck("name")
	if err != nil {
		t.Fatalf("pluck failed: %v", err)
	}

	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0].(string) != "Product A" || names[1].(string) != "Product B" {
		t.Errorf("pluck returned incorrect values: %v", names)
	}
}
