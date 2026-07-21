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
	// Streets are sample street names (locale-flavored).
	Streets []string
	// Jobs are sample job titles.
	Jobs []string
	// Products are sample product names.
	Products []string
	// IPBlocks are representative first octets of IPv4 ranges allocated to the
	// country, so a generated IP plausibly geolocates to the locale.
	IPBlocks []int
}

// countryIPBlocks maps a locale to representative first-octet IPv4 blocks
// allocated to that country (approximate, from RIR allocations). Used so a
// generated IP plausibly geolocates to the record's locale.
var countryIPBlocks = map[string][]int{
	"en_US": {23, 24, 50, 63, 66, 72, 104, 174},
	"uz_UZ": {84, 213, 195, 217},
	"ru_RU": {5, 31, 46, 78, 95, 178, 188},
	"de_DE": {46, 78, 88, 91, 178, 217},
	"fr_FR": {80, 82, 90, 92, 176, 195},
	"ja_JP": {27, 43, 106, 126, 133, 210, 219},
	"zh_CN": {36, 39, 58, 101, 116, 122, 202, 218},
	"ko_KR": {14, 27, 39, 175, 211, 220},
	"tr_TR": {78, 88, 176, 178, 212},
	"hi_IN": {14, 27, 49, 103, 117, 122, 182},
	"pt_BR": {45, 177, 179, 189, 191, 200, 201},
}

// Colors and genders are locale-independent enough to share.
var (
	Colors  = []string{"Red", "Green", "Blue", "Yellow", "Black", "White", "Orange", "Purple", "Cyan", "Magenta"}
	Genders = []string{"male", "female", "other"}
)

var registry = map[string]*Locale{
	"en_US": enUS,
	"uz_UZ": uzUZ,
	"ru_RU": ruRU,
}

// Names returns all registered locale names.
func Names() []string {
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	return out
}

// Get returns a locale by name, falling back to en_US.
func Get(name string) *Locale {
	l, ok := registry[name]
	if !ok {
		return enUS
	}
	if l.IPBlocks == nil {
		if b, ok := countryIPBlocks[name]; ok {
			l.IPBlocks = b
		} else {
			l.IPBlocks = countryIPBlocks["en_US"]
		}
	}
	return l
}

var enUS = &Locale{
	Name: "en_US",
	FirstNames: []string{
		"James", "Mary", "John", "Patricia", "Robert", "Jennifer", "Michael", "Linda", "David", "Elizabeth",
		"William", "Barbara", "Richard", "Susan", "Joseph", "Jessica", "Thomas", "Sarah", "Charles", "Karen",
		"Daniel", "Nancy", "Matthew", "Lisa", "Anthony", "Betty", "Mark", "Sandra", "Donald", "Ashley",
		"Steven", "Emily", "Andrew", "Kimberly",
	},
	LastNames: []string{
		"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Wilson", "Anderson",
		"Taylor", "Thomas", "Moore", "Jackson", "Martin", "Lee", "Thompson", "White", "Harris", "Clark",
		"Lewis", "Robinson", "Walker", "Young", "Allen", "King", "Wright", "Scott", "Green", "Baker",
		"Hall", "Nelson", "Carter", "Mitchell",
	},
	CountryCode: "+1",
	EmailDomain: []string{"example.com", "mail.com", "test.org"},
	CardBINs:    []string{"4539", "4556", "5425", "5105", "374245"},
	IBANCountry: "US",
	IBANLength:  24,
	Country:     "United States",
	Currency:    "USD",
	Companies:   []string{"Acme Corp", "Globex", "Initech", "Umbrella LLC", "Stark Industries"},
	Streets:     []string{"Main St", "Oak Ave", "Maple Dr", "Cedar Ln", "Pine Rd"},
	Jobs:        []string{"Software Engineer", "Product Manager", "Data Analyst", "Designer", "Accountant"},
	Products:    []string{"Widget", "Gadget", "Gizmo", "Doohickey", "Contraption"},
	Places: []Place{
		{"California", "Los Angeles", "90001", "213"},
		{"California", "San Diego", "92101", "619"},
		{"New York", "New York", "10001", "212"},
		{"Texas", "Houston", "77001", "713"},
		{"Illinois", "Chicago", "60601", "312"},
	},
}

