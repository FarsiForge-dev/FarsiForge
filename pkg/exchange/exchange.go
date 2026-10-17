package exchange

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"farsiforge/pkg/core"

	"github.com/xuri/excelize/v2"
)

// validStatuses is the set of accepted import status values.
var validStatuses = map[core.Status]bool{
	core.StatusUntranslated: true,
	core.StatusTranslated:   true,
	core.StatusApproved:     true,
	core.StatusSkipped:      true,
}

// ImportResult reports the outcome of an import operation.
type ImportResult struct {
	Accepted int      // rows applied to the project
	Skipped  int      // rows skipped (empty ID/translation)
	Rejected int      // rows rejected due to validation errors
	Errors   []string // human-readable rejection reasons
}

// pendingImport holds a validated row ready to be applied.
type pendingImport struct {
	id          string
	translation string
	status      core.Status
}

// ExportXLSX exports project entries to an Excel file for translation.
// Columns: ID | Source | Translation | Status | Context | File | Path | Notes
func ExportXLSX(proj *core.Project, outputPath string) (retErr error) {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); retErr == nil && err != nil {
			retErr = fmt.Errorf("close xlsx: %w", err)
		}
	}()

	sheet := "Translations"
	if err := f.SetSheetName(f.GetSheetName(0), sheet); err != nil {
		return fmt.Errorf("rename xlsx sheet: %w", err)
	}

	// Set headers
	headers := []string{"ID", "Source (متن اصلی)", "Translation (ترجمه)", "Status", "Context", "File", "Path", "Notes"}
	for i, h := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return fmt.Errorf("header cell: %w", err)
		}
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return fmt.Errorf("write header cell %s: %w", cell, err)
		}
	}

	// Style header row
	style, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return fmt.Errorf("create header style: %w", err)
	}
	if err := f.SetCellStyle(sheet, "A1", "H1", style); err != nil {
		return fmt.Errorf("style header row: %w", err)
	}

	// Set column widths
	for _, width := range []struct {
		column string
		value  float64
	}{{"A", 10}, {"B", 40}, {"C", 40}, {"D", 15}, {"E", 15}, {"F", 30}, {"G", 30}, {"H", 20}} {
		if err := f.SetColWidth(sheet, width.column, width.column, width.value); err != nil {
			return fmt.Errorf("set column %s width: %w", width.column, err)
		}
	}

	// RTL alignment for Source and Translation columns
	rtlStyle, err := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center", ReadingOrder: 2},
	})
	if err != nil {
		return fmt.Errorf("create RTL style: %w", err)
	}

	// Data rows
	for i, entry := range proj.Entries {
		row := i + 2
		values := []interface{}{entry.ID, entry.Source, entry.Translation, string(entry.Status), entry.Context, entry.File, entry.Path, entry.Notes}
		for col, value := range values {
			cellName, err := excelize.CoordinatesToCellName(col+1, row)
			if err != nil {
				return fmt.Errorf("entry %d cell: %w", i, err)
			}
			if err := f.SetCellValue(sheet, cellName, value); err != nil {
				return fmt.Errorf("write entry %d cell %s: %w", i, cellName, err)
			}
		}

		// Apply RTL style to source and translation columns
		if err := f.SetCellStyle(sheet, cell(2, row), cell(3, row), rtlStyle); err != nil {
			return fmt.Errorf("style entry %d: %w", i, err)
		}
	}

	// Freeze header row
	if err := f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	}); err != nil {
		return fmt.Errorf("freeze xlsx header: %w", err)
	}

	// Enable autofilter
	if err := f.AutoFilter(sheet, "A1:H1", []excelize.AutoFilterOptions{}); err != nil {
		return fmt.Errorf("set xlsx filter: %w", err)
	}

	return f.SaveAs(outputPath)
}

