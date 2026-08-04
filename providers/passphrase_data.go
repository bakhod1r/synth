package providers

import "github.com/bakhod1r/synth/schema"

// Passphrase word banks.
//
// This is the one locale dataset where size genuinely buys something. A
// weekday list cannot grow past seven and a colour list runs out at the edge of
// the language's real vocabulary, but a passphrase draws several words at once:
// 64 words gives 64⁴ ≈ 16.7 million four-word phrases, and 8 million more for
// every extra word. Volume here is not padding — it is the strength of the
// passphrase.
//
// The words are chosen to be short, concrete and unambiguous when spoken aloud,
// which is what a passphrase is for.

func init() { registerPassphraseBanks(passphraseBanks) }

// registerPassphraseBanks files each word bank under its locale, creating the
// locale's catalog entry when this is the first dataset it has. It is separate
// from init so both cases stay exercisable: init runs once, before a test can
// arrange a locale that has no catalog yet.
func registerPassphraseBanks(banks map[string][]string) {
	for code, words := range banks {
		if localeCatalog[code] == nil {
			localeCatalog[code] = map[schema.Kind][]string{}
		}
		localeCatalog[code][schema.KindPassphrase] = words
	}
}

var passphraseBanks = map[string][]string{
	"uz_UZ": {
		"olma", "anor", "bodom", "bahor", "baliq", "bargi", "bodring", "bulut",
		"chinor", "chiroq", "dala", "daraxt", "dengiz", "devor", "gilos", "gulzor",
		"ilon", "ismaloq", "kaptar", "karvon", "kitob", "koson", "kulcha", "kumush",
		"lola", "lochin", "meva", "misol", "nonushta", "olov", "oltin", "orzu",
		"osmon", "otash", "paxta", "pichoq", "qalam", "qamish", "qanot", "qishloq",
		"quyosh", "sabzi", "shamol", "sharshara", "shakar", "somon", "sopol", "suvda",
		"tarvuz", "tepalik", "tokcha", "tulki", "tuman", "turna", "uzum", "vodiy",
		"xazina", "yulduz", "yomg'ir", "yong'oq", "zangori", "zargar", "zilola", "zumrad",
	},
	"ru_RU": {
		"берёза", "болото", "ворона", "весна", "ветер", "вишня", "герань", "гнездо",
		"голубь", "гора", "гроза", "дерево", "дорога", "дятел", "ёжик", "жемчуг",
		"заря", "звезда", "земля", "зима", "камень", "капля", "кедр", "клевер",
		"колодец", "корабль", "костёр", "кувшин", "лебедь", "лестница", "листва", "лужайка",
		"луна", "медведь", "молния", "море", "мороз", "мостик", "облако", "озеро",
		"олень", "орешник", "остров", "парус", "песок", "печка", "поляна", "пшеница",
		"радуга", "река", "рябина", "сирень", "снегирь", "солнце", "сосна", "стрела",
		"туман", "тюльпан", "уголёк", "фонарь", "хвоя", "цапля", "черёмуха", "ястреб",
	},
	"tr_TR": {
		"ada", "akşam", "alev", "arı", "asma", "ayva", "badem", "bahar",
		"balık", "bulut", "ceviz", "cam", "çınar", "çilek", "damla", "deniz",
		"dere", "elma", "erik", "fener", "fidan", "fırtına", "gökyüzü", "gölge",
		"güneş", "incir", "ırmak", "kanat", "kavun", "kayık", "kekik", "kestane",
		"kiraz", "kule", "kumsal", "lale", "leylek", "limon", "mercan", "meşe",
		"nar", "orman", "papatya", "pınar", "rüzgar", "sedef", "serçe", "sonbahar",
		"şafak", "tarla", "tepe", "toprak", "turna", "üzüm", "vadi", "yaprak",
		"yıldız", "yağmur", "yelken", "yonca", "zeytin", "zümrüt", "kartal", "kaplan",
	},
	"de_DE": {
		"Ahorn", "Amsel", "Anker", "Apfel", "Bergen", "Birke", "Blume", "Brücke",
		"Distel", "Donner", "Eiche", "Eisen", "Falke", "Feder", "Felsen", "Feuer",
		"Fluss", "Garten", "Gipfel", "Hafen", "Hasel", "Himmel", "Hirsch", "Honig",
		"Insel", "Kastanie", "Kiesel", "Kirsche", "Klee", "Krone", "Küste", "Lampe",
		"Linde", "Möwe", "Mond", "Nebel", "Nelke", "Otter", "Pilz", "Quelle",
		"Raupe", "Regen", "Reiher", "Ruder", "Sattel", "Segel", "Silber", "Sommer",
		"Sonne", "Specht", "Stern", "Sturm", "Tanne", "Taube", "Ufer", "Veilchen",
		"Wald", "Weide", "Welle", "Wiese", "Wolke", "Wurzel", "Zeder", "Zweig",
	},
	"fr_FR": {
		"abeille", "amande", "ancre", "argile", "aurore", "banc", "bouleau", "branche",
		"brume", "cabane", "caillou", "canard", "cerise", "chêne", "cigogne", "colline",
		"corail", "coquille", "digue", "épine", "érable", "étoile", "falaise", "faucon",
		"fleuve", "forêt", "fougère", "givre", "grenier", "hibou", "houle", "jardin",
		"lagune", "lanterne", "lierre", "lilas", "loutre", "lumière", "marée", "menthe",
		"miel", "montagne", "mouette", "nuage", "orage", "orme", "pivoine", "plage",
		"pluie", "prairie", "quartz", "renard", "rivière", "rocher", "roseau", "sable",
		"saule", "sentier", "soleil", "source", "tilleul", "torrent", "vallée", "vigne",
	},
	"es_ES": {
		"abeja", "álamo", "ancla", "arena", "arroyo", "aurora", "avellana", "bahía",
		"barca", "bosque", "brisa", "cabaña", "caleta", "canela", "cantera", "cascada",
		"cedro", "cereza", "cigüeña", "colina", "coral", "cumbre", "duna", "encina",
		"escarcha", "espiga", "estrella", "faro", "helecho", "hiedra", "hoguera", "huerto",
		"jazmín", "laguna", "ladera", "lirio", "llanura", "luna", "manzana", "mirlo",
		"molino", "nube", "olivo", "orilla", "pantano", "pinar", "puente", "quijote",
		"rama", "ribera", "riachuelo", "roble", "rocío", "sendero", "sierra", "sombra",
		"tejado", "tomillo", "torrente", "trigo", "valle", "vereda", "viento", "zarza",
	},
	"it_IT": {
		"abete", "acero", "airone", "ancora", "arnia", "baita", "betulla", "borgo",
		"brezza", "cascata", "castagna", "cedro", "ciliegia", "cicogna", "collina", "conchiglia",
		"corallo", "duna", "edera", "faggio", "falco", "farfalla", "fienile", "fiume",
		"fontana", "foresta", "gabbiano", "ghianda", "giardino", "ginepro", "grotta", "isola",
		"lanterna", "lavanda", "laguna", "luna", "mandorla", "mirtillo", "montagna", "mulino",
		"nuvola", "olivo", "ombra", "onda", "orzo", "papavero", "pineta", "ponte",
		"prato", "quercia", "ramo", "riva", "roccia", "rugiada", "salice", "sentiero",
		"sorgente", "spiaggia", "stella", "tempesta", "tiglio", "valle", "vigneto", "zafferano",
	},
	"pt_BR": {
		"abelha", "açude", "âncora", "areia", "aurora", "barco", "brisa", "buriti",
		"cachoeira", "caju", "campina", "canela", "caverna", "cedro", "cerrado", "chuva",
		"cigarra", "colina", "coral", "duna", "enseada", "estrela", "farol", "figueira",
		"floresta", "fogueira", "fonte", "garça", "goiaba", "horta", "ilha", "ipê",
		"jabuti", "jasmim", "lagoa", "lua", "manga", "mangue", "moinho", "montanha",
		"nuvem", "orvalho", "palmeira", "pântano", "pedra", "pitanga", "planalto", "ponte",
		"praia", "raiz", "riacho", "sabiá", "samambaia", "serra", "sombra", "tamarindo",
		"telhado", "trilha", "trovão", "vale", "vento", "vereda", "vitória", "vulcão",
	},
	"pl_PL": {
		"bocian", "brzoza", "bursztyn", "chmura", "cisza", "czapla", "dąb", "deszcz",
		"dolina", "drzewo", "dzięcioł", "gałąź", "gniazdo", "góra", "grzyb", "iskra",
		"jabłoń", "jarzębina", "jaskinia", "jesion", "jezioro", "kamień", "kasztan", "klon",
		"komin", "koniczyna", "kotwica", "kropla", "las", "latarnia", "lawenda", "lipa",
		"łąka", "mgła", "miedza", "modrzew", "morze", "mostek", "obłok", "ogród",
		"orzech", "paproć", "piasek", "polana", "promień", "przystań", "rzeka", "sarna",
		"skała", "sosna", "srebro", "stodoła", "strumień", "szron", "śnieg", "topola",
		"tęcza", "wierzba", "wiatr", "wrzos", "wyspa", "zboże", "zdrój", "żagiel",
	},
	"ja_JP": {
		"あかね", "あさひ", "あゆみ", "いずみ", "いなほ", "うしお", "かえで", "かがみ",
		"かざん", "かすみ", "かもめ", "きぼう", "きりん", "くじら", "くもり", "こだま",
		"こはる", "さくら", "さざなみ", "しぐれ", "しずく", "すいれん", "すみれ", "せせらぎ",
		"たいよう", "たけのこ", "たんぽぽ", "つばき", "つばめ", "つゆくさ", "とうげ", "ときわ",
		"なぎさ", "なでしこ", "にじいろ", "ぬまち", "はやぶさ", "はるかぜ", "ひかり", "ひばり",
		"ふじさん", "ふぶき", "ほたる", "まつかぜ", "みずうみ", "みなと", "むぎばたけ", "もみじ",
		"やまざくら", "ゆきどけ", "ゆうやけ", "わかば", "あおぞら", "いしがき", "うめぼし", "おおぞら",
		"かいがん", "きたかぜ", "こうげん", "さんみゃく", "しらゆき", "たにがわ", "つきよ", "はなび",
	},
}
