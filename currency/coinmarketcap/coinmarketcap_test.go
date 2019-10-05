package coinmarketcap

import (
	"testing"
	"time"

	log "github.com/thrasher-corp/gocryptotrader/logger"
)

var c Coinmarketcap

// Please set API keys to test endpoint
const (
	apikey              = ""
	apiAccountPlanLevel = ""
)

// Checks credentials but also checks to see if the function can take the
// required account plan level
func areAPICredtionalsSet(minAllowable uint8) bool {
	if apiAccountPlanLevel != "" && apikey != "" {
		if err := c.CheckAccountPlan(minAllowable); err != nil {
			log.Warn("coinmarketpcap test suite - account plan not allowed for function, please review or upgrade plan to test")
			return false
		}
		return true
	}
	return false
}

func TestSetDefaults(t *testing.T) {
	c.SetDefaults()
}

func TestSetup(t *testing.T) {
	c.SetDefaults()

	cfg := Settings{}
	cfg.APIkey = apikey
	cfg.AccountPlan = apiAccountPlanLevel
	cfg.Enabled = true
	cfg.AccountPlan = "basic"

	c.Setup(cfg)
}

func TestCheckAccountPlan(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)

	if areAPICredtionalsSet(Basic) {
		err := c.CheckAccountPlan(Enterprise)
		if err == nil {
			t.Error("Test Failed - CheckAccountPlan() error cannot be nil")
		}

		err = c.CheckAccountPlan(Professional)
		if err == nil {
			t.Error("Test Failed - CheckAccountPlan() error cannot be nil")
		}

		err = c.CheckAccountPlan(Standard)
		if err == nil {
			t.Error("Test Failed - CheckAccountPlan() error cannot be nil")
		}

		err = c.CheckAccountPlan(Hobbyist)
		if err == nil {
			t.Error("Test Failed - CheckAccountPlan() error cannot be nil")
		}

		err = c.CheckAccountPlan(Startup)
		if err == nil {
			t.Error("Test Failed - CheckAccountPlan() error cannot be nil")
		}

		err = c.CheckAccountPlan(Basic)
		if err != nil {
			t.Error("Test Failed - CheckAccountPlan() error", err)
		}
	}
}

func TestGetCryptocurrencyInfo(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetCryptocurrencyInfo(1)
	if areAPICredtionalsSet(Basic) {
		if err != nil {
			t.Error("Test Failed - GetCryptocurrencyInfo() error", err)
		}
	} else {
		if err == nil {
			t.Error("Test Failed - GetCryptocurrencyInfo() error cannot be nil")
		}
	}
}

func TestGetCryptocurrencyIDMap(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetCryptocurrencyIDMap()
	if areAPICredtionalsSet(Basic) {
		if err != nil {
			t.Error("Test Failed - GetCryptocurrencyIDMap() error", err)
		}
	} else {
		if err == nil {
			t.Error("Test Failed - GetCryptocurrencyIDMap() error cannot be nil")
		}
	}
}

func TestGetCryptocurrencyHistoricalListings(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetCryptocurrencyHistoricalListings()
	if err == nil {
		t.Error("Test Failed - GetCryptocurrencyHistoricalListings() error cannot be nil")
	}
}

func TestGetCryptocurrencyLatestListing(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetCryptocurrencyLatestListing(0, 0)
	if areAPICredtionalsSet(Basic) {
		if err != nil {
			t.Error("Test Failed - GetCryptocurrencyLatestListing() error", err)
		}
	} else {
		if err == nil {
			t.Error("Test Failed - GetCryptocurrencyLatestListing() error cannot be nil")
		}
	}
}

