package currency

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency/coinmarketcap"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// CurrencyFileUpdateDelay defines the rate at which the currency.json file is
// updated
const (
	DefaultCurrencyFileDelay    = 168 * time.Hour
	DefaultForeignExchangeDelay = 1 * time.Minute
)

func init() {
	storage.SetDefaults()
}

// storage is an overarching type that keeps track of and updates currency,
// currency exchange rates and pairs
var storage Storage

// Storage contains the loaded storage currencies supported on available crypto
// or fiat marketplaces
// NOTE: All internal currencies are upper case
type Storage struct {
	// FiatCurrencies defines the running fiat currencies in the currency
	// storage
	fiatCurrencies Currencies

	// Cryptocurrencies defines the running cryptocurrencies in the currency
	// storage
	cryptocurrencies Currencies

	// CurrencyCodes is a full basket of currencies either crypto, fiat, ico or
	// contract being tracked by the currency storage system
	currencyCodes BaseCodes

	// Main converting currency
	baseCurrency Code

	// FXRates defines a protected conversion rate map
	fxRates ConversionRates

	// DefaultBaseCurrency is the base currency used for conversion
	defaultBaseCurrency Code

	// DefaultFiatCurrencies has the default minimum of FIAT values
	defaultFiatCurrencies Currencies

	// DefaultCryptoCurrencies has the default minimum of crytpocurrency values
	defaultCryptoCurrencies Currencies

	// FiatExchangeMarkets defines an interface to access FX data for fiat
	// currency rates
	fiatExchangeMarkets *forexprovider.ForexProviders

	// CurrencyAnalysis defines a full market analysis suite to receieve and
	// define different fiat currencies, cryptocurrencies and markets
	currencyAnalysis *coinmarketcap.Coinmarketcap

	// Path defines the main folder to dump and find currency JSON
	path string

	// Update delay variables
	currencyFileUpdateDelay    time.Duration
	foreignExchangeUpdateDelay time.Duration

	mtx            sync.Mutex
	wg             sync.WaitGroup
	shutdownC      chan struct{}
	updaterRunning bool
	Verbose        bool
}

// SetDefaults sets storage defaults for basic package functionality
func (s *Storage) SetDefaults() {
	s.defaultBaseCurrency = USD
	s.baseCurrency = s.defaultBaseCurrency
	s.SetDefaultFiatCurrencies(USD, AUD, EUR, CNY)
	s.SetDefaultCryptocurrencies(BTC, LTC, ETH, DOGE, DASH, XRP, XMR)
	s.SetupConversionRates()
	s.fiatExchangeMarkets = forexprovider.NewDefaultFXProvider()
}

