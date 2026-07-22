package providers

import (
	"fmt"
	"strings"

	"github.com/bakhodir/synth/schema"
)

// Named things: publications, vehicles, vessels, aircraft, venues and hardware.
//
// Two different jobs live in this file, and they are handled differently on
// purpose:
//
//   - A ship, an aircraft or a magazine has a real name. Those are curated
//     lists of things that exist, because a made-up airliner model in a test
//     fixture is spotted instantly by anyone who works in the domain.
//   - A headline has no canonical list. Real headlines are somebody's writing,
//     and twenty of them repeated across ten thousand rows reads worse than
//     nothing. Those are composed from patterns, which is what gives a blog
//     table thousands of distinct plausible titles instead of twenty.

func init() {
	set(schema.KindMagazine, magazines)
	set(schema.KindShipName, shipNames)
	set(schema.KindAircraft, aircraft)
	set(schema.KindHotel, hotels)
	set(schema.KindComputer, computers)
	set(schema.KindTrainModel, trains)
	set(schema.KindShipType, shipTypes)

	registry[schema.KindArticleTitle] = articleTitle
	registry[schema.KindAirportName] = airportName
}

// airportName gives the airport's full name. Linked to an airport-code column
// with from=, it names that airport rather than a different one — the same
// rule that keeps a card brand tied to its number.
func airportName(c Ctx) any {
	if c.Field != nil && c.Field.From != "" && c.Sibling != nil {
		if code, ok := c.Sibling(c.Field.From).(string); ok {
			if name, found := airportsByCode[code]; found {
				return name
			}
		}
	}
	return airportNames[c.Rand.Pick(len(airportNames))]
}

// articleTitle composes a headline from patterns. Cardinality is the point: a
// blog table wants thousands of distinct plausible titles, and a fixed list of
// twenty repeated five hundred times is worse than useless for testing search,
// truncation or layout.
func articleTitle(c Ctx) any {
	pattern := titlePatterns[c.Rand.Pick(len(titlePatterns))]
	out := pattern
	for strings.Contains(out, "{") {
		open := strings.Index(out, "{")
		close := strings.Index(out[open:], "}")
		if close < 0 {
			break
		}
		token := out[open+1 : open+close]
		out = out[:open] + titleWord(c, token) + out[open+close+1:]
	}
	return out
}

func titleWord(c Ctx, token string) string {
	switch token {
	case "n":
		return fmt.Sprint([]int{3, 5, 7, 10, 12, 15, 20, 25}[c.Rand.Pick(8)])
	case "year":
		return fmt.Sprint(2019 + c.Rand.Pick(8))
	case "topic":
		return titleTopics[c.Rand.Pick(len(titleTopics))]
	case "verb":
		return titleVerbs[c.Rand.Pick(len(titleVerbs))]
	case "adj":
		return titleAdjectives[c.Rand.Pick(len(titleAdjectives))]
	case "noun":
		return titleNouns[c.Rand.Pick(len(titleNouns))]
	}
	return token
}

// titlePatterns are the shapes real headlines take. Together with the word
// lists below they yield well over a hundred thousand distinct titles.
var titlePatterns = []string{
	"How to {verb} your {noun}",
	"{n} ways to {verb} {topic}",
	"The complete guide to {topic}",
	"Why {topic} is {adj} in {year}",
	"{n} {adj} {noun} mistakes to avoid",
	"What nobody tells you about {topic}",
	"{topic}: a {adj} introduction",
	"Stop {verb}ing your {noun} the hard way",
	"We {verb}ed our {noun}. Here is what happened",
	"The {adj} case for {topic}",
	"{n} lessons from {verb}ing {topic}",
	"Rethinking {topic} for {year}",
	"A {adj} approach to {topic}",
	"From {noun} to {noun}: our {topic} story",
	"Is {topic} still {adj}?",
	"{topic} in practice: {n} lessons",
	"Building a {adj} {noun} from scratch",
	"The hidden cost of {adj} {noun}",
	"{verb}ing {topic} without losing your mind",
	"Everything we learned {verb}ing {topic}",
}