func TestGetCryptocurrencyLatestMarketPairs(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetCryptocurrencyLatestMarketPairs(1, 0, 0)
	if areAPICredtionalsSet(Standard) {
		if err != nil {
			t.Error("Test Failed - GetCryptocurrencyLatestMarketPairs() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("Test Failed - GetCryptocurrencyLatestMarketPairs() error cannot be nil")
		}
	}
}

func TestGetCryptocurrencyOHLCHistorical(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetCryptocurrencyOHLCHistorical(1, time.Now(), time.Now())
	if areAPICredtionalsSet(Standard) {
		if err != nil {
			t.Error("Test Failed - GetCryptocurrencyOHLCHistorical() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("Test Failed - GetCryptocurrencyOHLCHistorical() error cannot be nil")
		}
	}
}

func TestGetCryptocurrencyOHLCLatest(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetCryptocurrencyOHLCLatest(1)
	if areAPICredtionalsSet(Startup) {
		if err != nil {
			t.Error("Test Failed - GetCryptocurrencyOHLCLatest() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("Test Failed - GetCryptocurrencyOHLCLatest() error cannot be nil")
		}
	}
}

func TestGetCryptocurrencyLatestQuotes(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetCryptocurrencyLatestQuotes(1)
	if areAPICredtionalsSet(Basic) {
		if err != nil {
			t.Error("Test Failed - GetCryptocurrencyLatestQuotes() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("Test Failed - GetCryptocurrencyLatestQuotes() error cannot be nil")
		}
	}
}

func TestGetCryptocurrencyHistoricalQuotes(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetCryptocurrencyHistoricalQuotes(1, time.Now(), time.Now())
	if areAPICredtionalsSet(Standard) {
		if err != nil {
			t.Error("Test Failed - GetCryptocurrencyHistoricalQuotes() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("Test Failed - GetCryptocurrencyHistoricalQuotes() error cannot be nil")
		}
	}
}

func TestGetExchangeInfo(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetExchangeInfo(1)
	if areAPICredtionalsSet(Startup) {
		if err != nil {
			t.Error("Test Failed - GetExchangeInfo() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("Test Failed - GetExchangeInfo() error cannot be nil")
		}
	}
}

func TestGetExchangeMap(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetExchangeMap(0, 0)
	if areAPICredtionalsSet(Startup) {
		if err != nil {
			t.Error("Test Failed - GetExchangeMap() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("Test Failed - GetExchangeMap() error cannot be nil")
		}
	}
}

func TestGetExchangeHistoricalListings(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetExchangeHistoricalListings()
	if err == nil {
		// TODO: update this once the feature above is implemented
		t.Error("Test Failed - GetExchangeHistoricalListings() error cannot be nil")
	}
}

func TestGetExchangeLatestListings(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetExchangeLatestListings()
	if err == nil {
		// TODO: update this once the feature above is implemented
		t.Error("Test Failed - GetExchangeHistoricalListings() error cannot be nil")
	}
}

func TestGetExchangeLatestMarketPairs(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetExchangeLatestMarketPairs(1, 0, 0)
	if areAPICredtionalsSet(Standard) {
		if err != nil {
			t.Error("Test Failed - GetExchangeLatestMarketPairs() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("Test Failed - GetExchangeLatestMarketPairs() error cannot be nil")
		}
	}
}

func TestGetExchangeLatestQuotes(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetExchangeLatestQuotes(1)
	if areAPICredtionalsSet(Standard) {
		if err != nil {
			t.Error("Test Failed - GetExchangeLatestQuotes() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("Test Failed - GetExchangeLatestQuotes() error cannot be nil")
		}
	}
}

func TestGetExchangeHistoricalQuotes(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetExchangeHistoricalQuotes(1, time.Now(), time.Now())
	if areAPICredtionalsSet(Standard) {
		if err != nil {
			t.Error("Test Failed - GetExchangeHistoricalQuotes() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("Test Failed - GetExchangeHistoricalQuotes() error cannot be nil")
		}
	}
}

func TestGetGlobalMeticLatestQuotes(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetGlobalMeticLatestQuotes()
	if areAPICredtionalsSet(Basic) {
		if err != nil {
			t.Error("Test Failed - GetGlobalMeticLatestQuotes() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("Test Failed - GetGlobalMeticLatestQuotes() error cannot be nil")
		}
	}
}

func TestGetGlobalMeticHistoricalQuotes(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetGlobalMeticHistoricalQuotes(time.Now(), time.Now())
	if areAPICredtionalsSet(Standard) {
		if err != nil {
			t.Error("Test Failed - GetGlobalMeticHistoricalQuotes() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("Test Failed - GetGlobalMeticHistoricalQuotes() error cannot be nil")
		}
	}
}

func TestGetPriceConversion(t *testing.T) {
	c.SetDefaults()
	TestSetup(t)
	_, err := c.GetPriceConversion(0, 1, time.Now())
	if areAPICredtionalsSet(Hobbyist) {
		if err != nil {
			t.Error("Test Failed - GetPriceConversion() error",
				err)
		}
	} else {
		if err == nil {
			t.Error("Test Failed - GetPriceConversion() error cannot be nil")
		}
	}
}

func TestSetAccountPlan(t *testing.T) {
	accPlans := []string{"basic", "startup", "hobbyist", "standard", "professional", "enterprise"}
	for _, plan := range accPlans {
		err := c.SetAccountPlan(plan)
		if err != nil {
			t.Error("Test Failed - SetAccountPlan() error", err)
		}

		switch plan {
		case "basic":
			if c.Plan != Basic {
				t.Error("Test Failed - SetAccountPlan() error basic plan not set correctly")
			}
		case "startup":
			if c.Plan != Startup {
				t.Error("Test Failed - SetAccountPlan() error startup plan not set correctly")
			}
		case "hobbyist":
			if c.Plan != Hobbyist {
				t.Error("Test Failed - SetAccountPlan() error hobbyist plan not set correctly")
			}
		case "standard":
			if c.Plan != Standard {
				t.Error("Test Failed - SetAccountPlan() error standard plan not set correctly")
			}
		case "professional":
			if c.Plan != Professional {
				t.Error("Test Failed - SetAccountPlan() error professional plan not set correctly")
			}
		case "enterprise":
			if c.Plan != Enterprise {
				t.Error("Test Failed - SetAccountPlan() error enterprise plan not set correctly")
			}
		}
	}

	if err := c.SetAccountPlan("bra"); err == nil {
		t.Error("Test Failed - SetAccountPlan() error cannot be nil")
	}
}