// RunUpdater runs the foreign exchange updater service. This will set up a JSON
// dump file and keep foreign exchange rates updated as fast as possible without
// triggering rate limiters, it will also run a full cryptocurrency check
// through coin market cap and expose analytics for exchange services
func (s *Storage) RunUpdater(overrides BotOverrides, settings *MainConfiguration, filePath string, verbose bool) error {
	s.mtx.Lock()

	if !settings.Cryptocurrencies.HasData() {
		s.mtx.Unlock()
		return errors.New("currency storage error, no cryptocurrencies loaded")
	}
	s.cryptocurrencies = settings.Cryptocurrencies

	if settings.FiatDisplayCurrency.IsEmpty() {
		s.mtx.Unlock()
		return errors.New("currency storage error, no fiat display currency set in config")
	}
	s.baseCurrency = settings.FiatDisplayCurrency
	log.Debugf("Fiat display currency: %s.", s.baseCurrency)

	if settings.CryptocurrencyProvider.Enabled {
		log.Debugf("Setting up currency analysis system with Coinmarketcap...")
		c := &coinmarketcap.Coinmarketcap{}
		c.SetDefaults()
		c.Setup(coinmarketcap.Settings{
			Name:        settings.CryptocurrencyProvider.Name,
			Enabled:     true,
			AccountPlan: settings.CryptocurrencyProvider.AccountPlan,
			APIkey:      settings.CryptocurrencyProvider.APIkey,
			Verbose:     settings.CryptocurrencyProvider.Verbose,
		})

		s.currencyAnalysis = c
	}

	if filePath == "" {
		s.mtx.Unlock()
		return errors.New("currency package runUpdater error filepath not set")
	}

	s.path = filePath + common.GetOSPathSlash() + "currency.json"

	if settings.CurrencyDelay.Nanoseconds() == 0 {
		s.currencyFileUpdateDelay = DefaultCurrencyFileDelay
	} else {
		s.currencyFileUpdateDelay = settings.CurrencyDelay
	}

	if settings.FxRateDelay.Nanoseconds() == 0 {
		s.foreignExchangeUpdateDelay = DefaultForeignExchangeDelay
	} else {
		s.foreignExchangeUpdateDelay = settings.FxRateDelay
	}

	var fxSettings []base.Settings
	for i := range settings.ForexProviders {
		switch settings.ForexProviders[i].Name {
		case "CurrencyConverter":
			if overrides.FxCurrencyConverter ||
				settings.ForexProviders[i].Enabled {
				settings.ForexProviders[i].Enabled = true
				fxSettings = append(fxSettings,
					base.Settings(settings.ForexProviders[i]))
			}

		case "CurrencyLayer":
			if overrides.FxCurrencyLayer || settings.ForexProviders[i].Enabled {
				settings.ForexProviders[i].Enabled = true
				fxSettings = append(fxSettings,
					base.Settings(settings.ForexProviders[i]))
			}

		case "Fixer":
			if overrides.FxFixer || settings.ForexProviders[i].Enabled {
				settings.ForexProviders[i].Enabled = true
				fxSettings = append(fxSettings,
					base.Settings(settings.ForexProviders[i]))
			}

		case "OpenExchangeRates":
			if overrides.FxOpenExchangeRates ||
				settings.ForexProviders[i].Enabled {
				settings.ForexProviders[i].Enabled = true
				fxSettings = append(fxSettings,
					base.Settings(settings.ForexProviders[i]))
			}

		case "ExchangeRates":
			// TODO ADD OVERRIDE
			if settings.ForexProviders[i].Enabled {
				settings.ForexProviders[i].Enabled = true
				fxSettings = append(fxSettings,
					base.Settings(settings.ForexProviders[i]))
			}
		}
	}

	if len(fxSettings) != 0 {
		var err error
		s.fiatExchangeMarkets, err = forexprovider.StartFXService(fxSettings)
		if err != nil {
			s.mtx.Unlock()
			return err
		}

		log.Debugf("Primary foreign exchange conversion provider %s enabled",
			s.fiatExchangeMarkets.Primary.Provider.GetName())

		for i := range s.fiatExchangeMarkets.Support {
			log.Debugf("Support forex conversion provider %s enabled",
				s.fiatExchangeMarkets.Support[i].Provider.GetName())
		}

		// Mutex present in this go routine to lock down retrieving rate data
		// until this system initially updates
		go s.ForeignExchangeUpdater()
	} else {
		log.Warnf("No foreign exchange providers enabled in config.json")
		s.mtx.Unlock()
	}

	return nil
}

// SetupConversionRates sets default conversion rate values
func (s *Storage) SetupConversionRates() {
	s.fxRates = ConversionRates{
		m: make(map[*Item]map[*Item]*float64),
	}
}

// SetDefaultFiatCurrencies assigns the default fiat currency list and adds it
// to the running list
func (s *Storage) SetDefaultFiatCurrencies(c ...Code) {
	for _, currency := range c {
		s.defaultFiatCurrencies = append(s.defaultFiatCurrencies, currency)
		s.fiatCurrencies = append(s.fiatCurrencies, currency)
	}
}