var titleTopics = []string{
	"remote work", "database design", "product strategy", "code review",
	"observability", "hiring", "technical debt", "onboarding", "pricing",
	"customer support", "data privacy", "accessibility", "performance",
	"microservices", "testing", "documentation", "incident response",
	"design systems", "machine learning", "open source", "team culture",
	"project estimates", "user research", "content strategy", "automation",
	"security reviews", "developer experience", "capacity planning",
	"analytics", "migrations", "refactoring", "caching",
}

var titleVerbs = []string{
	"scale", "rebuild", "simplify", "measure", "ship", "debug", "automate",
	"migrate", "document", "test", "monitor", "refactor", "design", "plan",
	"review", "optimize", "secure", "deploy", "maintain", "improve",
}

var titleAdjectives = []string{
	"practical", "honest", "surprising", "expensive", "quiet", "simple",
	"overlooked", "hard", "boring", "modern", "pragmatic", "unpopular",
	"necessary", "underrated", "brittle", "durable", "obvious", "costly",
}

var titleNouns = []string{
	"pipeline", "backlog", "roadmap", "database", "test suite", "API",
	"dashboard", "deployment", "schema", "monolith", "cache", "queue",
	"runbook", "changelog", "postmortem", "prototype", "codebase", "budget",
	"team", "release", "workflow", "handbook",
}

// magazines are real, long-running publications.
var magazines = []string{
	"National Geographic", "The Economist", "Time", "Nature", "Scientific American",
	"The New Yorker", "Wired", "Forbes", "Bloomberg Businessweek", "The Atlantic",
	"Harvard Business Review", "MIT Technology Review", "Popular Science", "New Scientist",
	"Rolling Stone", "Vogue", "GQ", "Elle", "Harper's Bazaar", "Esquire",
	"Vanity Fair", "The Spectator", "Le Monde diplomatique", "Der Spiegel",
	"Stern", "Paris Match", "L'Express", "Corriere della Sera", "El País Semanal",
	"Foreign Affairs", "The Paris Review", "Granta", "Aperture", "Monocle",
	"Architectural Digest", "Dwell", "Bon Appétit", "Condé Nast Traveler",
	"Sports Illustrated", "Autocar", "Top Gear", "Car and Driver",
}

// shipNames are real, historically notable vessels.
var shipNames = []string{
	"Cutty Sark", "Endeavour", "Beagle", "Endurance", "Discovery", "Fram",
	"Victory", "Constitution", "Mary Rose", "Vasa", "Golden Hind", "Santa María",
	"Bounty", "Great Eastern", "Titanic", "Britannic", "Lusitania", "Mauretania",
	"Queen Mary", "Queen Elizabeth 2", "Normandie", "United States", "Rotterdam",
	"Kon-Tiki", "Calypso", "Nautilus", "Amerigo Vespucci", "Gorch Fock",
	"Sedov", "Kruzenshtern", "Pallada", "Mir", "Dar Młodzieży", "Statsraad Lehmkuhl",
	"Christian Radich", "Sørlandet", "Danmark", "Eagle", "Esmeralda", "Juan Sebastián Elcano",
	"Alexander von Humboldt", "Peking", "Balclutha", "Star of India",
}

// shipTypes are the vessel classes a shipping or naval dataset uses.
var shipTypes = []string{
	"container ship", "bulk carrier", "oil tanker", "LNG carrier", "chemical tanker",
	"ro-ro carrier", "car carrier", "general cargo", "reefer", "cruise ship",
	"ferry", "tug", "barge", "trawler", "research vessel", "icebreaker",
	"dredger", "supply vessel", "cable layer", "heavy lift ship", "sailing yacht",
	"catamaran", "hydrofoil", "hovercraft",
}

// aircraft are real production models.
var aircraft = []string{
	"Airbus A320neo", "Airbus A321neo", "Airbus A319", "Airbus A330-900",
	"Airbus A350-900", "Airbus A350-1000", "Airbus A380-800", "Airbus A220-300",
	"Boeing 737-800", "Boeing 737 MAX 8", "Boeing 737 MAX 9", "Boeing 747-8",
	"Boeing 757-200", "Boeing 767-300ER", "Boeing 777-200LR", "Boeing 777-300ER",
	"Boeing 787-8", "Boeing 787-9", "Boeing 787-10", "Embraer E175",
	"Embraer E190", "Embraer E195-E2", "Bombardier CRJ900", "Bombardier CRJ1000",
	"ATR 72-600", "ATR 42-600", "De Havilland Dash 8-400", "Sukhoi Superjet 100",
	"Cessna 172 Skyhawk", "Cessna Citation XLS", "Beechcraft King Air 350",
	"Pilatus PC-12", "Gulfstream G650", "Gulfstream G550", "Bombardier Global 7500",
	"Dassault Falcon 8X", "Antonov An-124", "Ilyushin Il-76", "Airbus Beluga XL",
	"Boeing 767 Freighter",
}

