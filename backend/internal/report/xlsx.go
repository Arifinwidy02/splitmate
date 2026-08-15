package report

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

const (
	sheetSummary     = "Summary"
	sheetExpenses    = "Expenses"
	sheetSettlements = "Settlements"

	headerFill    = "1E8E5A"
	headerFontHex = "FFFFFF"
)

var amountNumFmt = `#,##0.00`

// RenderXLSX builds the report workbook. It is a pure function: the same
// report always produces the same workbook content, so it can be tested
// without HTTP or database dependencies.
func RenderXLSX(r *Report) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	if err := writeSummarySheet(f, r); err != nil {
		return nil, err
	}
	if err := writeExpensesSheet(f, r); err != nil {
		return nil, err
	}
	if err := writeSettlementsSheet(f, r); err != nil {
		return nil, err
	}

	f.DeleteSheet("Sheet1")

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("write workbook: %w", err)
	}
	return buf.Bytes(), nil
}

func writeSummarySheet(f *excelize.File, r *Report) error {
	titleStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	if err != nil {
		return err
	}
	labelStyle, err := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	if err != nil {
		return err
	}
	headerStyle, err := headerStyle(f)
	if err != nil {
		return err
	}
	moneyStyle, err := f.NewStyle(&excelize.Style{
		CustomNumFmt: &amountNumFmt,
	})
	if err != nil {
		return err
	}
	sectionStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 13},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	if err != nil {
		return err
	}

	f.SetSheetName("Sheet1", sheetSummary)

	f.SetCellValue(sheetSummary, "A1", "SplitMate Report")
	f.SetCellStyle(sheetSummary, "A1", "A1", titleStyle)
	f.SetCellValue(sheetSummary, "A2", fmt.Sprintf("Group: %s", r.GroupName))
	f.SetCellStyle(sheetSummary, "A2", "A2", labelStyle)
	f.SetCellValue(sheetSummary, "A3", fmt.Sprintf("Currency: %s", r.GroupCurrency))
	f.SetCellStyle(sheetSummary, "A3", "A3", labelStyle)
	f.SetCellValue(sheetSummary, "A4", fmt.Sprintf("Generated: %s", r.GeneratedAt.Format("2006-01-02 15:04 UTC")))
	f.SetCellStyle(sheetSummary, "A4", "A4", labelStyle)

	if err := f.SetColWidth(sheetSummary, "A", "A", 40); err != nil {
		return err
	}
	if err := f.SetColWidth(sheetSummary, "B", "B", 18); err != nil {
		return err
	}
	if err := f.SetColWidth(sheetSummary, "C", "C", 18); err != nil {
		return err
	}

	row := 6

	sectionRow := row
	f.SetCellValue(sheetSummary, fmt.Sprintf("A%d", sectionRow), "Balances")
	f.SetCellStyle(sheetSummary, fmt.Sprintf("A%d", sectionRow), fmt.Sprintf("A%d", sectionRow), sectionStyle)
	row++

	f.SetCellValue(sheetSummary, fmt.Sprintf("A%d", row), "Member")
	f.SetCellValue(sheetSummary, fmt.Sprintf("B%d", row), "Balance")
	f.SetCellStyle(sheetSummary, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), headerStyle)
	row++

	firstBalanceRow := row
	for _, b := range r.Balances {
		f.SetCellValue(sheetSummary, fmt.Sprintf("A%d", row), b.Name)
		f.SetCellValue(sheetSummary, fmt.Sprintf("B%d", row), toFloat(b.BalanceSen))
		f.SetCellStyle(sheetSummary, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), moneyStyle)
		row++
	}
	if len(r.Balances) > 0 {
		if err := f.AutoFilter(sheetSummary, fmt.Sprintf("A%d:B%d", firstBalanceRow, row-1), nil); err != nil {
			return err
		}
	}
	row++

	sectionRow = row
	f.SetCellValue(sheetSummary, fmt.Sprintf("A%d", sectionRow), "Settlement suggestions")
	f.SetCellStyle(sheetSummary, fmt.Sprintf("A%d", sectionRow), fmt.Sprintf("A%d", sectionRow), sectionStyle)
	row++

	f.SetCellValue(sheetSummary, fmt.Sprintf("A%d", row), "From")
	f.SetCellValue(sheetSummary, fmt.Sprintf("B%d", row), "To")
	f.SetCellValue(sheetSummary, fmt.Sprintf("C%d", row), "Amount")
	f.SetCellStyle(sheetSummary, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), headerStyle)
	row++

	for _, s := range r.Suggestions {
		f.SetCellValue(sheetSummary, fmt.Sprintf("A%d", row), s.FromName)
		f.SetCellValue(sheetSummary, fmt.Sprintf("B%d", row), s.ToName)
		f.SetCellValue(sheetSummary, fmt.Sprintf("C%d", row), toFloat(s.AmountSen))
		f.SetCellStyle(sheetSummary, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), moneyStyle)
		row++
	}

	return nil
}