// SetDefaultCryptocurrencies assigns the default cryptocurrency list and adds
// it to the running list
func (s *Storage) SetDefaultCryptocurrencies(c ...Code) {
	for _, currency := range c {
		s.defaultCryptoCurrencies = append(s.defaultCryptoCurrencies, currency)
		s.cryptocurrencies = append(s.cryptocurrencies, currency)
	}
}

// SetupForexProviders sets up a new instance of the forex providers
func (s *Storage) SetupForexProviders(setting ...base.Settings) error {
	addr, err := forexprovider.StartFXService(setting)
	if err != nil {
		return err
	}

	s.fiatExchangeMarkets = addr
	return nil
}

// ForeignExchangeUpdater is a routine that seeds foreign exchange rate and keeps
// updated as fast as possible
func (s *Storage) ForeignExchangeUpdater() {
	log.Debugf("Foreign exchange updater started, seeding FX rate list..")

	s.wg.Add(1)
	defer s.wg.Done()

	err := s.SeedCurrencyAnalysisData()
	if err != nil {
		log.Error(err)
	}

	err = s.SeedForeignExchangeRates()
	if err != nil {
		log.Error(err)
	}

	// Unlock main rate retrieval mutex so all routines waiting can get access
	// to data
	s.mtx.Unlock()

	// Set tickers to client defined rates or defaults
	SeedForeignExchangeTick := time.NewTicker(s.foreignExchangeUpdateDelay)
	SeedCurrencyAnalysisTick := time.NewTicker(s.currencyFileUpdateDelay)
	defer SeedForeignExchangeTick.Stop()
	defer SeedCurrencyAnalysisTick.Stop()

	for {
		select {
		case <-s.shutdownC:
			return

		case <-SeedForeignExchangeTick.C:
			err := s.SeedForeignExchangeRates()
			if err != nil {
				log.Error(err)
			}

		case <-SeedCurrencyAnalysisTick.C:
			err := s.SeedCurrencyAnalysisData()
			if err != nil {
				log.Error(err)
			}
		}
	}
}

// SeedCurrencyAnalysisData sets a new instance of a coinmarketcap data.
func (s *Storage) SeedCurrencyAnalysisData() error {
	b, err := common.ReadFile(s.path)
	if err != nil {
		err = s.FetchCurrencyAnalysisData()
		if err != nil {
			return s.WriteCurrencyDataToFile(s.path, false)
		}

		return s.WriteCurrencyDataToFile(s.path, true)
	}

	var fromFile File
	err = common.JSONDecode(b, &fromFile)
	if err != nil {
		return err
	}

	err = s.LoadFileCurrencyData(&fromFile)
	if err != nil {
		return err
	}

	// Based on update delay update the file
	if fromFile.LastMainUpdate.After(fromFile.LastMainUpdate.Add(s.currencyFileUpdateDelay)) ||
		fromFile.LastMainUpdate.IsZero() {
		err = s.FetchCurrencyAnalysisData()
		if err != nil {
			return s.WriteCurrencyDataToFile(s.path, false)
		}

		return s.WriteCurrencyDataToFile(s.path, true)
	}

	return nil
}

// FetchCurrencyAnalysisData fetches a new fresh batch of currency data and
// loads it into memory
func (s *Storage) FetchCurrencyAnalysisData() error {
	if s.currencyAnalysis == nil {
		log.Warn("Currency analysis system offline please set api keys for coinmarketcap")
		return errors.New("currency analysis system offline")
	}

	return s.UpdateCurrencies()
}

// WriteCurrencyDataToFile writes the full currency data to a designated file
func (s *Storage) WriteCurrencyDataToFile(path string, mainUpdate bool) error {
	data, err := s.currencyCodes.GetFullCurrencyData()
	if err != nil {
		return err
	}

	if mainUpdate {
		t := time.Now()
		data.LastMainUpdate = t
		s.currencyCodes.LastMainUpdate = t
	}

	var encoded []byte
	encoded, err = json.MarshalIndent(data, "", " ")
	if err != nil {
		return err
	}

	return common.WriteFile(path, encoded)
}

