package locale

// This file registers a broad set of locales (top ~50 languages/markets) so
// Synth can produce locale-coherent data far beyond the three fully-detailed
// datasets above. Each entry carries native first/last names, the country
// dialing code, currency, and a capital Place (region/city/postcode/prefix).
// Fields left blank fall back to en_US samples via mk, keeping every locale
// usable while allowing incremental enrichment.

type seed struct {
	key, country, code, currency, ibanCC string
	ibanLen                              int
	first, last                          []string
	region, city, postcode, prefix       string
	// Optional native datasets; when empty, mk falls back to en_US samples.
	companies, streets, jobs, products []string
}

// orDefault returns v if non-empty, else the fallback dataset.
func orDefault(v, fallback []string) []string {
	if len(v) > 0 {
		return v
	}
	return fallback
}

// mk builds a *Locale from a seed, filling gaps with sensible defaults so no
// provider ever sees an empty dataset.
func mk(s seed) *Locale {
	l := &Locale{
		Name:        s.key,
		FirstNames:  s.first,
		LastNames:   s.last,
		CountryCode: s.code,
		Country:     s.country,
		Currency:    s.currency,
		IBANCountry: s.ibanCC,
		IBANLength:  s.ibanLen,
		EmailDomain: enUS.EmailDomain,
		CardBINs:    enUS.CardBINs,
		Companies:   orDefault(s.companies, enUS.Companies),
		Streets:     orDefault(s.streets, enUS.Streets),
		Jobs:        orDefault(s.jobs, enUS.Jobs),
		Products:    orDefault(s.products, enUS.Products),
		Places:      []Place{{s.region, s.city, s.postcode, s.prefix}},
	}
	if l.IBANLength == 0 {
		l.IBANLength = 22
	}
	if l.IBANCountry == "" {
		l.IBANCountry = "XX"
	}
	return l
}

func init() {
	for _, s := range extLocales {
		if _, exists := registry[s.key]; !exists {
			registry[s.key] = mk(s)
		}
	}
}