// airports pairs real IATA codes with their full names, so a code column and
// a name column can describe the same airport.
var airports = []struct{ code, name string }{
	{"JFK", "John F. Kennedy International Airport"},
	{"LHR", "London Heathrow Airport"},
	{"CDG", "Paris Charles de Gaulle Airport"},
	{"HND", "Tokyo Haneda Airport"},
	{"NRT", "Narita International Airport"},
	{"DXB", "Dubai International Airport"},
	{"SIN", "Singapore Changi Airport"},
	{"FRA", "Frankfurt Airport"},
	{"IST", "Istanbul Airport"},
	{"TAS", "Tashkent International Airport"},
	{"ICN", "Incheon International Airport"},
	{"AMS", "Amsterdam Airport Schiphol"},
	{"MAD", "Adolfo Suárez Madrid–Barajas Airport"},
	{"BCN", "Josep Tarradellas Barcelona–El Prat Airport"},
	{"FCO", "Leonardo da Vinci–Fiumicino Airport"},
	{"MXP", "Milan Malpensa Airport"},
	{"MUC", "Munich Airport"},
	{"ZRH", "Zurich Airport"},
	{"VIE", "Vienna International Airport"},
	{"CPH", "Copenhagen Airport"},
	{"ARN", "Stockholm Arlanda Airport"},
	{"OSL", "Oslo Gardermoen Airport"},
	{"HEL", "Helsinki-Vantaa Airport"},
	{"WAW", "Warsaw Chopin Airport"},
	{"PRG", "Václav Havel Airport Prague"},
	{"BUD", "Budapest Ferenc Liszt International Airport"},
	{"OTP", "Henri Coandă International Airport"},
	{"SVO", "Sheremetyevo International Airport"},
	{"DME", "Domodedovo International Airport"},
	{"LED", "Pulkovo Airport"},
	{"KBP", "Boryspil International Airport"},
	{"ALA", "Almaty International Airport"},
	{"GYD", "Heydar Aliyev International Airport"},
	{"TBS", "Tbilisi International Airport"},
	{"DOH", "Hamad International Airport"},
	{"AUH", "Zayed International Airport"},
	{"JED", "King Abdulaziz International Airport"},
	{"RUH", "King Khalid International Airport"},
	{"CAI", "Cairo International Airport"},
	{"JNB", "O. R. Tambo International Airport"},
	{"CPT", "Cape Town International Airport"},
	{"LOS", "Murtala Muhammed International Airport"},
	{"NBO", "Jomo Kenyatta International Airport"},
	{"DEL", "Indira Gandhi International Airport"},
	{"BOM", "Chhatrapati Shivaji Maharaj International Airport"},
	{"BLR", "Kempegowda International Airport"},
	{"DAC", "Hazrat Shahjalal International Airport"},
	{"BKK", "Suvarnabhumi Airport"},
	{"KUL", "Kuala Lumpur International Airport"},
	{"CGK", "Soekarno–Hatta International Airport"},
	{"MNL", "Ninoy Aquino International Airport"},
	{"SGN", "Tan Son Nhat International Airport"},
	{"HAN", "Noi Bai International Airport"},
	{"PEK", "Beijing Capital International Airport"},
	{"PKX", "Beijing Daxing International Airport"},
	{"PVG", "Shanghai Pudong International Airport"},
	{"CAN", "Guangzhou Baiyun International Airport"},
	{"HKG", "Hong Kong International Airport"},
	{"TPE", "Taiwan Taoyuan International Airport"},
	{"SYD", "Sydney Kingsford Smith Airport"},
	{"MEL", "Melbourne Airport"},
	{"AKL", "Auckland Airport"},
	{"YYZ", "Toronto Pearson International Airport"},
	{"YVR", "Vancouver International Airport"},
	{"LAX", "Los Angeles International Airport"},
	{"SFO", "San Francisco International Airport"},
	{"ORD", "O'Hare International Airport"},
	{"ATL", "Hartsfield–Jackson Atlanta International Airport"},
	{"DFW", "Dallas/Fort Worth International Airport"},
	{"DEN", "Denver International Airport"},
	{"SEA", "Seattle–Tacoma International Airport"},
	{"MIA", "Miami International Airport"},
	{"BOS", "Boston Logan International Airport"},
	{"GRU", "São Paulo/Guarulhos International Airport"},
	{"GIG", "Rio de Janeiro/Galeão International Airport"},
	{"EZE", "Ministro Pistarini International Airport"},
	{"SCL", "Arturo Merino Benítez International Airport"},
	{"BOG", "El Dorado International Airport"},
	{"MEX", "Mexico City International Airport"},
	{"LIM", "Jorge Chávez International Airport"},
}