func writeExpensesSheet(f *excelize.File, r *Report) error {
	headerStyle, err := headerStyle(f)
	if err != nil {
		return err
	}
	moneyStyle, err := f.NewStyle(&excelize.Style{CustomNumFmt: &amountNumFmt})
	if err != nil {
		return err
	}
	totalStyle, err := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true},
		CustomNumFmt: &amountNumFmt,
	})
	if err != nil {
		return err
	}
	boldStyle, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return err
	}

	idx, err := f.NewSheet(sheetExpenses)
	if err != nil {
		return err
	}
	f.SetActiveSheet(idx)

	headers := []string{"Date", "Description", "Category", "Paid By", "Amount", "Participants", "Note"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetExpenses, cell, h)
	}
	f.SetCellStyle(sheetExpenses, "A1", "G1", headerStyle)

	row := 2
	var totalSen int64
	for _, e := range r.Expenses {
		f.SetCellValue(sheetExpenses, fmt.Sprintf("A%d", row), e.ExpenseDate.Format("2006-01-02"))
		f.SetCellValue(sheetExpenses, fmt.Sprintf("B%d", row), e.Description)
		f.SetCellValue(sheetExpenses, fmt.Sprintf("C%d", row), e.Category)
		f.SetCellValue(sheetExpenses, fmt.Sprintf("D%d", row), e.PaidByName)
		f.SetCellValue(sheetExpenses, fmt.Sprintf("E%d", row), toFloat(e.AmountSen))
		f.SetCellStyle(sheetExpenses, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), moneyStyle)
		f.SetCellValue(sheetExpenses, fmt.Sprintf("F%d", row), participantsText(e.Participants))
		f.SetCellValue(sheetExpenses, fmt.Sprintf("G%d", row), e.Note)
		totalSen += e.AmountSen
		row++
	}

	if len(r.Expenses) > 0 {
		f.SetCellValue(sheetExpenses, fmt.Sprintf("B%d", row), "TOTAL")
		f.SetCellStyle(sheetExpenses, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), boldStyle)
		f.SetCellValue(sheetExpenses, fmt.Sprintf("E%d", row), toFloat(totalSen))
		f.SetCellStyle(sheetExpenses, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), totalStyle)
	}

	widths := map[string]float64{"A": 12, "B": 32, "C": 18, "D": 18, "E": 14, "F": 45, "G": 30}
	for col, w := range widths {
		if err := f.SetColWidth(sheetExpenses, col, col, w); err != nil {
			return err
		}
	}

	return nil
}

func writeSettlementsSheet(f *excelize.File, r *Report) error {
	headerStyle, err := headerStyle(f)
	if err != nil {
		return err
	}
	moneyStyle, err := f.NewStyle(&excelize.Style{CustomNumFmt: &amountNumFmt})
	if err != nil {
		return err
	}

	idx, err := f.NewSheet(sheetSettlements)
	if err != nil {
		return err
	}
	f.SetActiveSheet(idx)

	headers := []string{"Date", "Payer", "Receiver", "Amount"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetSettlements, cell, h)
	}
	f.SetCellStyle(sheetSettlements, "A1", "D1", headerStyle)

	row := 2
	for _, s := range r.Settlements {
		f.SetCellValue(sheetSettlements, fmt.Sprintf("A%d", row), s.SettledAt.Format("2006-01-02 15:04"))
		f.SetCellValue(sheetSettlements, fmt.Sprintf("B%d", row), s.PayerName)
		f.SetCellValue(sheetSettlements, fmt.Sprintf("C%d", row), s.ReceiverName)
		f.SetCellValue(sheetSettlements, fmt.Sprintf("D%d", row), toFloat(s.AmountSen))
		f.SetCellStyle(sheetSettlements, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), moneyStyle)
		row++
	}

	widths := map[string]float64{"A": 18, "B": 18, "C": 18, "D": 14}
	for col, w := range widths {
		if err := f.SetColWidth(sheetSettlements, col, col, w); err != nil {
			return err
		}
	}

	return nil
}

func headerStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: headerFontHex},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{headerFill}},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
	})
}

func toFloat(sen int64) float64 {
	return float64(sen) / 100
}

func participantsText(participants []ParticipantRow) string {
	if len(participants) == 0 {
		return ""
	}
	parts := make([]string, 0, len(participants))
	for _, p := range participants {
		parts = append(parts, fmt.Sprintf("%s: %.2f", p.Name, toFloat(p.AmountSen)))
	}
	return strings.Join(parts, "; ")
}
