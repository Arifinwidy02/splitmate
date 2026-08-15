package report

import (
	"bytes"
	"strconv"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

func TestRenderXLSX(t *testing.T) {
	report := &Report{
		GroupName:     "Bali Trip",
		GroupCurrency: "IDR",
		GeneratedAt:   time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC),
		Balances: []BalanceRow{
			{Name: "Arifin", BalanceSen: 30000000},
			{Name: "Budi", BalanceSen: -10000000},
			{Name: "Citra", BalanceSen: -20000000},
		},
		Suggestions: []SuggestionRow{
			{FromName: "Citra", ToName: "Arifin", AmountSen: 20000000},
			{FromName: "Budi", ToName: "Arifin", AmountSen: 10000000},
		},
		Expenses: []ExpenseRow{
			{
				Description: "Dinner",
				Category:    "Food & Drinks",
				PaidByName:  "Arifin",
				ExpenseDate: time.Date(2026, 8, 10, 19, 0, 0, 0, time.UTC),
				AmountSen:   60000000,
				Participants: []ParticipantRow{
					{Name: "Arifin", AmountSen: 20000000},
					{Name: "Budi", AmountSen: 20000000},
					{Name: "Citra", AmountSen: 20000000},
				},
			},
			{
				Description: "Taxi",
				Category:    "Transportation",
				PaidByName:  "Budi",
				ExpenseDate: time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC),
				AmountSen:   30000000,
				Note:        "airport run",
				Participants: []ParticipantRow{
					{Name: "Arifin", AmountSen: 10000000},
					{Name: "Citra", AmountSen: 20000000},
				},
			},
		},
		Settlements: []SettlementRow{
			{
				PayerName:    "Budi",
				ReceiverName: "Arifin",
				AmountSen:    10000000,
				SettledAt:    time.Date(2026, 8, 12, 15, 0, 0, 0, time.UTC),
			},
		},
	}

	data, err := RenderXLSX(report)
	if err != nil {
		t.Fatalf("render xlsx: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("open rendered xlsx: %v", err)
	}
	defer f.Close()

	wantSheets := []string{"Summary", "Expenses", "Settlements"}
	gotSheets := f.GetSheetList()
	if len(gotSheets) != len(wantSheets) {
		t.Fatalf("expected %d sheets, got %d: %v", len(wantSheets), len(gotSheets), gotSheets)
	}
	for i, want := range wantSheets {
		if gotSheets[i] != want {
			t.Errorf("sheet %d = %q, want %q", i, gotSheets[i], want)
		}
	}

	checkCell(t, f, "Summary", "A1", "SplitMate Report")
	checkCell(t, f, "Summary", "A2", "Group: Bali Trip")
	checkCell(t, f, "Summary", "A3", "Currency: IDR")
	checkCell(t, f, "Summary", "A6", "Balances")
	checkCell(t, f, "Summary", "A7", "Member")
	checkCell(t, f, "Summary", "B7", "Balance")
	checkCell(t, f, "Summary", "A8", "Arifin")
	checkCell(t, f, "Summary", "A9", "Budi")
	checkCell(t, f, "Summary", "B8", float64(30000000)/100)
	checkCell(t, f, "Summary", "A10", "Citra")
	checkCell(t, f, "Summary", "A12", "Settlement suggestions")
	checkCell(t, f, "Summary", "A13", "From")
	checkCell(t, f, "Summary", "B13", "To")
	checkCell(t, f, "Summary", "C14", float64(20000000)/100)

	checkCell(t, f, "Expenses", "A1", "Date")
	checkCell(t, f, "Expenses", "B2", "Dinner")
	checkCell(t, f, "Expenses", "D2", "Arifin")
	checkCell(t, f, "Expenses", "E2", float64(60000000)/100)
	checkCell(t, f, "Expenses", "F2", "Arifin: 200000.00; Budi: 200000.00; Citra: 200000.00")
	checkCell(t, f, "Expenses", "G3", "airport run")
	checkCell(t, f, "Expenses", "B4", "TOTAL")
	checkCell(t, f, "Expenses", "E4", float64(90000000)/100)

	checkCell(t, f, "Settlements", "A1", "Date")
	checkCell(t, f, "Settlements", "B2", "Budi")
	checkCell(t, f, "Settlements", "C2", "Arifin")
	checkCell(t, f, "Settlements", "D2", float64(10000000)/100)
}

func TestRenderXLSXEmptyReport(t *testing.T) {
	data, err := RenderXLSX(&Report{
		GroupName:     "Empty Group",
		GroupCurrency: "USD",
		GeneratedAt:   time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("render empty xlsx: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("open rendered xlsx: %v", err)
	}
	defer f.Close()

	checkCell(t, f, "Summary", "A2", "Group: Empty Group")
}

func checkCell(t *testing.T, f *excelize.File, sheet, cell string, want any) {
	t.Helper()

	if wantStr, ok := want.(string); ok {
		got, err := f.GetCellValue(sheet, cell)
		if err != nil {
			t.Fatalf("read %s!%s: %v", sheet, cell, err)
		}
		if got != wantStr {
			t.Errorf("%s!%s = %q, want %q", sheet, cell, got, wantStr)
		}
		return
	}

	wantFloat, ok := want.(float64)
	if !ok {
		t.Fatalf("bad test expectation type for %s!%s", sheet, cell)
	}

	got, err := f.GetCellValue(sheet, cell, excelize.Options{RawCellValue: true})
	if err != nil {
		t.Fatalf("read numeric %s!%s: %v", sheet, cell, err)
	}
	gotFloat, err := strconv.ParseFloat(got, 64)
	if err != nil {
		t.Fatalf("parse %s!%s value %q: %v", sheet, cell, got, err)
	}
	if gotFloat != wantFloat {
		t.Errorf("%s!%s = %v, want %v", sheet, cell, gotFloat, wantFloat)
	}
}

func TestSafeFilename(t *testing.T) {
	cases := map[string]string{
		"Bali Trip":     "Bali-Trip",
		"Trip/2026 (a)": "Trip-2026-a-",
		"100% Real!":    "100-Real-",
		"simple":        "simple",
		"a_b-c.d":       "a_b-c.d",
	}
	for in, want := range cases {
		if got := safeFilename(in); got != want {
			t.Errorf("safeFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestToFloat(t *testing.T) {
	if got := toFloat(123456); got != 1234.56 {
		t.Errorf("toFloat(123456) = %v, want 1234.56", got)
	}
	if got := toFloat(-500); got != -5.00 {
		t.Errorf("toFloat(-500) = %v, want -5.00", got)
	}
	if got := toFloat(0); got != 0 {
		t.Errorf("toFloat(0) = %v, want 0", got)
	}
}
