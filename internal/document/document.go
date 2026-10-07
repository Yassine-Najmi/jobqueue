package document

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

var ErrInvalidDocument = errors.New("invalid document")

type Document struct {
	Reference     string    // doc number
	Date          time.Time // document date;
	DisplayedDate time.Time // delivery date
	Tags          []string  // source tags;

	Client      Client
	ShipAddress Address

	LineItems []LineItem

	LinkedEstimates  []string
	LinkedOrders     []string
	LinkedDeliveries []string
}

type ClientKind string

const (
	ClientCustomer ClientKind = "customer"
	ClientProspect ClientKind = "prospect"
)

type Client struct {
	Code  string // ERP client number
	Kind  ClientKind
	Name  string
	Email string
	Phone string
}

type Address struct {
	Name        string
	Part1       string
	Zip         string
	Town        string
	CountryCode string // ISO alpha-2
}

type LineKind string

const (
	LineProduct LineKind = "product"
	LineComment LineKind = "comment"
	LineEmpty   LineKind = "empty"
)

type LineItem struct {
	Kind           LineKind
	ProductRef     string
	Description    string
	Comment        string
	Quantity       decimal.Decimal
	UnitAmount     decimal.Decimal // total HT / quantity, 6 decimals
	TaxRate        decimal.Decimal // percent, e.g. 20
	PurchaseAmount decimal.Decimal
}

type ParseFunc func(raw string) (Document, error)

var Parsers = map[string]ParseFunc{
	"sage": ParseSage,
}