var uzUZ = &Locale{
	Name: "uz_UZ",
	FirstNames: []string{
		"Azizbek", "Dilnoza", "Jasur", "Malika", "Sardor", "Nilufar", "Bekzod", "Gulnora", "Otabek", "Shahnoza",
		"Sanjar", "Zuhra", "Farrux", "Kamola", "Ulug'bek", "Feruza", "Doniyor", "Sevara", "Islom", "Madina",
		"Jahongir", "Nodira", "Rustam", "Charos", "Alisher", "Dilfuza", "Bobur", "Muslima", "Temur", "Gulbahor",
		"Shohruh", "Zilola", "Aziz", "Nargiza",
	},
	LastNames: []string{
		"Karimov", "Rashidova", "Yusupov", "Ismoilova", "Tursunov", "Abdullayeva", "Qodirov", "Saidova", "Mirzayev", "Rahimova",
		"Ergashev", "Yo'ldosheva", "Sobirov", "Nazarova", "Umarov", "Xolmatova", "Rasulov", "G'aniyeva", "Yoqubov", "Sultonova",
		"Aliyev", "Boboyeva", "Xakimov", "Nurmatova", "Sharipov", "Islomova", "Toshpo'latov", "Qosimova", "Berdiyev", "Ochilova",
		"Jo'rayev", "Hamidova", "Nabiyev", "Salimova",
	},
	CountryCode: "+998",
	EmailDomain: []string{"mail.uz", "umail.uz", "example.uz"},
	// HUMO (9860...) and UZCARD (8600...) prefixes.
	CardBINs:    []string{"8600", "9860"},
	IBANCountry: "UZ",
	IBANLength:  28,
	Country:     "Uzbekistan",
	Currency:    "UZS",
	Companies:   []string{"Uzum", "Payme", "Click", "Artel", "Uztelecom"},
	Streets:     []string{"Amir Temur ko'chasi", "Navoiy ko'chasi", "Bobur ko'chasi", "Chilonzor ko'chasi", "Yunusobod ko'chasi"},
	Jobs:        []string{"Dasturchi", "Menejer", "Muhandis", "O'qituvchi", "Hisobchi"},
	Products:    []string{"Choynak", "Gilam", "Do'ppi", "Chopon", "Piyola"},
	Places: []Place{
		{"Toshkent", "Toshkent", "100000", "90"},
		{"Samarqand", "Samarqand", "140100", "91"},
		{"Buxoro", "Buxoro", "200100", "93"},
		{"Andijon", "Andijon", "170100", "94"},
		{"Farg'ona", "Farg'ona", "150100", "95"},
	},
}

var ruRU = &Locale{
	Name: "ru_RU",
	FirstNames: []string{
		"Иван", "Мария", "Дмитрий", "Анна", "Сергей", "Елена", "Алексей", "Ольга", "Андрей", "Наталья",
		"Михаил", "Татьяна", "Владимир", "Ирина", "Николай", "Светлана", "Александр", "Екатерина", "Павел", "Юлия",
		"Максим", "Виктория", "Артём", "Марина", "Кирилл", "Дарья", "Роман", "Людмила", "Егор", "Валентина",
		"Никита", "Галина", "Денис", "Полина",
	},
	LastNames: []string{
		"Иванов", "Петрова", "Смирнов", "Кузнецова", "Попов", "Соколова", "Волков", "Морозова", "Новиков", "Лебедева",
		"Козлов", "Егорова", "Павлов", "Николаева", "Орлов", "Макарова", "Андреев", "Фёдорова", "Захаров", "Васильева",
		"Степанов", "Романова", "Сорокин", "Ковалёва", "Зайцев", "Белова", "Соловьёв", "Комарова", "Борисов", "Медведева",
		"Киселёв", "Тарасова", "Фролов", "Гусева",
	},
	CountryCode: "+7",
	EmailDomain: []string{"mail.ru", "yandex.ru", "example.ru"},
	CardBINs:    []string{"2200", "2202", "4276"},
	IBANCountry: "RU",
	IBANLength:  33,
	Country:     "Russia",
	Currency:    "RUB",
	Companies:   []string{"Яндекс", "Газпром", "Сбербанк", "Озон", "Ростех"},
	Streets:     []string{"Ленина", "Пушкина", "Гагарина", "Мира", "Советская"},
	Jobs:        []string{"Разработчик", "Менеджер", "Инженер", "Дизайнер", "Бухгалтер"},
	Products:    []string{"Виджет", "Гаджет", "Устройство", "Прибор", "Механизм"},
	Places: []Place{
		{"Московская область", "Москва", "101000", "495"},
		{"Ленинградская область", "Санкт-Петербург", "190000", "812"},
		{"Свердловская область", "Екатеринбург", "620000", "343"},
		{"Новосибирская область", "Новосибирск", "630000", "383"},
		{"Татарстан", "Казань", "420000", "843"},
	},
}