// LoadFileCurrencyData loads currencies into the currency codes
func (s *Storage) LoadFileCurrencyData(f *File) error {

	for i := range f.Contracts {
		err := s.currencyCodes.LoadItem(&f.Contracts[i])
		if err != nil {
			return err
		}
	}

	for i := range f.Cryptocurrency {
		err := s.currencyCodes.LoadItem(&f.Cryptocurrency[i])
		if err != nil {
			return err
		}
	}

	for i := range f.Token {
		err := s.currencyCodes.LoadItem(&f.Token[i])
		if err != nil {
			return err
		}
	}

	for i := range f.FiatCurrency {
		err := s.currencyCodes.LoadItem(&f.FiatCurrency[i])
		if err != nil {
			return err
		}
	}

	for i := range f.UnsetCurrency {
		err := s.currencyCodes.LoadItem(&f.UnsetCurrency[i])
		if err != nil {
			return err
		}
	}

	s.currencyCodes.LastMainUpdate = f.LastMainUpdate

	return nil
}

// UpdateCurrencies updates currency roll and information using coin market cap
func (s *Storage) UpdateCurrencies() error {
	m, err := s.currencyAnalysis.GetCryptocurrencyIDMap()
	if err != nil {
		return err
	}

	for x := range m {
		if m[x].IsActive != 1 {
			continue
		}

		if m[x].Platform.Symbol != "" {
			err := s.currencyCodes.UpdateToken(m[x].Name,
				m[x].Symbol,
				m[x].Platform.Symbol,
				m[x].ID)
			if err != nil {
				return err
			}
			continue
		}

		err := s.currencyCodes.UpdateCryptocurrency(m[x].Name,
			m[x].Symbol,
			m[x].ID)
		if err != nil {
			return err
		}
	}
	return nil
}

// SeedForeignExchangeRatesByCurrencies seeds the foreign exchange rates by
// currencies supplied
func (s *Storage) SeedForeignExchangeRatesByCurrencies(c Currencies) error {
	s.fxRates.mtx.Lock()
	defer s.fxRates.mtx.Unlock()
	rates, err := s.fiatExchangeMarkets.GetCurrencyData(s.baseCurrency.String(),
		c.Strings())
	if err != nil {
		return err
	}
	return s.updateExchangeRates(rates)
}

// SeedForeignExchangeRate returns a singular exchange rate
func (s *Storage) SeedForeignExchangeRate(from, to Code) (map[string]float64, error) {
	return s.fiatExchangeMarkets.GetCurrencyData(from.String(),
		[]string{to.String()})
}

// GetDefaultForeignExchangeRates returns foreign exchange rates based off
// default fiat currencies.
func (s *Storage) GetDefaultForeignExchangeRates() (Conversions, error) {
	if !s.updaterRunning {
		err := s.SeedDefaultForeignExchangeRates()
		if err != nil {
			return nil, err
		}
	}
	return s.fxRates.GetFullRates(), nil
}

// SeedDefaultForeignExchangeRates seeds the default foreign exchange rates
func (s *Storage) SeedDefaultForeignExchangeRates() error {
	s.fxRates.mtx.Lock()
	defer s.fxRates.mtx.Unlock()
	rates, err := s.fiatExchangeMarkets.GetCurrencyData(
		s.defaultBaseCurrency.String(),
		s.defaultFiatCurrencies.Strings())
	if err != nil {
		return err
	}
	return s.updateExchangeRates(rates)
}

// GetExchangeRates returns storage seeded exchange rates
func (s *Storage) GetExchangeRates() (Conversions, error) {
	if !s.updaterRunning {
		err := s.SeedForeignExchangeRates()
		if err != nil {
			return nil, err
		}
	}
	return s.fxRates.GetFullRates(), nil
}

