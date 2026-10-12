package document

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func loadFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/sage_invoice.csv")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestParseSage_Fixture(t *testing.T) {
	doc, err := ParseSage(loadFixture(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Reference != "DOC000001" {
		t.Errorf("Reference = %q", doc.Reference)
	}
	wantDate := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	if !doc.Date.Equal(wantDate) {
		t.Errorf("Date = %v, want %v", doc.Date, wantDate)
	}
	// First row is the comment line whose DO_DateLivr is the 1753 sentinel -> fallback to Date.
	if !doc.DisplayedDate.Equal(wantDate) {
		t.Errorf("DisplayedDate = %v, want fallback to %v", doc.DisplayedDate, wantDate)
	}
	if doc.Client.Code != "CLI001" || doc.Client.Kind != ClientCustomer || doc.Client.Name != "ACME CHEMINEES" {
		t.Errorf("Client = %+v", doc.Client)
	}
	if doc.ShipAddress != (Address{Name: "ACME CHEMINEES", Part1: "12 RUE DES TESTS", Zip: "35000", Town: "RENNES", CountryCode: "FR"}) {
		t.Errorf("ShipAddress = %+v", doc.ShipAddress)
	}

	// 6 rows in the file, 1 exact duplicate (same AR_Ref+DL_Ligne) -> 5 items.
	wantKinds := []LineKind{LineComment, LineProduct, LineProduct, LineEmpty, LineProduct}
	var gotKinds []LineKind
	for _, it := range doc.LineItems {
		gotKinds = append(gotKinds, it.Kind)
	}
	if !reflect.DeepEqual(gotKinds, wantKinds) {
		t.Fatalf("kinds = %v, want %v", gotKinds, wantKinds)
	}

	if c := doc.LineItems[0].Comment; c != "REF/ACME" {
		t.Errorf("comment = %q", c)
	}

	p := doc.LineItems[1]
	check(t, "P100 qty", p.Quantity.String(), "1")
	check(t, "P100 unit", p.UnitAmount.String(), "140.87")
	check(t, "P100 tax", p.TaxRate.String(), "20")
	check(t, "P100 purchase", p.PurchaseAmount.String(), "110.04")
	if p.ProductRef != "P100" || p.Description != "TUYAU ACIER 150 - LONGUEUR 1M" {
		t.Errorf("P100 = %+v", p)
	}

	// Fractional quantity: 41.21 / 0.11 = 374.636363... -> 6 decimals.
	f := doc.LineItems[2]
	check(t, "P200 qty", f.Quantity.String(), "0.11")
	check(t, "P200 unit", f.UnitAmount.String(), "374.636364")

	// Free line: total 0, qty 2 -> unit 0, not an error.
	check(t, "P300 unit", doc.LineItems[4].UnitAmount.String(), "0")

	// Linked documents: deduplicated, first-seen order, blanks dropped.
	if want := []string{"DE000001"}; !reflect.DeepEqual(doc.LinkedEstimates, want) {
		t.Errorf("LinkedEstimates = %v", doc.LinkedEstimates)
	}
	if want := []string{"BC000001", "BC000002"}; !reflect.DeepEqual(doc.LinkedOrders, want) {
		t.Errorf("LinkedOrders = %v", doc.LinkedOrders)
	}
	if want := []string{"BL000001", "BL000002"}; !reflect.DeepEqual(doc.LinkedDeliveries, want) {
		t.Errorf("LinkedDeliveries = %v", doc.LinkedDeliveries)
	}
}

func check(t *testing.T, name, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", name, got, want)
	}
}

func TestParseSage_ProspectAndLFLineEndings(t *testing.T) {
	csv := strings.ReplaceAll(loadFixture(t), "\r\n", "\n")
	csv = strings.ReplaceAll(csv, ";ACME CHEMINEES;0;", ";ACME CHEMINEES;1;")
	doc, err := ParseSage(csv)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Client.Kind != ClientProspect {
		t.Errorf("Kind = %q, want prospect", doc.Client.Kind)
	}
}

func TestParseSage_DuplicateColumnsLastWins(t *testing.T) {
	// DO_DateLivr appears twice: first = 1753 sentinel, last = real date.
	// A product row must therefore expose the LAST one. Make it the first data row.
	lines := strings.Split(loadFixture(t), "\r\n")
	csv := strings.Join([]string{lines[0], lines[2]}, "\r\n") // header + P100 row only
	doc, err := ParseSage(csv)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	if !doc.DisplayedDate.Equal(want) {
		t.Errorf("DisplayedDate = %v, want %v (last duplicate column)", doc.DisplayedDate, want)
	}
}

func TestParseSage_Errors(t *testing.T) {
	good := loadFixture(t)
	lines := strings.Split(good, "\r\n")

	tests := []struct {
		name string
		csv  string
	}{
		{"empty", ""},
		{"header only", lines[0]},
		{"missing required column", strings.Replace(good, "DL_Qte", "DL_Quantite", 1)},
		{"mixed documents", strings.Replace(good, "DOC000001;BL000002;BC000002;30/09/2026 00:00:00;5000", "DOC999999;BL000002;BC000002;30/09/2026 00:00:00;5000", 1)},
		{"unknown client type", strings.ReplaceAll(good, ";ACME CHEMINEES;0;", ";ACME CHEMINEES;7;")},
		{"bad number", strings.Replace(good, "110,040000", "abc", 1)},
		// DO_Date is duplicated in the header and the LAST column wins, so every
		// occurrence must be altered for the change to be seen.
		{"garbage date", strings.ReplaceAll(good, "30/09/2026 00:00:00", "30-09-2026")},
		{"missing document date", strings.ReplaceAll(good, "30/09/2026 00:00:00", "01/01/1753 00:00:00")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseSage(tc.csv)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !errors.Is(err, ErrInvalidDocument) {
				t.Errorf("error %q does not wrap ErrInvalidDocument", err)
			}
		})
	}
}

func TestParsersRegistry(t *testing.T) {
	if _, ok := Parsers["sage"]; !ok {
		t.Fatal("sage parser not registered")
	}
	if _, ok := Parsers["nope"]; ok {
		t.Fatal("unexpected parser")
	}
}
