// Package locale holds locale-coherent datasets. Picking a locale changes
// names, phone prefixes, and city↔region↔postcode triples together, so every
// field of a record agrees (uz_UZ → Uzbek names, +998, Tashkent districts).
package locale

import "sort"

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
	Name       string
	FirstNames []string
	LastNames  []string
	// Gendered name banks. When set, first name, last name and the gender field
	// stay consistent within a record (a male first name gets a male surname
	// form in gendered-surname locales like uz/ru). When empty, the mixed
	// FirstNames/LastNames are used with no gender coherence.
	MaleFirst, FemaleFirst []string
	MaleLast, FemaleLast   []string
	Places                 []Place
	CountryCode            string // e.g. "+998"
	EmailDomain            []string
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

// FirstNamesFor returns the first-name bank for a gender, falling back to the
// mixed list when the locale has no gendered names.
func (l *Locale) FirstNamesFor(gender string) []string {
	switch gender {
	case "female":
		if len(l.FemaleFirst) > 0 {
			return l.FemaleFirst
		}
	case "male":
		if len(l.MaleFirst) > 0 {
			return l.MaleFirst
		}
	}
	return l.FirstNames
}

// LastNamesFor returns the surname bank for a gender, falling back to the mixed
// list (locales without gendered surnames share one list).
func (l *Locale) LastNamesFor(gender string) []string {
	switch gender {
	case "female":
		if len(l.FemaleLast) > 0 {
			return l.FemaleLast
		}
	case "male":
		if len(l.MaleLast) > 0 {
			return l.MaleLast
		}
	}
	return l.LastNames
}

// Names returns all registered locale names.
func Names() []string {
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	// Sorted, because map order is random: without this the same build lists
	// the locales differently on every run, which makes the picker unusable and
	// any output that names them non-reproducible.
	sort.Strings(out)
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
	MaleFirst: []string{
		"James", "John", "Robert", "Michael", "William", "David", "Richard", "Joseph", "Thomas", "Charles",
		"Daniel", "Matthew", "Anthony", "Mark", "Donald", "Steven", "Andrew", "Paul", "Joshua", "Kevin",
		"Brian", "George", "Edward", "Ronald",
	},
	FemaleFirst: []string{
		"Mary", "Patricia", "Jennifer", "Linda", "Elizabeth", "Barbara", "Susan", "Jessica", "Sarah", "Karen",
		"Nancy", "Lisa", "Betty", "Sandra", "Ashley", "Emily", "Kimberly", "Donna", "Michelle", "Carol",
		"Amanda", "Melissa", "Deborah", "Stephanie",
	},
	// English surnames are not gendered — shared across both.
	LastNames: []string{
		"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Wilson", "Anderson",
		"Taylor", "Thomas", "Moore", "Jackson", "Martin", "Lee", "Thompson", "White", "Harris", "Clark",
		"Lewis", "Robinson", "Walker", "Young", "Allen", "King", "Wright", "Scott", "Green", "Baker",
		"Hall", "Nelson", "Carter", "Mitchell",
	},
	CountryCode: "+1",
	EmailDomain: []string{
		"example.com", "mail.com", "test.org",
		"gmail.com", "yahoo.com", "outlook.com", "hotmail.com", "icloud.com",
		"aol.com", "proton.me", "live.com", "msn.com", "gmx.com", "zoho.com",
	},
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
	MaleFirst: []string{
		"Azizbek", "Jasur", "Sardor", "Bekzod", "Otabek", "Sanjar", "Farrux", "Ulug'bek", "Doniyor", "Islom",
		"Jahongir", "Rustam", "Alisher", "Bobur", "Temur", "Shohruh", "Aziz", "Sherzod", "Javohir", "Diyor",
		"Akmal", "Botir", "Kamron", "Umid",
	},
	FemaleFirst: []string{
		"Dilnoza", "Malika", "Nilufar", "Gulnora", "Shahnoza", "Zuhra", "Kamola", "Feruza", "Sevara", "Madina",
		"Nodira", "Charos", "Dilfuza", "Muslima", "Gulbahor", "Zilola", "Nargiza", "Sabina", "Laylo", "Zarina",
		"Maftuna", "Dilorom", "Shahzoda", "Ozoda",
	},
	MaleLast: []string{
		"Karimov", "Yusupov", "Tursunov", "Qodirov", "Mirzayev", "Ergashev", "Sobirov", "Umarov", "Rasulov", "Yoqubov",
		"Aliyev", "Xakimov", "Sharipov", "Toshpo'latov", "Berdiyev", "Jo'rayev", "Nabiyev", "Sultonov", "Ismoilov", "Rahimov",
		"Abdullayev", "Saidov", "Nazarov", "Xolmatov",
	},
	FemaleLast: []string{
		"Karimova", "Yusupova", "Tursunova", "Qodirova", "Mirzayeva", "Ergasheva", "Sobirova", "Umarova", "Rasulova", "Yoqubova",
		"Aliyeva", "Xakimova", "Sharipova", "Toshpo'latova", "Berdiyeva", "Jo'rayeva", "Nabiyeva", "Sultonova", "Ismoilova", "Rahimova",
		"Abdullayeva", "Saidova", "Nazarova", "Xolmatova",
	},
	CountryCode: "+998",
	EmailDomain: []string{
		"mail.uz", "umail.uz", "example.uz", "inbox.uz", "bk.uz",
		"gmail.com", "mail.ru", "yandex.ru", "outlook.com", "icloud.com",
	},
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
	MaleFirst: []string{
		"Иван", "Дмитрий", "Сергей", "Алексей", "Андрей", "Михаил", "Владимир", "Николай", "Александр", "Павел",
		"Максим", "Артём", "Кирилл", "Роман", "Егор", "Никита", "Денис", "Владислав", "Илья", "Виктор",
		"Олег", "Антон", "Григорий", "Юрий",
	},
	FemaleFirst: []string{
		"Мария", "Анна", "Елена", "Ольга", "Наталья", "Татьяна", "Ирина", "Светлана", "Екатерина", "Юлия",
		"Виктория", "Марина", "Дарья", "Людмила", "Валентина", "Галина", "Полина", "Ксения", "Анастасия", "Вера",
		"Надежда", "Алина", "Оксана", "Инна",
	},
	MaleLast: []string{
		"Иванов", "Смирнов", "Попов", "Волков", "Новиков", "Козлов", "Павлов", "Орлов", "Андреев", "Захаров",
		"Степанов", "Сорокин", "Зайцев", "Соловьёв", "Борисов", "Киселёв", "Фролов", "Морозов", "Васильев", "Петров",
		"Михайлов", "Николаев", "Фёдоров", "Макаров",
	},
	FemaleLast: []string{
		"Иванова", "Смирнова", "Попова", "Волкова", "Новикова", "Козлова", "Павлова", "Орлова", "Андреева", "Захарова",
		"Степанова", "Сорокина", "Зайцева", "Соловьёва", "Борисова", "Киселёва", "Фролова", "Морозова", "Васильева", "Петрова",
		"Михайлова", "Николаева", "Фёдорова", "Макарова",
	},
	CountryCode: "+7",
	EmailDomain: []string{
		"mail.ru", "yandex.ru", "example.ru", "bk.ru", "list.ru", "inbox.ru",
		"rambler.ru", "gmail.com", "outlook.com", "icloud.com",
	},
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
