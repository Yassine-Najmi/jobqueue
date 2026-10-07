package document

import (
	"encoding/csv"
	"fmt"
	"strings"
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

	return Document{}, nil
}

func invalidf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidDocument, fmt.Sprintf(format, args...))
}
