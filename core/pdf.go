package core

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
)

// InvoiceItem represents a line item in an invoice.
type InvoiceItem struct {
	Description string
	Qty         float64
	UnitPrice   float64
	Total       float64
}

// InvoicePDFData holds all fields necessary to render a professional invoice PDF.
type InvoicePDFData struct {
	CompanyName    string
	CompanyAddress string
	CompanyPhone   string
	CompanyEmail   string

	InvoiceNumber string
	InvoiceDate   string
	DueDate       string

	CustomerName    string
	CustomerAddress string
	CustomerPhone   string

	Items    []InvoiceItem
	SubTotal float64
	Discount float64
	Tax      float64
	Total    float64

	Currency string // default: BDT
	InWords  string // In words representation (if empty, auto-generated)
	Notes    string
}

// GenerateInvoicePDF renders a clean, professional invoice PDF into a byte buffer.
func GenerateInvoicePDF(data InvoicePDFData) ([]byte, error) {
	if data.Currency == "" {
		data.Currency = "BDT"
	}
	if data.InWords == "" {
		data.InWords = NumberToWords(data.Total, NumberToWordsOptions{Currency: data.Currency})
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	// --- Header Section ---
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetTextColor(33, 37, 41)
	pdf.Cell(120, 8, data.CompanyName)

	pdf.SetFont("Helvetica", "B", 20)
	pdf.SetTextColor(13, 110, 253) // Blue accent
	pdf.CellFormat(60, 8, "INVOICE", "", 1, "R", false, 0, "")

	// Company Subtitle
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(108, 117, 125)
	pdf.Cell(120, 5, data.CompanyAddress)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(33, 37, 41)
	pdf.CellFormat(60, 5, fmt.Sprintf("Invoice #: %s", data.InvoiceNumber), "", 1, "R", false, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(108, 117, 125)
	pdf.Cell(120, 5, fmt.Sprintf("Phone: %s | Email: %s", data.CompanyPhone, data.CompanyEmail))
	pdf.CellFormat(60, 5, fmt.Sprintf("Date: %s", data.InvoiceDate), "", 1, "R", false, 0, "")

	if data.DueDate != "" {
		pdf.Cell(120, 5, "")
		pdf.CellFormat(60, 5, fmt.Sprintf("Due Date: %s", data.DueDate), "", 1, "R", false, 0, "")
	}

	pdf.Ln(8)

	// --- Bill To Box ---
	pdf.SetDrawColor(222, 226, 230)
	pdf.SetFillColor(248, 249, 250)
	pdf.Rect(15, pdf.GetY(), 180, 20, "FD")

	pdf.SetXY(18, pdf.GetY()+2)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(73, 80, 87)
	pdf.Cell(174, 5, "BILL TO:")
	pdf.Ln(5)

	pdf.SetX(18)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(33, 37, 41)
	pdf.Cell(174, 5, data.CustomerName)
	pdf.Ln(4)

	pdf.SetX(18)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(108, 117, 125)
	custInfo := data.CustomerAddress
	if data.CustomerPhone != "" {
		custInfo += " | Tel: " + data.CustomerPhone
	}
	pdf.Cell(174, 4, custInfo)

	pdf.Ln(12)

	// --- Items Table Header ---
	pdf.SetFillColor(52, 58, 64)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 9)

	pdf.CellFormat(15, 8, "#", "1", 0, "C", true, 0, "")
	pdf.CellFormat(85, 8, "Description", "1", 0, "L", true, 0, "")
	pdf.CellFormat(25, 8, "Qty", "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 8, "Unit Price", "1", 0, "R", true, 0, "")
	pdf.CellFormat(30, 8, "Total Amount", "1", 1, "R", true, 0, "")

	// --- Items Table Body ---
	pdf.SetTextColor(33, 37, 41)
	pdf.SetFont("Helvetica", "", 9)
	fill := false

	for i, item := range data.Items {
		if fill {
			pdf.SetFillColor(248, 249, 250)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		itemTotal := item.Total
		if itemTotal == 0 {
			itemTotal = item.Qty * item.UnitPrice
		}

		pdf.CellFormat(15, 7, fmt.Sprintf("%d", i+1), "1", 0, "C", fill, 0, "")
		pdf.CellFormat(85, 7, item.Description, "1", 0, "L", fill, 0, "")
		pdf.CellFormat(25, 7, fmt.Sprintf("%.2f", item.Qty), "1", 0, "C", fill, 0, "")
		pdf.CellFormat(25, 7, fmt.Sprintf("%.2f", item.UnitPrice), "1", 0, "R", fill, 0, "")
		pdf.CellFormat(30, 7, fmt.Sprintf("%.2f", itemTotal), "1", 1, "R", fill, 0, "")

		fill = !fill
	}

	pdf.Ln(3)

	// --- Calculations Summary (Right-aligned) ---
	calcLeft := 115.0
	calcLabelWidth := 35.0
	calcValWidth := 30.0

	pdf.SetFont("Helvetica", "", 9)
	if data.SubTotal > 0 {
		pdf.SetX(calcLeft)
		pdf.CellFormat(calcLabelWidth, 6, "Sub Total:", "", 0, "R", false, 0, "")
		pdf.CellFormat(calcValWidth, 6, fmt.Sprintf("%.2f %s", data.SubTotal, data.Currency), "", 1, "R", false, 0, "")
	}

	if data.Discount > 0 {
		pdf.SetX(calcLeft)
		pdf.CellFormat(calcLabelWidth, 6, "Discount:", "", 0, "R", false, 0, "")
		pdf.CellFormat(calcValWidth, 6, fmt.Sprintf("-%.2f %s", data.Discount, data.Currency), "", 1, "R", false, 0, "")
	}

	if data.Tax > 0 {
		pdf.SetX(calcLeft)
		pdf.CellFormat(calcLabelWidth, 6, "Tax / VAT:", "", 0, "R", false, 0, "")
		pdf.CellFormat(calcValWidth, 6, fmt.Sprintf("+%.2f %s", data.Tax, data.Currency), "", 1, "R", false, 0, "")
	}

	// Grand Total Row
	pdf.SetX(calcLeft)
	pdf.SetFillColor(230, 240, 255)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(13, 110, 253)
	pdf.CellFormat(calcLabelWidth, 8, "Grand Total:", "1", 0, "R", true, 0, "")
	pdf.CellFormat(calcValWidth, 8, fmt.Sprintf("%.2f %s", data.Total, data.Currency), "1", 1, "R", true, 0, "")

	pdf.Ln(6)

	// --- In Words Box ---
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(73, 80, 87)
	pdf.Cell(30, 6, "Amount in Words:")
	pdf.SetFont("Helvetica", "I", 9)
	pdf.SetTextColor(33, 37, 41)
	pdf.MultiCell(150, 6, data.InWords, "", "L", false)

	pdf.Ln(4)

	// Notes & Terms
	if data.Notes != "" {
		pdf.SetFont("Helvetica", "B", 8)
		pdf.SetTextColor(108, 117, 125)
		pdf.Cell(180, 5, "Terms & Notes:")
		pdf.Ln(4)
		pdf.SetFont("Helvetica", "", 8)
		pdf.MultiCell(180, 4, data.Notes, "", "L", false)
	}

	// Signature Lines at Bottom
	pdf.SetY(260)
	pdf.SetDrawColor(180, 180, 180)
	pdf.Line(20, 260, 70, 260)
	pdf.Line(130, 260, 180, 260)

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(108, 117, 125)
	pdf.SetXY(20, 262)
	pdf.CellFormat(50, 4, "Customer Signature", "", 0, "C", false, 0, "")
	pdf.SetXY(130, 262)
	pdf.CellFormat(50, 4, "Authorized Signature", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// MoneyReceiptPDFData holds fields for rendering a money receipt PDF.
type MoneyReceiptPDFData struct {
	CompanyName    string
	CompanyAddress string
	CompanyPhone   string

	ReceiptNumber string
	ReceiptDate   string
	ReceivedFrom  string
	Amount        float64
	Currency      string // default: BDT
	InWords       string // auto-generated if empty
	PaymentMethod string // Cash, Bank, Cheque, bKash, etc.
	ForAccountOf  string // Purpose / Invoice No
	Remarks       string
}

// GenerateMoneyReceiptPDF renders a standard Money Receipt document into PDF bytes.
func GenerateMoneyReceiptPDF(data MoneyReceiptPDFData) ([]byte, error) {
	if data.Currency == "" {
		data.Currency = "BDT"
	}
	if data.InWords == "" {
		data.InWords = NumberToWords(data.Amount, NumberToWordsOptions{Currency: data.Currency})
	}

	pdf := fpdf.New("P", "mm", "A5", "") // A5 size is standard for money receipts
	pdf.SetMargins(12, 12, 12)
	pdf.AddPage()

	// Header Box
	pdf.SetFont("Helvetica", "B", 15)
	pdf.SetTextColor(33, 37, 41)
	pdf.Cell(80, 7, data.CompanyName)

	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(25, 135, 84) // Green accent for receipt
	pdf.CellFormat(40, 7, "MONEY RECEIPT", "", 1, "R", false, 0, "")

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(108, 117, 125)
	pdf.Cell(80, 4, data.CompanyAddress)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(33, 37, 41)
	pdf.CellFormat(40, 4, fmt.Sprintf("Receipt #: %s", data.ReceiptNumber), "", 1, "R", false, 0, "")

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(108, 117, 125)
	pdf.Cell(80, 4, fmt.Sprintf("Phone: %s", data.CompanyPhone))
	pdf.CellFormat(40, 4, fmt.Sprintf("Date: %s", data.ReceiptDate), "", 1, "R", false, 0, "")

	pdf.Ln(6)

	// Details Card
	pdf.SetDrawColor(200, 200, 200)
	pdf.SetFillColor(250, 250, 250)
	pdf.Rect(12, pdf.GetY(), 124, 65, "FD")

	yStart := pdf.GetY() + 4

	// Received From
	pdf.SetXY(16, yStart)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(73, 80, 87)
	pdf.Cell(32, 6, "Received From:")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(33, 37, 41)
	pdf.Cell(80, 6, data.ReceivedFrom)

	// Amount
	pdf.SetXY(16, yStart+8)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(73, 80, 87)
	pdf.Cell(32, 6, "Amount Paid:")
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetTextColor(25, 135, 84)
	pdf.Cell(80, 6, fmt.Sprintf("%.2f %s", data.Amount, data.Currency))

	// In Words
	pdf.SetXY(16, yStart+16)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(73, 80, 87)
	pdf.Cell(32, 6, "In Words:")
	pdf.SetFont("Helvetica", "I", 9)
	pdf.SetTextColor(33, 37, 41)
	pdf.MultiCell(72, 5, data.InWords, "", "L", false)

	// Payment Method
	pdf.SetXY(16, yStart+30)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(73, 80, 87)
	pdf.Cell(32, 6, "Payment Method:")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(33, 37, 41)
	pdf.Cell(80, 6, strings.ToUpper(data.PaymentMethod))

	// For Purpose
	pdf.SetXY(16, yStart+38)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(73, 80, 87)
	pdf.Cell(32, 6, "On Account Of:")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(33, 37, 41)
	pdf.Cell(80, 6, data.ForAccountOf)

	// Remarks
	if data.Remarks != "" {
		pdf.SetXY(16, yStart+46)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(73, 80, 87)
		pdf.Cell(32, 6, "Remarks:")
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(33, 37, 41)
		pdf.Cell(80, 6, data.Remarks)
	}

	// Signatures
	pdf.SetY(165)
	pdf.SetDrawColor(180, 180, 180)
	pdf.Line(16, 165, 55, 165)
	pdf.Line(85, 165, 125, 165)

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(108, 117, 125)
	pdf.SetXY(16, 167)
	pdf.CellFormat(39, 4, "Received By", "", 0, "C", false, 0, "")
	pdf.SetXY(85, 167)
	pdf.CellFormat(40, 4, "Authorized Signature", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