// SeedForeignExchangeRates seeds the foreign exchange rates from storage config
// currencies
func (s *Storage) SeedForeignExchangeRates() error {
	s.fxRates.mtx.Lock()
	defer s.fxRates.mtx.Unlock()
	rates, err := s.fiatExchangeMarkets.GetCurrencyData(
		s.baseCurrency.String(),
		s.fiatCurrencies.Strings())
	if err != nil {
		return err
	}
	return s.updateExchangeRates(rates)
}

// UpdateForeignExchangeRates sets exchange rates on the FX map
func (s *Storage) updateExchangeRates(m map[string]float64) error {
	err := s.fxRates.Update(m)
	if err != nil {
		return err
	}

	if s.path != "" {
		return s.WriteCurrencyDataToFile(s.path, false)
	}
	return nil
}

// SetupCryptoProvider sets congiguration parameters and starts a new instance
// of the currency analyser
func (s *Storage) SetupCryptoProvider(settings coinmarketcap.Settings) error {
	if settings.APIkey == "" ||
		settings.APIkey == "key" ||
		settings.AccountPlan == "" ||
		settings.AccountPlan == "accountPlan" {
		return errors.New("currencyprovider error api key or plan not set in config.json")
	}

	s.currencyAnalysis = new(coinmarketcap.Coinmarketcap)
	s.currencyAnalysis.SetDefaults()
	s.currencyAnalysis.Setup(settings)

	return nil
}

// GetTotalMarketCryptocurrencies returns the total seeded market
// cryptocurrencies
func (s *Storage) GetTotalMarketCryptocurrencies() (Currencies, error) {
	if !s.currencyCodes.HasData() {
		return nil, errors.New("market currency codes not populated")
	}
	return s.currencyCodes.GetCurrencies(), nil
}

// IsDefaultCurrency returns if a currency is a default currency
func (s *Storage) IsDefaultCurrency(c Code) bool {
	t, _ := GetTranslation(c)
	for _, d := range s.defaultFiatCurrencies {
		if d.Match(c) || d.Match(t) {
			return true
		}
	}
	return false
}

// IsDefaultCryptocurrency returns if a cryptocurrency is a default
// cryptocurrency
func (s *Storage) IsDefaultCryptocurrency(c Code) bool {
	t, _ := GetTranslation(c)
	for _, d := range s.defaultCryptoCurrencies {
		if d.Match(c) || d.Match(t) {
			return true
		}
	}
	return false
}

// IsFiatCurrency returns if a currency is part of the enabled fiat currency
// list
func (s *Storage) IsFiatCurrency(c Code) bool {
	if c.Item.Role != Unset {
		return c.Item.Role == Fiat
	}

	if c == USDT {
		return false
	}

	t, _ := GetTranslation(c)
	for _, d := range s.fiatCurrencies {
		if d.Match(c) || d.Match(t) {
			return true
		}
	}

	return false
}

// IsCryptocurrency returns if a cryptocurrency is part of the enabled
// cryptocurrency list
func (s *Storage) IsCryptocurrency(c Code) bool {
	if c.Item.Role != Unset {
		return c.Item.Role == Cryptocurrency
	}

	if c == USD {
		return false
	}

	t, _ := GetTranslation(c)
	for _, d := range s.cryptocurrencies {
		if d.Match(c) || d.Match(t) {
			return true
		}
	}

	return false
}

// ValidateCode validates string against currency list and returns a currency
// code
func (s *Storage) ValidateCode(newCode string) Code {
	return s.currencyCodes.Register(newCode)
}

// ValidateFiatCode validates a fiat currency string and returns a currency
// code
func (s *Storage) ValidateFiatCode(newCode string) (Code, error) {
	c, err := s.currencyCodes.RegisterFiat(newCode)
	if err != nil {
		return c, err
	}
	if !s.fiatCurrencies.Contains(c) {
		s.fiatCurrencies = append(s.fiatCurrencies, c)
	}
	return c, nil
}

