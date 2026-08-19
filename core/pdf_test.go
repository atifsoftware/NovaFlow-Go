package core

import (
	"bytes"
	"testing"
)

func TestGenerateInvoicePDF(t *testing.T) {
	data := InvoicePDFData{
		CompanyName:    "TechFlow Solutions Ltd.",
		CompanyAddress: "Gulshan-2, Dhaka-1212, Bangladesh",
		CompanyPhone:   "+880 1711 000000",
		CompanyEmail:   "billing@techflow.com",
		InvoiceNumber:  "INV-2026-0042",
		InvoiceDate:    "2026-08-19",
		DueDate:        "2026-09-19",
		CustomerName:   "Apex Enterprises",
		CustomerAddress: "Banani, Dhaka",
		CustomerPhone:  "+880 1811 111111",
		Items: []InvoiceItem{
			{Description: "ERP Cloud Setup & Architecture", Qty: 1, UnitPrice: 50000, Total: 50000},
			{Description: "Custom POS Integration Module", Qty: 2, UnitPrice: 15000, Total: 30000},
		},
		SubTotal: 80000,
		Discount: 5000,
		Tax:      3750,
		Total:    78750,
		Currency: "BDT",
		Notes:    "Thank you for choosing TechFlow. Please pay within 30 days.",
	}

	pdfBytes, err := GenerateInvoicePDF(data)
	if err != nil {
		t.Fatalf("GenerateInvoicePDF failed: %v", err)
	}

	if len(pdfBytes) < 1000 {
		t.Fatalf("generated PDF bytes unexpectedly small: %d bytes", len(pdfBytes))
	}

	// Verify valid PDF file header (%PDF-)
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("expected PDF byte header %%PDF-, got: %s", string(pdfBytes[:10]))
	}
}

func TestGenerateMoneyReceiptPDF(t *testing.T) {
	data := MoneyReceiptPDFData{
		CompanyName:    "TechFlow Solutions Ltd.",
		CompanyAddress: "Gulshan-2, Dhaka-1212",
		CompanyPhone:   "+880 1711 000000",
		ReceiptNumber:  "MR-2026-019",
		ReceiptDate:    "2026-08-19",
		ReceivedFrom:   "Apex Enterprises",
		Amount:         78750,
		Currency:       "BDT",
		PaymentMethod:  "Bank Transfer (City Bank)",
		ForAccountOf:   "Payment against INV-2026-0042",
		Remarks:        "Full payment received with thanks.",
	}

	pdfBytes, err := GenerateMoneyReceiptPDF(data)
	if err != nil {
		t.Fatalf("GenerateMoneyReceiptPDF failed: %v", err)
	}

	if len(pdfBytes) < 500 {
		t.Fatalf("generated Money Receipt PDF bytes unexpectedly small: %d bytes", len(pdfBytes))
	}

	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("expected PDF byte header %%PDF-, got: %s", string(pdfBytes[:10]))
	}
}
