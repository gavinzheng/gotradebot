package currency

import (
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// NewPairDelimiter splits the desired currency string at delimeter, the returns
// a Pair struct
func NewPairDelimiter(currencyPair, delimiter string) Pair {
	result := strings.Split(currencyPair, delimiter)
	return Pair{
		Delimiter: delimiter,
		Base:      NewCode(result[0]),
		Quote:     NewCode(result[1]),
	}
}

// NewPairFromStrings returns a CurrencyPair without a delimiter
func NewPairFromStrings(baseCurrency, quoteCurrency string) Pair {
	return Pair{
		Base:  NewCode(baseCurrency),
		Quote: NewCode(quoteCurrency),
	}
}

// NewPair returns a currency pair from currency codes
func NewPair(baseCurrency, quoteCurrency Code) Pair {
	return Pair{
		Base:  baseCurrency,
		Quote: quoteCurrency,
	}
}

// NewPairWithDelimiter returns a CurrencyPair with a delimiter
func NewPairWithDelimiter(base, quote, delimiter string) Pair {
	return Pair{
		Base:      NewCode(base),
		Quote:     NewCode(quote),
		Delimiter: delimiter,
	}
}

// NewPairFromIndex returns a CurrencyPair via a currency string and specific
// index
func NewPairFromIndex(currencyPair, index string) (Pair, error) {
	i := strings.Index(currencyPair, index)
	if i == -1 {
		return Pair{},
			fmt.Errorf("index %s not found in currency pair string", index)
	}
	if i == 0 {
		return NewPairFromStrings(currencyPair[0:len(index)],
				currencyPair[len(index):]),
			nil
	}
	return NewPairFromStrings(currencyPair[0:i], currencyPair[i:]), nil
}

// NewPairFromString converts currency string into a new CurrencyPair
// with or without delimeter
func NewPairFromString(currencyPair string) Pair {
	delimiters := []string{"_", "-", "/"}
	var delimiter string
	for _, x := range delimiters {
		if strings.Contains(currencyPair, x) {
			delimiter = x
			return NewPairDelimiter(currencyPair, delimiter)
		}
	}
	return NewPairFromStrings(currencyPair[0:3], currencyPair[3:])
}

// Pair holds currency pair information
type Pair struct {
	Delimiter string `json:"delimiter"`
	Base      Code   `json:"base"`
	Quote     Code   `json:"quote"`
}

// String returns a currency pair string
func (p Pair) String() string {
	return p.Base.String() + p.Delimiter + p.Quote.String()
}

// Lower converts the pair object to lowercase
func (p Pair) Lower() Pair {
	return Pair{
		Delimiter: p.Delimiter,
		Base:      p.Base.Lower(),
		Quote:     p.Quote.Lower(),
	}
}

// Upper converts the pair object to uppercase
func (p Pair) Upper() Pair {
	return Pair{
		Delimiter: p.Delimiter,
		Base:      p.Base.Upper(),
		Quote:     p.Quote.Upper(),
	}
}

// UnmarshalJSON comforms type to the umarshaler interface
func (p *Pair) UnmarshalJSON(d []byte) error {
	var pair string
	err := common.JSONDecode(d, &pair)
	if err != nil {
		return err
	}

	*p = NewPairFromString(pair)
	return nil
}

// MarshalJSON conforms type to the marshaler interface
func (p Pair) MarshalJSON() ([]byte, error) {
	return common.JSONEncode(p.String())
}

// Format changes the currency based on user preferences overriding the default
// String() display
func (p Pair) Format(delimiter string, uppercase bool) Pair {
	p.Delimiter = delimiter

	if uppercase {
		return p.Upper()
	}
	return p.Lower()
}

// Equal compares two currency pairs and returns whether or not they are equal
func (p Pair) Equal(cPair Pair) bool {
	return p.Base.Item == cPair.Base.Item && p.Quote.Item == cPair.Quote.Item
}

// EqualIncludeReciprocal compares two currency pairs and returns whether or not
// they are the same including reciprocal currencies.
func (p Pair) EqualIncludeReciprocal(cPair Pair) bool {
	if p.Base.Item == cPair.Base.Item &&
		p.Quote.Item == cPair.Quote.Item ||
		p.Base.Item == cPair.Quote.Item &&
			p.Quote.Item == cPair.Base.Item {
		return true
	}
	return false
}

// IsCryptoPair checks to see if the pair is a crypto pair e.g. BTCLTC
func (p Pair) IsCryptoPair() bool {
	return storage.IsCryptocurrency(p.Base) &&
		storage.IsCryptocurrency(p.Quote)
}

// IsCryptoFiatPair checks to see if the pair is a crypto fiat pair e.g. BTCUSD
func (p Pair) IsCryptoFiatPair() bool {
	return storage.IsCryptocurrency(p.Base) &&
		storage.IsFiatCurrency(p.Quote) ||
		storage.IsFiatCurrency(p.Base) &&
			storage.IsCryptocurrency(p.Quote)
}

// IsFiatPair checks to see if the pair is a fiat pair e.g. EURUSD
func (p Pair) IsFiatPair() bool {
	return storage.IsFiatCurrency(p.Base) && storage.IsFiatCurrency(p.Quote)
}

// IsInvalid checks invalid pair if base and quote are the same
func (p Pair) IsInvalid() bool {
	return p.Base.Item == p.Quote.Item
}

// Swap turns the currency pair into its reciprocal
func (p Pair) Swap() Pair {
	p.Base, p.Quote = p.Quote, p.Base
	return p
}

// IsEmpty returns whether or not the pair is empty or is missing a currency
// code
func (p Pair) IsEmpty() bool {
	return p.Base.IsEmpty() || p.Quote.IsEmpty()
}

// ContainsCurrency checks to see if a pair contains a specific currency
func (p Pair) ContainsCurrency(c Code) bool {
	return p.Base.Item == c.Item || p.Quote.Item == c.Item
}