// ValidateCryptoCode validates a cryptocurrency string and returns a currency
// code
// TODO: Update and add in RegisterCrypto member func
func (s *Storage) ValidateCryptoCode(newCode string) Code {
	c := s.currencyCodes.Register(newCode)
	if !s.cryptocurrencies.Contains(c) {
		s.cryptocurrencies = append(s.cryptocurrencies, c)
	}
	return c
}

// UpdateBaseCurrency changes base currency
func (s *Storage) UpdateBaseCurrency(c Code) error {
	if c.IsFiatCurrency() {
		s.baseCurrency = c
		return nil
	}
	return fmt.Errorf("currency %s not fiat failed to set currency", c)
}

// GetCryptocurrencies returns the cryptocurrency list
func (s *Storage) GetCryptocurrencies() Currencies {
	return s.cryptocurrencies
}

// GetDefaultCryptocurrencies returns a list of default cryptocurrencies
func (s *Storage) GetDefaultCryptocurrencies() Currencies {
	return s.defaultCryptoCurrencies
}

// GetFiatCurrencies returns the fiat currencies list
func (s *Storage) GetFiatCurrencies() Currencies {
	return s.fiatCurrencies
}

// GetDefaultFiatCurrencies returns the default fiat currencies list
func (s *Storage) GetDefaultFiatCurrencies() Currencies {
	return s.defaultFiatCurrencies
}

// GetDefaultBaseCurrency returns the default base currency
func (s *Storage) GetDefaultBaseCurrency() Code {
	return s.defaultBaseCurrency
}

// GetBaseCurrency returns the current storage base currency
func (s *Storage) GetBaseCurrency() Code {
	return s.baseCurrency
}

// UpdateEnabledCryptoCurrencies appends new cryptocurrencies to the enabled
// currency list
func (s *Storage) UpdateEnabledCryptoCurrencies(c Currencies) {
	for _, i := range c {
		if !s.cryptocurrencies.Contains(i) {
			s.cryptocurrencies = append(s.cryptocurrencies, i)
		}
	}
}

// UpdateEnabledFiatCurrencies appends new fiat currencies to the enabled
// currency list
func (s *Storage) UpdateEnabledFiatCurrencies(c Currencies) {
	for _, i := range c {
		if !s.fiatCurrencies.Contains(i) && !s.cryptocurrencies.Contains(i) {
			s.fiatCurrencies = append(s.fiatCurrencies, i)
		}
	}
}

// ConvertCurrency for example converts $1 USD to the equivalent Japanese Yen
// or vice versa.
func (s *Storage) ConvertCurrency(amount float64, from, to Code) (float64, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if !s.fxRates.HasData() {
		err := s.SeedDefaultForeignExchangeRates()
		if err != nil {
			return 0, err
		}
	}

	r, err := s.fxRates.GetRate(from, to)
	if err != nil {
		return 0, err
	}

	return r * amount, nil
}

// GetStorageRate returns the rate of the conversion value
func (s *Storage) GetStorageRate(from, to Code) (float64, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if !s.fxRates.HasData() {
		err := s.SeedDefaultForeignExchangeRates()
		if err != nil {
			return 0, err
		}
	}

	return s.fxRates.GetRate(from, to)
}

// NewConversion returns a new conversion object that has a pointer to a related
// rate with its inversion.
func (s *Storage) NewConversion(from, to Code) (Conversion, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if !s.fxRates.HasData() {
		err := storage.SeedDefaultForeignExchangeRates()
		if err != nil {
			return Conversion{}, err
		}
	}
	return s.fxRates.Register(from, to)
}

// IsVerbose returns if the storage is in verbose mode
func (s *Storage) IsVerbose() bool {
	return s.Verbose
}
