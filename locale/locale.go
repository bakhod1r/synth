// Package locale holds locale-coherent datasets. Picking a locale changes
// names, phone prefixes, and city↔region↔postcode triples together, so every
// field of a record agrees (uz_UZ → Uzbek names, +998, Tashkent districts).
package locale

// Place ties a region, city and postcode together so correlated generation
// (region → city → phone prefix) draws from a consistent triple.
type Place struct {
	Region   string
	City     string
	Postcode string
	// PhonePrefix is the operator/area digits after the country code.
	PhonePrefix string
}

// Locale is one language/region dataset.
type Locale struct {
	Name        string
	FirstNames  []string
	LastNames   []string
	Places      []Place
	CountryCode string // e.g. "+998"
	EmailDomain []string
	// CardBINs are valid issuer prefixes for this locale's card brands.
	CardBINs []string
	// IBANCountry is the 2-letter ISO country for IBAN generation.
	IBANCountry string
	IBANLength  int
	// Country is the human-readable country name (e.g. "Uzbekistan").
	Country string
	// Currency is the ISO 4217 code for monetary amounts (e.g. "UZS").
	Currency string
	// Companies are sample company/brand names for this locale.
	Companies []string
}

var registry = map[string]*Locale{
	"en_US": enUS,
	"uz_UZ": uzUZ,
}

// Get returns a locale by name, falling back to en_US.
func Get(name string) *Locale {
	if l, ok := registry[name]; ok {
		return l
	}
	return enUS
}

var enUS = &Locale{
	Name:        "en_US",
	FirstNames:  []string{"James", "Mary", "John", "Patricia", "Robert", "Jennifer", "Michael", "Linda", "David", "Elizabeth"},
	LastNames:   []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Wilson", "Anderson"},
	CountryCode: "+1",
	EmailDomain: []string{"example.com", "mail.com", "test.org"},
	CardBINs:    []string{"4539", "4556", "5425", "5105", "374245"},
	IBANCountry: "US",
	IBANLength:  24,
	Country:     "United States",
	Currency:    "USD",
	Companies:   []string{"Acme Corp", "Globex", "Initech", "Umbrella LLC", "Stark Industries"},
	Places: []Place{
		{"California", "Los Angeles", "90001", "213"},
		{"California", "San Diego", "92101", "619"},
		{"New York", "New York", "10001", "212"},
		{"Texas", "Houston", "77001", "713"},
		{"Illinois", "Chicago", "60601", "312"},
	},
}

var uzUZ = &Locale{
	Name:        "uz_UZ",
	FirstNames:  []string{"Azizbek", "Dilnoza", "Jasur", "Malika", "Sardor", "Nilufar", "Bekzod", "Gulnora", "Otabek", "Shahnoza"},
	LastNames:   []string{"Karimov", "Rashidova", "Yusupov", "Ismoilova", "Tursunov", "Abdullayeva", "Qodirov", "Saidova", "Mirzayev", "Rahimova"},
	CountryCode: "+998",
	EmailDomain: []string{"mail.uz", "umail.uz", "example.uz"},
	// HUMO (9860...) and UZCARD (8600...) prefixes.
	CardBINs:    []string{"8600", "9860"},
	IBANCountry: "UZ",
	IBANLength:  28,
	Country:     "Uzbekistan",
	Currency:    "UZS",
	Companies:   []string{"Uzum", "Payme", "Click", "Artel", "Uztelecom"},
	Places: []Place{
		{"Toshkent", "Toshkent", "100000", "90"},
		{"Samarqand", "Samarqand", "140100", "91"},
		{"Buxoro", "Buxoro", "200100", "93"},
		{"Andijon", "Andijon", "170100", "94"},
		{"Farg'ona", "Farg'ona", "150100", "95"},
	},
}
