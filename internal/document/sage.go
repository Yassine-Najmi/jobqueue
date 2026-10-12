package document

import (
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const sageMissingDate = "01/01/1753"

var sageRequired = []string{
	"DO_Piece", "DO_Date", "CT_Num", "CT_Type", "CT_Intitule",
	"AR_Ref", "DL_Ligne", "DL_Design", "DL_Qte", "DL_MontantHT", "DL_Taxe1", "AR_PrixAch",
}

func ParseSage(raw string) (Document, error) {
	raw = strings.TrimPrefix(raw, "\xef\xbb\xbf")

	r := csv.NewReader(strings.NewReader(raw))
	r.Comma = ';'
	r.LazyQuotes = true    // a stray " shouldn't abort the file
	r.FieldsPerRecord = -1 // don't force equal row widths

	records, err := r.ReadAll()
	if err != nil {
		return Document{}, invalidf("cannot read CSV: %v", err)
	}
	if len(records) < 2 {
		return Document{}, invalidf("need a header and at least one data row, got %d row(s)", len(records))
	}

	col := make(map[string]int, len(records[0]))
	for i, name := range records[0] {
		col[strings.TrimSpace(name)] = i // Build a column-name -> index lookup.
	}
	for _, sageName := range sageRequired {
		if _, ok := col[sageName]; !ok {
			return Document{}, invalidf("missing column %q", sageName)
		}
	}

	get := func(row []string, name string) string {
		i, ok := col[name]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}

	rows := records[1:]
	first := rows[0]

	doc, err := sageHeader(first, get)
	if err != nil {
		return Document{}, nil
	}

	seen := make(map[string]bool)
	for n, row := range rows {
		rowNo := n + 1
		if p := get(row, "DO_Piece"); p != doc.Reference {
			return Document{}, invalidf("row %d belongs to document %q, expected %q", rowNo, p, doc.Reference)
		}

		key := get(row, "AR_Ref") + "_" + get(row, "DL_Ligne")
		if seen[key] {
			continue
		}
		seen[key] = true

		item, err := sageLine(row, get)
		if err != nil {
			return Document{}, invalidf("row %d: %v", rowNo, err)
		}
		doc.LineItems = append(doc.LineItems, item)

		doc.LinkedEstimates = appendUnique(doc.LinkedEstimates, get(row, "DL_PieceDE"))
		doc.LinkedOrders = appendUnique(doc.LinkedOrders, get(row, "DL_PieceBC"))
		doc.LinkedDeliveries = appendUnique(doc.LinkedDeliveries, get(row, "DL_PieceBL"))
	}

	return doc, nil
}

type getter func(row []string, name string) string

func sageHeader(row []string, get getter) (Document, error) {
	doc := Document{Reference: get(row, "Do_Piece")}
	if doc.Reference == "" {
		return Document{}, invalidf("empty DO_Piece")
	}

	var err error
	if doc.Date, err = parseSageDate(get(row, "DO_Date")); err != nil {
		return Document{}, invalidf("DO_Date: %v", err)
	}
	if doc.Date.IsZero() {
		return Document{}, invalidf("DO_Date is empty")
	}

	if doc.DisplayedDate, err = parseSageDate(get(row, "DO_DateLivr")); err != nil {
		return Document{}, invalidf("DO_DateLivr: %v", err)
	}
	if doc.DisplayedDate.IsZero() {
		doc.DisplayedDate = doc.Date
	}

	var kind ClientKind
	switch t := get(row, "CT_Type"); t {
	case "0":
		kind = ClientCustomer
	case "1":
		kind = ClientProspect
	default:
		return Document{}, invalidf("unknown CT_Type %q", t)
	}

	doc.Client = Client{
		Code: get(row, "CT_Num"),
		Kind: kind,
		Name: get(row, "CT_Intitule"),
		// Email: ,
		// Phone: ,
	}
	if doc.Client.Code == "" {
		return Document{}, invalidf("empty CT_Num")
	}

	doc.ShipAddress = Address{
		Name:        doc.Client.Name,
		Part1:       get(row, "CT_Adresse"),
		Zip:         get(row, "CT_CodePostal"),
		Town:        get(row, "CT_Ville"),
		CountryCode: "FR",
	}

	return doc, nil
}

// sageLine classifies one row. The rule mirrors the ERP's own convention:
// no product reference + no text = spacer, no reference + text = comment.
func sageLine(row []string, get getter) (LineItem, error) {
	ref, design := get(row, "AR_Ref"), get(row, "DL_Design")
	switch {
	case ref == "" && design == "":
		return LineItem{Kind: LineEmpty}, nil
	case ref == "":
		return LineItem{Kind: LineComment, Comment: design}, nil
	}

	qty, err := parseSageDecimal(get(row, "DL_Qte"))
	if err != nil {
		return LineItem{}, fmt.Errorf("DL_Qte: %w", err)
	}
	total, err := parseSageDecimal(get(row, "DL_MontantHT"))
	if err != nil {
		return LineItem{}, fmt.Errorf("DL_MontantHT: %w", err)
	}
	tax, err := parseSageDecimal(get(row, "DL_Taxe1"))
	if err != nil {
		return LineItem{}, fmt.Errorf("DL_Taxe1: %w", err)
	}
	purchase, err := parseSageDecimal(get(row, "AR_PrixAch"))
	if err != nil {
		return LineItem{}, fmt.Errorf("AR_PrixAch: %w", err)
	}

	// Sage stores the line total
	unit := decimal.Zero
	if !qty.IsZero() {
		unit = total.DivRound(qty, 6)
	}

	return LineItem{
		Kind:           LineProduct,
		ProductRef:     ref,
		Description:    design,
		Quantity:       qty,
		UnitAmount:     unit,
		TaxRate:        tax,
		PurchaseAmount: purchase,
	}, nil
}

// parseSageDate reads "dd/mm/yyyy[ hh:mm:ss]".
// Empty or 01/01/1753 gives the zero time (absent)
func parseSageDate(s string) (time.Time, error) {
	if s == "" || strings.HasPrefix(s, sageMissingDate) {
		return time.Time{}, nil
	}

	if len(s) < 10 {
		return time.Time{}, fmt.Errorf("bad date %q", s)
	}

	t, err := time.Parse("02/01/2006", s[:10]) // time.Parse returns UTC
	if err != nil {
		return time.Time{}, fmt.Errorf("bad date %q", s)
	}
	return t, nil
}

func parseSageDecimal(s string) (decimal.Decimal, error) {
	if s == "" {
		return decimal.Zero, nil
	}
	d, err := decimal.NewFromString(strings.Replace(s, ",", ".", 1))
	if err != nil {
		return decimal.Zero, fmt.Errorf("bad number %q", s)
	}
	return d, nil
}

func appendUnique(list []string, v string) []string {
	if v == "" {
		return list
	}
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

func invalidf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidDocument, fmt.Sprintf(format, args...))
}