var extLocales = []seed{
	{key: "en_GB", country: "United Kingdom", code: "+44", currency: "GBP", ibanCC: "GB", ibanLen: 22,
		first: []string{"Oliver", "Amelia", "George", "Isla", "Harry"}, last: []string{"Smith", "Jones", "Taylor", "Brown", "Wilson"},
		region: "England", city: "London", postcode: "EC1A", prefix: "20"},
	{key: "de_DE", country: "Germany", code: "+49", currency: "EUR", ibanCC: "DE", ibanLen: 22,
		first: []string{"Lukas", "Marie", "Leon", "Sophie", "Paul"}, last: []string{"Müller", "Schmidt", "Schneider", "Fischer", "Weber"},
		region: "Berlin", city: "Berlin", postcode: "10115", prefix: "30",
		companies: []string{"Siemens", "Volkswagen", "SAP", "Bosch", "Allianz", "BMW", "Adidas"},
		streets:   []string{"Hauptstraße", "Bahnhofstraße", "Schulstraße", "Gartenweg", "Lindenallee"},
		jobs:      []string{"Softwareentwickler", "Ingenieur", "Kaufmann", "Lehrer", "Arzt"},
		products:  []string{"Bierkrug", "Kuckucksuhr", "Lederhose", "Bratwurst", "Brezel"}},
	{key: "fr_FR", country: "France", code: "+33", currency: "EUR", ibanCC: "FR", ibanLen: 27,
		first: []string{"Louis", "Emma", "Gabriel", "Jade", "Hugo"}, last: []string{"Martin", "Bernard", "Dubois", "Thomas", "Robert"},
		region: "Île-de-France", city: "Paris", postcode: "75001", prefix: "1",
		companies: []string{"L'Oréal", "Renault", "Airbus", "Danone", "Orange", "Total", "Carrefour"},
		streets:   []string{"Rue de la Paix", "Avenue des Champs-Élysées", "Rue du Faubourg", "Boulevard Saint-Germain", "Rue de Rivoli"},
		jobs:      []string{"Développeur", "Ingénieur", "Comptable", "Professeur", "Médecin"},
		products:  []string{"Baguette", "Fromage", "Parfum", "Croissant", "Vin"}},
	{key: "es_ES", country: "Spain", code: "+34", currency: "EUR", ibanCC: "ES", ibanLen: 24,
		first: []string{"Hugo", "Lucía", "Martín", "Sofía", "Pablo"}, last: []string{"García", "Rodríguez", "González", "Fernández", "López"},
		region: "Madrid", city: "Madrid", postcode: "28001", prefix: "91",
		companies: []string{"Zara", "Santander", "Telefónica", "Iberdrola", "Repsol", "Mango", "Seat"},
		streets:   []string{"Calle Mayor", "Gran Vía", "Calle de Alcalá", "Paseo del Prado", "Calle Serrano"},
		jobs:      []string{"Desarrollador", "Ingeniero", "Contador", "Profesor", "Médico"},
		products:  []string{"Paella", "Jamón", "Guitarra", "Abanico", "Turrón"}},
	{key: "it_IT", country: "Italy", code: "+39", currency: "EUR", ibanCC: "IT", ibanLen: 27,
		first: []string{"Leonardo", "Sofia", "Francesco", "Giulia", "Alessandro"}, last: []string{"Rossi", "Russo", "Ferrari", "Esposito", "Bianchi"},
		region: "Lazio", city: "Roma", postcode: "00118", prefix: "6",
		companies: []string{"Ferrari", "Fiat", "Barilla", "Gucci", "Prada", "Lavazza", "Pirelli"},
		streets:   []string{"Via Roma", "Via Nazionale", "Corso Italia", "Via Garibaldi", "Piazza del Duomo"},
		jobs:      []string{"Sviluppatore", "Ingegnere", "Contabile", "Insegnante", "Medico"},
		products:  []string{"Pasta", "Pizza", "Espresso", "Gelato", "Parmigiano"}},
	{key: "pt_BR", country: "Brazil", code: "+55", currency: "BRL", ibanCC: "BR", ibanLen: 29,
		first: []string{"Miguel", "Helena", "Arthur", "Alice", "Bernardo"}, last: []string{"Silva", "Santos", "Oliveira", "Souza", "Lima"},
		region: "São Paulo", city: "São Paulo", postcode: "01000", prefix: "11",
		companies: []string{"Petrobras", "Vale", "Itaú", "Ambev", "Natura", "Magazine Luiza", "Embraer"},
		streets:   []string{"Avenida Paulista", "Rua Augusta", "Avenida Brasil", "Rua das Flores", "Avenida Atlântica"},
		jobs:      []string{"Desenvolvedor", "Engenheiro", "Contador", "Professor", "Médico"},
		products:  []string{"Café", "Açaí", "Feijoada", "Havaianas", "Guaraná"}},
	{key: "pt_PT", country: "Portugal", code: "+351", currency: "EUR", ibanCC: "PT", ibanLen: 25,
		first: []string{"João", "Maria", "Rodrigo", "Leonor", "Tomás"}, last: []string{"Silva", "Santos", "Ferreira", "Pereira", "Costa"},
		region: "Lisboa", city: "Lisboa", postcode: "1000", prefix: "21"},
	{key: "nl_NL", country: "Netherlands", code: "+31", currency: "EUR", ibanCC: "NL", ibanLen: 18,
		first: []string{"Daan", "Emma", "Sem", "Julia", "Lucas"}, last: []string{"De Jong", "Jansen", "De Vries", "Van den Berg", "Bakker"},
		region: "Noord-Holland", city: "Amsterdam", postcode: "1011", prefix: "20"},
	{key: "pl_PL", country: "Poland", code: "+48", currency: "PLN", ibanCC: "PL", ibanLen: 28,
		first: []string{"Jakub", "Zuzanna", "Jan", "Julia", "Antoni"}, last: []string{"Nowak", "Kowalski", "Wiśniewski", "Wójcik", "Kowalczyk"},
		region: "Mazowieckie", city: "Warszawa", postcode: "00-001", prefix: "22"},
	{key: "tr_TR", country: "Turkey", code: "+90", currency: "TRY", ibanCC: "TR", ibanLen: 26,
		first: []string{"Yusuf", "Zeynep", "Eymen", "Elif", "Ömer"}, last: []string{"Yılmaz", "Kaya", "Demir", "Şahin", "Çelik"},
		region: "İstanbul", city: "İstanbul", postcode: "34000", prefix: "212",
		companies: []string{"Turkish Airlines", "Arçelik", "Turkcell", "Vestel", "Ülker", "Beko", "Garanti"},
		streets:   []string{"İstiklal Caddesi", "Bağdat Caddesi", "Atatürk Bulvarı", "Cumhuriyet Caddesi", "Barbaros Bulvarı"},
		jobs:      []string{"Yazılımcı", "Mühendis", "Muhasebeci", "Öğretmen", "Doktor"},
		products:  []string{"Baklava", "Halı", "Lokum", "Çay", "Kebap"}},
	{key: "ar_SA", country: "Saudi Arabia", code: "+966", currency: "SAR", ibanCC: "SA", ibanLen: 24,
		first: []string{"محمد", "فاطمة", "أحمد", "عائشة", "علي"}, last: []string{"العتيبي", "الغامدي", "الشهري", "القحطاني", "الدوسري"},
		region: "الرياض", city: "الرياض", postcode: "11564", prefix: "11"},
	{key: "ar_EG", country: "Egypt", code: "+20", currency: "EGP", ibanCC: "EG", ibanLen: 29,
		first: []string{"محمد", "مريم", "أحمد", "نور", "يوسف"}, last: []string{"محمد", "علي", "حسن", "إبراهيم", "محمود"},
		region: "القاهرة", city: "القاهرة", postcode: "11511", prefix: "2"},
	{key: "fa_IR", country: "Iran", code: "+98", currency: "IRR", ibanCC: "IR", ibanLen: 26,
		first: []string{"علی", "زهرا", "امیر", "فاطمه", "محمد"}, last: []string{"محمدی", "حسینی", "رضایی", "موسوی", "کریمی"},
		region: "تهران", city: "تهران", postcode: "11369", prefix: "21"},
	{key: "he_IL", country: "Israel", code: "+972", currency: "ILS", ibanCC: "IL", ibanLen: 23,
		first: []string{"נועם", "תמר", "איתי", "מאיה", "יונתן"}, last: []string{"כהן", "לוי", "מזרחי", "פרץ", "ביטון"},
		region: "תל אביב", city: "תל אביב", postcode: "61000", prefix: "3"},
	{key: "hi_IN", country: "India", code: "+91", currency: "INR", ibanCC: "IN", ibanLen: 0,
		first: []string{"आरव", "सान्या", "विहान", "अनन्या", "अद्वैत"}, last: []string{"शर्मा", "वर्मा", "गुप्ता", "सिंह", "कुमार"},
		region: "दिल्ली", city: "नई दिल्ली", postcode: "110001", prefix: "11"},
	{key: "bn_BD", country: "Bangladesh", code: "+880", currency: "BDT", ibanCC: "BD", ibanLen: 0,
		first: []string{"আবির", "নুসরাত", "রাফি", "তানিয়া", "সাকিব"}, last: []string{"ইসলাম", "রহমান", "আহমেদ", "হোসেন", "খান"},
		region: "ঢাকা", city: "ঢাকা", postcode: "1000", prefix: "2"},
	{key: "ja_JP", country: "Japan", code: "+81", currency: "JPY", ibanCC: "JP", ibanLen: 0,
		first: []string{"陽翔", "陽菜", "蓮", "凛", "湊"}, last: []string{"佐藤", "鈴木", "高橋", "田中", "渡辺"},
		region: "東京都", city: "東京", postcode: "100-0001", prefix: "3",
		companies: []string{"トヨタ", "ソニー", "任天堂", "パナソニック", "ソフトバンク", "ホンダ", "楽天"},
		streets:   []string{"銀座通り", "青山通り", "表参道", "中央通り", "桜通り"},
		jobs:      []string{"エンジニア", "デザイナー", "会計士", "教師", "医師"},
		products:  []string{"寿司", "ラーメン", "着物", "扇子", "抹茶"}},
	{key: "ko_KR", country: "South Korea", code: "+82", currency: "KRW", ibanCC: "KR", ibanLen: 0,
		first: []string{"서준", "서연", "도윤", "지우", "하준"}, last: []string{"김", "이", "박", "최", "정"},
		region: "서울", city: "서울", postcode: "04524", prefix: "2",
		companies: []string{"삼성", "현대", "LG", "SK", "카카오", "네이버", "기아"},
		streets:   []string{"강남대로", "테헤란로", "종로", "세종대로", "을지로"},
		jobs:      []string{"개발자", "엔지니어", "회계사", "교사", "의사"},
		products:  []string{"김치", "비빔밥", "한복", "인삼", "떡"}},
	{key: "zh_CN", country: "China", code: "+86", currency: "CNY", ibanCC: "CN", ibanLen: 0,
		first: []string{"伟", "芳", "娜", "敏", "静"}, last: []string{"王", "李", "张", "刘", "陈"},
		region: "北京市", city: "北京", postcode: "100000", prefix: "10",
		companies: []string{"阿里巴巴", "腾讯", "华为", "百度", "小米", "京东", "字节跳动"},
		streets:   []string{"长安街", "南京路", "中山路", "人民路", "解放路"},
		jobs:      []string{"软件工程师", "设计师", "会计", "教师", "医生"},
		products:  []string{"茶", "瓷器", "丝绸", "饺子", "月饼"}},
	{key: "zh_TW", country: "Taiwan", code: "+886", currency: "TWD", ibanCC: "TW", ibanLen: 0,
		first: []string{"家豪", "淑芬", "俊傑", "美玲", "志明"}, last: []string{"陳", "林", "黃", "張", "李"},
		region: "臺北市", city: "臺北", postcode: "100", prefix: "2"},
	{key: "th_TH", country: "Thailand", code: "+66", currency: "THB", ibanCC: "TH", ibanLen: 0,
		first: []string{"สมชาย", "สมหญิง", "ธนา", "มาลี", "อนุชา"}, last: []string{"แซ่ตั้ง", "ศรีสุข", "บุญมี", "รักดี", "ทองดี"},
		region: "กรุงเทพมหานคร", city: "กรุงเทพ", postcode: "10100", prefix: "2"},
	{key: "vi_VN", country: "Vietnam", code: "+84", currency: "VND", ibanCC: "VN", ibanLen: 0,
		first: []string{"An", "Linh", "Minh", "Hương", "Tuấn"}, last: []string{"Nguyễn", "Trần", "Lê", "Phạm", "Hoàng"},
		region: "Hà Nội", city: "Hà Nội", postcode: "10000", prefix: "24"},
	{key: "id_ID", country: "Indonesia", code: "+62", currency: "IDR", ibanCC: "ID", ibanLen: 0,
		first: []string{"Budi", "Siti", "Agus", "Dewi", "Eko"}, last: []string{"Wijaya", "Santoso", "Hidayat", "Kusuma", "Pratama"},
		region: "Jakarta", city: "Jakarta", postcode: "10110", prefix: "21"},
	{key: "ms_MY", country: "Malaysia", code: "+60", currency: "MYR", ibanCC: "MY", ibanLen: 0,
		first: []string{"Ahmad", "Nurul", "Muhammad", "Siti", "Ali"}, last: []string{"Bin Abdullah", "Bin Ismail", "Binti Hassan", "Bin Osman", "Binti Yusof"},
		region: "Kuala Lumpur", city: "Kuala Lumpur", postcode: "50000", prefix: "3"},
	{key: "fil_PH", country: "Philippines", code: "+63", currency: "PHP", ibanCC: "PH", ibanLen: 0,
		first: []string{"Jose", "Maria", "Juan", "Rosa", "Antonio"}, last: []string{"Santos", "Reyes", "Cruz", "Bautista", "Garcia"},
		region: "Metro Manila", city: "Manila", postcode: "1000", prefix: "2"},
	{key: "sv_SE", country: "Sweden", code: "+46", currency: "SEK", ibanCC: "SE", ibanLen: 24,
		first: []string{"William", "Alice", "Oscar", "Maja", "Lucas"}, last: []string{"Andersson", "Johansson", "Karlsson", "Nilsson", "Eriksson"},
		region: "Stockholm", city: "Stockholm", postcode: "111 20", prefix: "8"},
	{key: "nb_NO", country: "Norway", code: "+47", currency: "NOK", ibanCC: "NO", ibanLen: 15,
		first: []string{"Jakob", "Emma", "Emil", "Nora", "Noah"}, last: []string{"Hansen", "Johansen", "Olsen", "Larsen", "Andersen"},
		region: "Oslo", city: "Oslo", postcode: "0001", prefix: "22"},
	{key: "da_DK", country: "Denmark", code: "+45", currency: "DKK", ibanCC: "DK", ibanLen: 18,
		first: []string{"William", "Emma", "Oscar", "Ida", "Noah"}, last: []string{"Jensen", "Nielsen", "Hansen", "Pedersen", "Andersen"},
		region: "Hovedstaden", city: "København", postcode: "1050", prefix: "3"},
	{key: "fi_FI", country: "Finland", code: "+358", currency: "EUR", ibanCC: "FI", ibanLen: 18,
		first: []string{"Elias", "Aada", "Onni", "Sofia", "Leo"}, last: []string{"Korhonen", "Virtanen", "Mäkinen", "Nieminen", "Mäkelä"},
		region: "Uusimaa", city: "Helsinki", postcode: "00100", prefix: "9"},
	{key: "cs_CZ", country: "Czechia", code: "+420", currency: "CZK", ibanCC: "CZ", ibanLen: 24,
		first: []string{"Jakub", "Eliška", "Jan", "Tereza", "Adam"}, last: []string{"Novák", "Svoboda", "Novotný", "Dvořák", "Černý"},
		region: "Praha", city: "Praha", postcode: "110 00", prefix: "2"},
	{key: "el_GR", country: "Greece", code: "+30", currency: "EUR", ibanCC: "GR", ibanLen: 27,
		first: []string{"Γιώργος", "Μαρία", "Δημήτρης", "Ελένη", "Νίκος"}, last: []string{"Παπαδόπουλος", "Παππάς", "Νικολάου", "Γεωργίου", "Δημητρίου"},
		region: "Αττική", city: "Αθήνα", postcode: "10431", prefix: "21"},
	{key: "hu_HU", country: "Hungary", code: "+36", currency: "HUF", ibanCC: "HU", ibanLen: 28,
		first: []string{"Bence", "Hanna", "Máté", "Anna", "Levente"}, last: []string{"Nagy", "Kovács", "Tóth", "Szabó", "Horváth"},
		region: "Budapest", city: "Budapest", postcode: "1051", prefix: "1"},
	{key: "ro_RO", country: "Romania", code: "+40", currency: "RON", ibanCC: "RO", ibanLen: 24,
		first: []string{"Andrei", "Maria", "Alexandru", "Elena", "Gabriel"}, last: []string{"Popescu", "Ionescu", "Popa", "Radu", "Dumitru"},
		region: "București", city: "București", postcode: "010011", prefix: "21"},
	{key: "uk_UA", country: "Ukraine", code: "+380", currency: "UAH", ibanCC: "UA", ibanLen: 29,
		first: []string{"Олександр", "Софія", "Максим", "Анна", "Дмитро"}, last: []string{"Мельник", "Шевченко", "Коваленко", "Бондаренко", "Ткаченко"},
		region: "Київ", city: "Київ", postcode: "01001", prefix: "44"},
	{key: "bg_BG", country: "Bulgaria", code: "+359", currency: "BGN", ibanCC: "BG", ibanLen: 22,
		first: []string{"Георги", "Мария", "Иван", "Виктория", "Александър"}, last: []string{"Иванов", "Георгиев", "Димитров", "Петров", "Николов"},
		region: "София", city: "София", postcode: "1000", prefix: "2"},
	{key: "sr_RS", country: "Serbia", code: "+381", currency: "RSD", ibanCC: "RS", ibanLen: 22,
		first: []string{"Nikola", "Jovana", "Marko", "Milica", "Stefan"}, last: []string{"Jovanović", "Petrović", "Nikolić", "Marković", "Đorđević"},
		region: "Beograd", city: "Beograd", postcode: "11000", prefix: "11"},
	{key: "hr_HR", country: "Croatia", code: "+385", currency: "EUR", ibanCC: "HR", ibanLen: 21,
		first: []string{"Luka", "Mia", "Ivan", "Ana", "David"}, last: []string{"Horvat", "Kovačević", "Babić", "Marić", "Jurić"},
		region: "Zagreb", city: "Zagreb", postcode: "10000", prefix: "1"},
	{key: "sk_SK", country: "Slovakia", code: "+421", currency: "EUR", ibanCC: "SK", ibanLen: 24,
		first: []string{"Jakub", "Sofia", "Adam", "Nina", "Samuel"}, last: []string{"Horváth", "Kováč", "Varga", "Tóth", "Nagy"},
		region: "Bratislava", city: "Bratislava", postcode: "811 01", prefix: "2"},
	{key: "sl_SI", country: "Slovenia", code: "+386", currency: "EUR", ibanCC: "SI", ibanLen: 19,
		first: []string{"Luka", "Eva", "Nik", "Ana", "Jan"}, last: []string{"Novak", "Horvat", "Kovačič", "Krajnc", "Zupan"},
		region: "Ljubljana", city: "Ljubljana", postcode: "1000", prefix: "1"},
	{key: "lt_LT", country: "Lithuania", code: "+370", currency: "EUR", ibanCC: "LT", ibanLen: 20,
		first: []string{"Matas", "Emilija", "Nojus", "Gabija", "Jokūbas"}, last: []string{"Kazlauskas", "Petrauskas", "Jankauskas", "Stankevičius", "Vasiliauskas"},
		region: "Vilnius", city: "Vilnius", postcode: "01100", prefix: "5"},
	{key: "lv_LV", country: "Latvia", code: "+371", currency: "EUR", ibanCC: "LV", ibanLen: 21,
		first: []string{"Roberts", "Sofija", "Markuss", "Alise", "Emīls"}, last: []string{"Bērziņš", "Kalniņš", "Ozoliņš", "Jansons", "Ozols"},
		region: "Rīga", city: "Rīga", postcode: "1050", prefix: "6"},
	{key: "et_EE", country: "Estonia", code: "+372", currency: "EUR", ibanCC: "EE", ibanLen: 20,
		first: []string{"Rasmus", "Sofia", "Robin", "Mia", "Martin"}, last: []string{"Tamm", "Saar", "Sepp", "Mägi", "Kask"},
		region: "Harjumaa", city: "Tallinn", postcode: "10111", prefix: "6"},
	{key: "kk_KZ", country: "Kazakhstan", code: "+7", currency: "KZT", ibanCC: "KZ", ibanLen: 20,
		first: []string{"Алихан", "Айым", "Нурлан", "Аружан", "Ерлан"}, last: []string{"Ахметов", "Оспанов", "Нурланов", "Сериков", "Беков"},
		region: "Астана", city: "Астана", postcode: "010000", prefix: "7172"},
	{key: "ka_GE", country: "Georgia", code: "+995", currency: "GEL", ibanCC: "GE", ibanLen: 22,
		first: []string{"გიორგი", "ნინო", "დავითი", "მარიამი", "ლუკა"}, last: []string{"ბერიძე", "მამედოვი", "კაპანაძე", "გელაშვილი", "მაისურაძე"},
		region: "თბილისი", city: "თბილისი", postcode: "0100", prefix: "32"},
	{key: "az_AZ", country: "Azerbaijan", code: "+994", currency: "AZN", ibanCC: "AZ", ibanLen: 28,
		first: []string{"Əli", "Nərgiz", "Murad", "Aysu", "Elvin"}, last: []string{"Əliyev", "Məmmədov", "Hüseynov", "Quliyev", "İsmayılov"},
		region: "Bakı", city: "Bakı", postcode: "AZ1000", prefix: "12"},
	{key: "en_CA", country: "Canada", code: "+1", currency: "CAD", ibanCC: "CA", ibanLen: 0,
		first: []string{"Liam", "Olivia", "Noah", "Emma", "Jackson"}, last: []string{"Smith", "Brown", "Tremblay", "Martin", "Roy"},
		region: "Ontario", city: "Toronto", postcode: "M5H", prefix: "416"},
	{key: "en_AU", country: "Australia", code: "+61", currency: "AUD", ibanCC: "AU", ibanLen: 0,
		first: []string{"Oliver", "Charlotte", "Jack", "Olivia", "Noah"}, last: []string{"Smith", "Jones", "Williams", "Brown", "Wilson"},
		region: "New South Wales", city: "Sydney", postcode: "2000", prefix: "2"},
	{key: "es_MX", country: "Mexico", code: "+52", currency: "MXN", ibanCC: "MX", ibanLen: 0,
		first: []string{"Santiago", "Sofía", "Mateo", "Valentina", "Diego"}, last: []string{"Hernández", "García", "Martínez", "López", "González"},
		region: "Ciudad de México", city: "Ciudad de México", postcode: "01000", prefix: "55"},
	{key: "es_AR", country: "Argentina", code: "+54", currency: "ARS", ibanCC: "AR", ibanLen: 0,
		first: []string{"Benjamín", "Martina", "Mateo", "Emma", "Santiago"}, last: []string{"González", "Rodríguez", "Gómez", "Fernández", "López"},
		region: "Buenos Aires", city: "Buenos Aires", postcode: "1000", prefix: "11"},
}
