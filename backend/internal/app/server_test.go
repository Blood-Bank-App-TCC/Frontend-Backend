package app

import "testing"

func TestParsePositiveInt(t *testing.T) {
	value, err := parsePositiveInt("", 10)
	if err != nil {
		t.Fatalf("expected fallback without error, got %v", err)
	}
	if value != 10 {
		t.Fatalf("expected fallback value 10, got %d", value)
	}

	value, err = parsePositiveInt("3", 10)
	if err != nil {
		t.Fatalf("expected valid integer without error, got %v", err)
	}
	if value != 3 {
		t.Fatalf("expected parsed value 3, got %d", value)
	}

	if _, err := parsePositiveInt("0", 10); err == nil {
		t.Fatal("expected zero to be rejected")
	}
	if _, err := parsePositiveInt("abc", 10); err == nil {
		t.Fatal("expected non-numeric value to be rejected")
	}
}

func TestParseHistoryDate(t *testing.T) {
	value, err := parseHistoryDate("")
	if err != nil {
		t.Fatalf("expected empty date to succeed, got %v", err)
	}
	if value != nil {
		t.Fatal("expected empty date to return nil")
	}

	value, err = parseHistoryDate("2026-06-03")
	if err != nil {
		t.Fatalf("expected valid date to parse, got %v", err)
	}
	if value == nil || value.Format("2006-01-02") != "2026-06-03" {
		t.Fatalf("expected parsed date 2026-06-03, got %#v", value)
	}

	if _, err := parseHistoryDate("06/03/2026"); err == nil {
		t.Fatal("expected invalid date format to fail")
	}
}
