package core

import (
	"fmt"
	"testing"
)

type pageItem struct {
	ID    int64   `db:"id"`
	Title string  `db:"title"`
	Price float64 `db:"price"`
}

func (pageItem) TableName() string { return "page_items" }

func TestQueryBuilderAndRepositoryPagination(t *testing.T) {
	db, err := OpenDBWithDialect(&SQLiteDialect{}, ":memory:")
	if err != nil {
		t.Fatalf("could not open in-memory sqlite db: %v", err)
	}
	defer db.Conn.Close()

	// 1. Create table
	_, err = db.Exec(`CREATE TABLE page_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		price REAL NOT NULL
	)`)
	if err != nil {
		t.Fatalf("could not create test table: %v", err)
	}

	// 2. Insert 10 rows
	for i := 1; i <= 10; i++ {
		_, err := db.Table("page_items").Insert(map[string]interface{}{
			"title": fmt.Sprintf("Item %d", i),
			"price": float64(i * 10),
		})
		if err != nil {
			t.Fatalf("insert failed for item %d: %v", i, err)
		}
	}

	// 3. Test QueryBuilder.Paginate() Page 1 (perPage = 3)
	qbRes, err := db.Table("page_items").OrderBy("id", "ASC").Paginate(1, 3)
	if err != nil {
		t.Fatalf("QueryBuilder Paginate failed: %v", err)
	}

	if qbRes.Total != 10 {
		t.Errorf("expected total 10, got %d", qbRes.Total)
	}
	if qbRes.CurrentPage != 1 {
		t.Errorf("expected current_page 1, got %d", qbRes.CurrentPage)
	}
	if qbRes.PerPage != 3 {
		t.Errorf("expected per_page 3, got %d", qbRes.PerPage)
	}
	if qbRes.LastPage != 4 {
		t.Errorf("expected last_page 4, got %d", qbRes.LastPage)
	}
	if qbRes.From != 1 || qbRes.To != 3 {
		t.Errorf("expected from=1, to=3, got from=%d, to=%d", qbRes.From, qbRes.To)
	}
	if len(qbRes.Data) != 3 {
		t.Fatalf("expected 3 items on page 1, got %d", len(qbRes.Data))
	}
	if qbRes.Data[0]["title"] != "Item 1" {
		t.Errorf("expected first item 'Item 1', got %v", qbRes.Data[0]["title"])
	}

	// 4. Test QueryBuilder.Paginate() Page 4 (last page, perPage = 3 -> only 1 item remaining)
	page4Res, err := db.Table("page_items").OrderBy("id", "ASC").Paginate(4, 3)
	if err != nil {
		t.Fatalf("QueryBuilder Paginate page 4 failed: %v", err)
	}
	if len(page4Res.Data) != 1 {
		t.Fatalf("expected 1 item on page 4, got %d", len(page4Res.Data))
	}
	if page4Res.From != 10 || page4Res.To != 10 {
		t.Errorf("expected from=10, to=10, got from=%d, to=%d", page4Res.From, page4Res.To)
	}

	// 5. Test Repository[pageItem].Paginate()
	repo := NewRepository[pageItem](db)
	repoRes, err := repo.Where("price", ">=", 50).Paginate(1, 4)
	if err != nil {
		t.Fatalf("Repository Paginate failed: %v", err)
	}

	// Items >= 50: Item 5, 6, 7, 8, 9, 10 -> Total = 6 items
	if repoRes.Total != 6 {
		t.Errorf("expected total 6, got %d", repoRes.Total)
	}
	if repoRes.LastPage != 2 {
		t.Errorf("expected last_page 2, got %d", repoRes.LastPage)
	}
	if len(repoRes.Data) != 4 {
		t.Fatalf("expected 4 items on page 1, got %d", len(repoRes.Data))
	}
	if repoRes.Data[0].Title != "Item 5" {
		t.Errorf("expected first item 'Item 5', got %v", repoRes.Data[0].Title)
	}
}
