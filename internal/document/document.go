import (
	"time"

	"github.com/shopspring/decimal"
)

type Document struct {
	Reference     string    // doc number (Sage DO_Piece); Sellsy "ident"
	Date          time.Time // document date;
	DisplayedDate time.Time // date sent to Sellsy on validate/updateFields
	Tags          []string  // source tags; the Sellsy handler adds Reference itself for dedup

	Client      Client
	ShipAddress Address // can differ from the client's address

	LineItems []LineItem

	LinkedEstimates  []string // estimates to set "accepted" (from DL_PieceDE)
	LinkedOrders     []string // orders to set "invoiced" (from DL_PieceBC)
	LinkedDeliveries []string // deliveries to set "invoiced" (from DL_PieceBL)
}

type ClientKind string

const (
	ClientCustomer ClientKind = "customer"
	ClientProspect ClientKind = "prospect"
)

type Client struct {
	Code  string     // CT_Num; used for the client lookup
	Kind  ClientKind // chooses Client.getList vs Prospects.getList
	Name  string     // used in the document subject
	Email string     // unused for now, left blank
	Phone string     // unused for now, left blank
}

type Address struct {
	Name        string
	Part1       string
	Zip         string
	Town        string
	CountryCode string // defaulted to "FR" when the source has none
}

type LineKind string

const (
	LineProduct LineKind = "product"
	LineComment LineKind = "comment"
	LineEmpty   LineKind = "empty"
)

type LineItem struct {
	Kind           LineKind
	ProductRef     string          // AR_Ref; used for the Sellsy product lookup
	Description    string          // DL_Design; product name/notes
	Comment        string          // only for LineComment
	Quantity       decimal.Decimal // fractional
	UnitAmount     decimal.Decimal // computed as total HT / quantity
	TaxRate        decimal.Decimal // matched against Sellsy's tax map
	PurchaseAmount decimal.Decimal
}