var airportsByCode = func() map[string]string {
	m := make(map[string]string, len(airports))
	for _, a := range airports {
		m[a.code] = a.name
	}
	return m
}()

var airportCodes = func() []string {
	out := make([]string, len(airports))
	for i, a := range airports {
		out[i] = a.code
	}
	return out
}()

var airportNames = func() []string {
	out := make([]string, len(airports))
	for i, a := range airports {
		out[i] = a.name
	}
	return out
}()

// hotels are real international chains and well-known individual properties.
var hotels = []string{
	"Hilton", "Hyatt Regency", "Marriott", "JW Marriott", "Sheraton", "Westin",
	"Radisson Blu", "Novotel", "Ibis", "Mercure", "Sofitel", "Pullman",
	"Holiday Inn", "Crowne Plaza", "InterContinental", "Ramada", "Wyndham",
	"Best Western", "Four Seasons", "Ritz-Carlton", "St. Regis", "Waldorf Astoria",
	"Mandarin Oriental", "Peninsula", "Shangri-La", "Raffles", "Banyan Tree",
	"Kempinski", "Meliá", "NH Collection", "Barceló", "Iberostar",
	"Scandic", "Clarion", "Comfort Inn", "Park Inn", "Courtyard", "Fairmont",
	"Le Méridien", "Hôtel Lutetia", "Hotel Sacher", "Grand Hotel Europe",
}

// computers are real desktop and laptop models.
var computers = []string{
	"MacBook Air M3", "MacBook Pro 14", "MacBook Pro 16", "iMac 24", "Mac mini",
	"Mac Studio", "Mac Pro", "ThinkPad X1 Carbon", "ThinkPad T14", "ThinkPad P1",
	"ThinkPad E15", "IdeaPad Slim 5", "Legion Pro 7", "Yoga Slim 7",
	"Dell XPS 13", "Dell XPS 15", "Dell Latitude 7440", "Dell Precision 5680",
	"Dell Inspiron 16", "Alienware m18", "HP Spectre x360", "HP EliteBook 840",
	"HP Pavilion 15", "HP ZBook Firefly", "HP Omen 16", "ASUS ZenBook 14",
	"ASUS ROG Zephyrus G14", "ASUS VivoBook 15", "ASUS ProArt Studiobook",
	"Acer Swift Go 14", "Acer Predator Helios 16", "Acer Aspire 5",
	"Microsoft Surface Laptop 6", "Microsoft Surface Pro 10", "Framework Laptop 13",
	"System76 Lemur Pro", "Razer Blade 15", "MSI Titan 18", "LG Gram 17",
	"Samsung Galaxy Book4 Pro", "Huawei MateBook X Pro",
}

// trains are real rolling-stock models.
var trains = []string{
	"Shinkansen N700S", "Shinkansen E5", "TGV Duplex", "TGV M", "ICE 3",
	"ICE 4", "Eurostar e320", "Frecciarossa 1000", "AVE S-103", "AVE S-112",
	"Sapsan", "Allegro", "Talgo 250", "Pendolino", "Railjet", "Nightjet",
	"CRH380A", "Fuxing CR400AF", "KTX-Sancheon", "HSR-350x", "Velaro Novo",
	"Stadler FLIRT", "Stadler KISS", "Siemens Desiro", "Bombardier Traxx",
	"Vectron", "Class 800 Azuma", "Class 390 Pendolino", "Amtrak Acela",
	"Metroliner", "Talent 3",
}