// ImportXLSX imports translations from an Excel file atomically.
// ALL rows are validated before any change is applied to the project.
// If any row is invalid, zero changes are applied and an ImportResult
// with Rejected > 0 and a populated Errors list is returned.
func ImportXLSX(proj *core.Project, inputPath string) (ImportResult, error) {
	f, err := excelize.OpenFile(inputPath)
	if err != nil {
		return ImportResult{}, fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()

	sheet := "Translations"
	rows, err := f.GetRows(sheet)
	if err != nil {
		return ImportResult{}, fmt.Errorf("read sheet: %w", err)
	}

	if len(rows) < 2 {
		return ImportResult{}, fmt.Errorf("no data rows in xlsx")
	}

	return applyImport(proj, rows[1:])
}

// ExportCSV exports project entries to a CSV file.
func ExportCSV(proj *core.Project, outputPath string) (retErr error) {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); retErr == nil && err != nil {
			retErr = fmt.Errorf("close CSV: %w", err)
		}
	}()

	// Write BOM for Excel UTF-8 detection
	if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return fmt.Errorf("write CSV BOM: %w", err)
	}

	w := csv.NewWriter(file)

	// Header
	if err := w.Write([]string{"ID", "Source", "Translation", "Status", "Context", "File", "Path", "Notes"}); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}

	// Data
	for _, entry := range proj.Entries {
		if err := w.Write([]string{
			entry.ID,
			entry.Source,
			entry.Translation,
			string(entry.Status),
			entry.Context,
			entry.File,
			entry.Path,
			entry.Notes,
		}); err != nil {
			return fmt.Errorf("write CSV entry %s: %w", entry.ID, err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("flush CSV: %w", err)
	}
	return nil
}

// ImportCSV imports translations from a CSV file atomically.
// ALL rows are validated before any change is applied to the project.
// If any row is invalid, zero changes are applied and an ImportResult
// with Rejected > 0 and a populated Errors list is returned.
func ImportCSV(proj *core.Project, inputPath string) (ImportResult, error) {
	file, err := os.Open(inputPath)
	if err != nil {
		return ImportResult{}, err
	}
	defer file.Close()

	// Skip BOM if present
	bom := make([]byte, 3)
	n, err := io.ReadFull(file, bom)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return ImportResult{}, fmt.Errorf("read CSV BOM: %w", err)
	}
	if n < 3 || bom[0] != 0xEF || bom[1] != 0xBB || bom[2] != 0xBF {
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return ImportResult{}, fmt.Errorf("rewind CSV: %w", err)
		}
	}

	r := csv.NewReader(file)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	rows, err := r.ReadAll()
	if err != nil {
		return ImportResult{}, err
	}

	if len(rows) < 2 {
		return ImportResult{}, fmt.Errorf("no data rows")
	}

	return applyImport(proj, rows[1:])
}

// applyImport validates all rows, then applies zero changes on any error
// (full atomicity). Returns ImportResult.
func applyImport(proj *core.Project, dataRows [][]string) (ImportResult, error) {
	var result ImportResult
	var pending []pendingImport
	seenIDs := make(map[string]int) // id -> first row index (1-based)

	for rowIdx, row := range dataRows {
		// Skip obviously empty or malformed rows.
		if len(row) < 3 {
			result.Skipped++
			continue
		}
		id := strings.TrimSpace(row[0])
		translation := row[2] // intentionally NOT trimmed: preserve meaningful whitespace

		if id == "" || translation == "" {
			result.Skipped++
			continue
		}

		// Duplicate ID check.
		if firstRow, exists := seenIDs[id]; exists {
			result.Rejected++
			result.Errors = append(result.Errors,
				fmt.Sprintf("row %d: duplicate ID %q (first seen at row %d)", rowIdx+1, id, firstRow))
			continue
		}
		seenIDs[id] = rowIdx + 1

		// Unknown ID check.
		if !proj.HasEntry(id) {
			result.Rejected++
			result.Errors = append(result.Errors,
				fmt.Sprintf("row %d: unknown entry ID %q", rowIdx+1, id))
			continue
		}

		// Status validation.
		status := core.StatusTranslated
		if len(row) > 3 && row[3] != "" {
			s := core.Status(strings.TrimSpace(row[3]))
			if !validStatuses[s] {
				result.Rejected++
				result.Errors = append(result.Errors,
					fmt.Sprintf("row %d: invalid status %q for ID %q (allowed: untranslated, translated, approved, skipped)",
						rowIdx+1, s, id))
				continue
			}
			status = s
		}

		pending = append(pending, pendingImport{id: id, translation: translation, status: status})
	}

	// If any rows were rejected, apply ZERO changes (atomic import).
	if result.Rejected > 0 {
		return result, fmt.Errorf("%d row(s) failed validation — no changes applied", result.Rejected)
	}

	// All rows valid — apply atomically.
	for _, p := range pending {
		if err := proj.SetTranslation(p.id, p.translation, p.status); err != nil {
			// SetTranslation failure after validation is unexpected; surface it.
			return result, fmt.Errorf("apply row for %q: %w", p.id, err)
		}
		result.Accepted++
	}
	return result, nil
}

// Export exports to XLSX or CSV based on file extension.
func Export(proj *core.Project, outputPath string) error {
	ext := strings.ToLower(filepath.Ext(outputPath))
	switch ext {
	case ".xlsx":
		return ExportXLSX(proj, outputPath)
	case ".csv":
		return ExportCSV(proj, outputPath)
	default:
		return fmt.Errorf("unsupported format: %s (use .xlsx or .csv)", ext)
	}
}

// Import imports from XLSX or CSV based on file extension.
func Import(proj *core.Project, inputPath string) (ImportResult, error) {
	ext := strings.ToLower(filepath.Ext(inputPath))
	switch ext {
	case ".xlsx":
		return ImportXLSX(proj, inputPath)
	case ".csv":
		return ImportCSV(proj, inputPath)
	default:
		return ImportResult{}, fmt.Errorf("unsupported format: %s (use .xlsx or .csv)", ext)
	}
}

func cell(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}
