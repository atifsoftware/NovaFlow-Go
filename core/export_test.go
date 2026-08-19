package core

import (
	"bytes"
	"testing"
)

func TestExportAndImportCSV(t *testing.T) {
	headers := []string{"ID", "Product", "Price", "Status"}
	rows := [][]interface{}{
		{1, "MacBook Pro", 1999.99, "active"},
		{2, "Dell XPS 15", 1499.50, "inactive"},
	}

	var buf bytes.Buffer
	if err := ExportCSV(&buf, headers, rows); err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}

	imported, err := ImportCSV(&buf)
	if err != nil {
		t.Fatalf("ImportCSV failed: %v", err)
	}

	if len(imported) != 2 {
		t.Fatalf("expected 2 imported rows, got %d", len(imported))
	}

	if imported[0]["Product"] != "MacBook Pro" || imported[0]["Price"] != "1999.99" {
		t.Errorf("unexpected first row data: %+v", imported[0])
	}
	if imported[1]["Product"] != "Dell XPS 15" || imported[1]["Status"] != "inactive" {
		t.Errorf("unexpected second row data: %+v", imported[1])
	}
}

func TestExportAndImportXLSX(t *testing.T) {
	headers := []string{"Invoice No", "Customer", "Amount", "Status"}
	rows := [][]interface{}{
		{"INV-001", "Acme Corp", 5400.00, "Paid"},
		{"INV-002", "Globex Inc", 12500.50, "Unpaid"},
	}

	xlsxBytes, err := ExportXLSX("Invoices", headers, rows)
	if err != nil {
		t.Fatalf("ExportXLSX failed: %v", err)
	}

	if len(xlsxBytes) == 0 {
		t.Fatal("expected non-empty XLSX bytes")
	}

	reader := bytes.NewReader(xlsxBytes)
	imported, err := ImportXLSX(reader, "Invoices")
	if err != nil {
		t.Fatalf("ImportXLSX failed: %v", err)
	}

	if len(imported) != 2 {
		t.Fatalf("expected 2 imported rows from XLSX, got %d", len(imported))
	}

	if imported[0]["Invoice No"] != "INV-001" || imported[0]["Customer"] != "Acme Corp" {
		t.Errorf("unexpected first row from XLSX: %+v", imported[0])
	}
	if imported[1]["Invoice No"] != "INV-002" || imported[1]["Status"] != "Unpaid" {
		t.Errorf("unexpected second row from XLSX: %+v", imported[1])
	}
}
