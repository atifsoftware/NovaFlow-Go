package core

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ExportCSV writes rows to an io.Writer in CSV format.
func ExportCSV(w io.Writer, headers []string, rows [][]interface{}) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	if len(headers) > 0 {
		if err := writer.Write(headers); err != nil {
			return err
		}
	}

	for _, row := range rows {
		record := make([]string, len(row))
		for i, val := range row {
			if val == nil {
				record[i] = ""
			} else {
				record[i] = fmt.Sprintf("%v", val)
			}
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// ExportXLSX creates an Excel workbook in memory with styled headers and returns the file bytes.
func ExportXLSX(sheetName string, headers []string, rows [][]interface{}) ([]byte, error) {
	if sheetName == "" {
		sheetName = "Sheet1"
	}

	f := excelize.NewFile()
	defer f.Close()

	// Default sheet is Sheet1; rename if necessary
	if sheetName != "Sheet1" {
		f.SetSheetName("Sheet1", sheetName)
	}

	// Create header style: Bold text, light gray background, center align
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 11,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E0E0E0"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})

	// Write headers
	for colIdx, header := range headers {
		cell, err := excelize.CoordinatesToCellName(colIdx+1, 1)
		if err != nil {
			return nil, err
		}
		_ = f.SetCellValue(sheetName, cell, header)
		_ = f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// Write data rows
	for rowIdx, row := range rows {
		for colIdx, val := range row {
			cell, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			if err != nil {
				return nil, err
			}
			_ = f.SetCellValue(sheetName, cell, val)
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ImportCSV reads a CSV stream and parses it into a slice of column->value maps.
func ImportCSV(r io.Reader) ([]map[string]string, error) {
	reader := csv.NewReader(r)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return []map[string]string{}, nil
	}

	headers := records[0]
	results := make([]map[string]string, 0, len(records)-1)

	for i := 1; i < len(records); i++ {
		row := records[i]
		rowMap := make(map[string]string, len(headers))
		for j, header := range headers {
			if j < len(row) {
				rowMap[strings.TrimSpace(header)] = strings.TrimSpace(row[j])
			} else {
				rowMap[strings.TrimSpace(header)] = ""
			}
		}
		results = append(results, rowMap)
	}

	return results, nil
}

// ImportXLSX reads an XLSX stream and parses the specified sheet into a slice of column->value maps.
func ImportXLSX(r io.Reader, sheetName string) ([]map[string]string, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if sheetName == "" {
		sheetList := f.GetSheetList()
		if len(sheetList) > 0 {
			sheetName = sheetList[0]
		} else {
			sheetName = "Sheet1"
		}
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, err
	}

	if len(rows) < 2 {
		return []map[string]string{}, nil
	}

	headers := rows[0]
	results := make([]map[string]string, 0, len(rows)-1)

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		rowMap := make(map[string]string, len(headers))
		for j, header := range headers {
			header = strings.TrimSpace(header)
			if j < len(row) {
				rowMap[header] = strings.TrimSpace(row[j])
			} else {
				rowMap[header] = ""
			}
		}
		results = append(results, rowMap)
	}

	return results, nil
}
