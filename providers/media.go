package providers

import (
	"fmt"
	"strings"

	"github.com/bakhod1r/synth/schema"
)

// Entertainment, gaming, payment tokens, roads and nature.

func init() {
	set(schema.KindMovieCharacter, movieCharacters)
	set(schema.KindGame, games)
	setLocalized(schema.KindBird, birds)
	setLocalized(schema.KindStreetType, streetTypes)
	setLocalized(schema.KindDisease, diseases)

	registry[schema.KindGamertag] = gamertag
	registry[schema.KindCardToken] = cardToken
}

// gamertag composes a handle the way players actually build them: a word, a
// second word or a number, sometimes leetspeak. A fixed list would repeat
// within a few hundred rows; composition gives millions.
func gamertag(c Ctx) any {
	prefix := gamerPrefixes[c.Rand.Pick(len(gamerPrefixes))]
	noun := gamerNouns[c.Rand.Pick(len(gamerNouns))]

	var b strings.Builder
	if c.Rand.Bool(0.35) {
		b.WriteString(gamerTags[c.Rand.Pick(len(gamerTags))])
	}
	b.WriteString(prefix)
	b.WriteString(noun)
	switch c.Rand.Pick(4) {
	case 0:
		fmt.Fprintf(&b, "%d", c.Rand.IntRange(1, 99))
	case 1:
		fmt.Fprintf(&b, "_%d", c.Rand.IntRange(1, 2010))
	case 2:
		b.WriteString("XD")
	}
	return b.String()
}

// cardToken is the opaque reference a payment gateway returns in place of a
// card number.
//
// It is deliberately *not* derived from any card column. That independence is
// the entire security property of tokenization: a token that leaks tells an
// attacker nothing about the card behind it. A generator that derived one from
// the other would quietly teach the opposite lesson, and code tested against it
// would be built on a false assumption.
func cardToken(c Ctx) any {
	return "tok_" + upperAlnum(c.Rand, 24)
}

// movieCharacters are characters from widely-known films.
var movieCharacters = []string{
	"Indiana Jones", "Ellen Ripley", "Sarah Connor", "Rick Deckard", "Marty McFly",
	"Doc Brown", "Vito Corleone", "Michael Corleone", "Travis Bickle", "Norman Bates",
	"Atticus Finch", "Forrest Gump", "Rocky Balboa", "John McClane", "Ethan Hunt",
	"James Bond", "Jason Bourne", "Maximus Decimus Meridius", "William Wallace",
	"Jack Sparrow", "Frodo Baggins", "Samwise Gamgee", "Gandalf", "Aragorn",
	"Gollum", "Luke Skywalker", "Leia Organa", "Han Solo", "Darth Vader",
	"Obi-Wan Kenobi", "Yoda", "Neo", "Trinity", "Morpheus", "Tony Stark",
	"Steve Rogers", "Natasha Romanoff", "Bruce Banner", "T'Challa", "Peter Parker",
	"Bruce Wayne", "Clark Kent", "Diana Prince", "Harry Potter", "Hermione Granger",
	"Ron Weasley", "Albus Dumbledore", "Severus Snape", "Katniss Everdeen",
	"Hannibal Lecter", "Clarice Starling", "Tyler Durden", "Andy Dufresne",
	"Ellis Redding", "Jules Winnfield", "Vincent Vega", "Mia Wallace",
	"Walter White", "Amélie Poulain", "Chihiro Ogino", "Totoro", "Princess Mononoke",
	"Woody", "Buzz Lightyear", "Shrek", "Simba", "Mufasa", "Elsa", "Moana",
	"Wall-E", "Remy", "Dory", "Nemo", "Po", "Hiccup",
}

// games are real, widely-played titles.
var games = []string{
	"Minecraft", "Fortnite", "Grand Theft Auto V", "Counter-Strike 2", "Dota 2",
	"League of Legends", "Valorant", "Apex Legends", "Overwatch 2", "Call of Duty: Warzone",
	"PUBG: Battlegrounds", "Rocket League", "Rainbow Six Siege", "Destiny 2",
	"World of Warcraft", "Final Fantasy XIV", "The Elder Scrolls Online", "Path of Exile",
	"Diablo IV", "Elden Ring", "Dark Souls III", "Sekiro: Shadows Die Twice",
	"Bloodborne", "The Witcher 3: Wild Hunt", "Cyberpunk 2077", "Red Dead Redemption 2",
	"The Last of Us Part II", "God of War", "Horizon Zero Dawn", "Ghost of Tsushima",
	"Spider-Man 2", "Uncharted 4", "Assassin's Creed Valhalla", "Far Cry 6",
	"Hollow Knight", "Celeste", "Stardew Valley", "Terraria", "Hades",
	"Slay the Spire", "Baldur's Gate 3", "Divinity: Original Sin 2", "Disco Elysium",
	"Civilization VI", "Total War: Warhammer III", "Age of Empires IV", "StarCraft II",
	"Factorio", "RimWorld", "Cities: Skylines", "The Sims 4", "Animal Crossing: New Horizons",
	"The Legend of Zelda: Tears of the Kingdom", "Super Mario Odyssey", "Mario Kart 8 Deluxe",
	"Super Smash Bros. Ultimate", "Splatoon 3", "Pokémon Scarlet", "Metroid Dread",
	"Portal 2", "Half-Life: Alyx", "Team Fortress 2", "Left 4 Dead 2", "Among Us",
	"Fall Guys", "Roblox", "Genshin Impact", "Honkai: Star Rail", "Clash of Clans",
	"Candy Crush Saga", "EA Sports FC 24", "NBA 2K24", "Forza Horizon 5",
	"Gran Turismo 7", "Microsoft Flight Simulator", "No Man's Sky", "Subnautica",
}

var gamerTags = []string{"x", "xX", "Its", "Real", "The", "Not", "Mr", "Lil", "Big"}

var gamerPrefixes = []string{
	"Dark", "Shadow", "Iron", "Frost", "Blaze", "Storm", "Night", "Silent",
	"Rapid", "Toxic", "Savage", "Ghost", "Cyber", "Neon", "Turbo", "Rogue",
	"Grim", "Swift", "Steel", "Crimson", "Void", "Solar", "Lunar", "Atomic",
	"Phantom", "Vortex", "Chaos", "Alpha", "Omega", "Hyper", "Ultra", "Mega",
}

var gamerNouns = []string{
	"Wolf", "Raven", "Hawk", "Viper", "Sniper", "Slayer", "Hunter", "Reaper",
	"Blade", "Fury", "Storm", "Fang", "Claw", "Strike", "Rider", "Knight",
	"Mage", "Ninja", "Samurai", "Titan", "Phoenix", "Dragon", "Kraken", "Beast",
	"Bolt", "Spark", "Ace", "King", "Lord", "Master", "Legend", "Nomad",
}

// birds is the English fallback; locale versions live in the locale catalog.
var birds = []string{
	"sparrow", "swallow", "starling", "blackbird", "robin", "wren", "finch",
	"goldfinch", "chaffinch", "bullfinch", "siskin", "linnet", "thrush",
	"nightingale", "lark", "skylark", "swift", "house martin", "wagtail",
	"pipit", "warbler", "flycatcher", "tit", "blue tit", "great tit",
	"nuthatch", "treecreeper", "jay", "magpie", "jackdaw", "rook", "crow",
	"raven", "cuckoo", "hoopoe", "kingfisher", "woodpecker", "owl", "barn owl",
	"tawny owl", "eagle owl", "kestrel", "falcon", "peregrine", "sparrowhawk",
	"buzzard", "harrier", "kite", "golden eagle", "osprey", "heron", "egret",
	"stork", "crane", "flamingo", "swan", "goose", "duck", "mallard", "teal",
	"grebe", "cormorant", "gull", "tern", "puffin", "guillemot", "pelican",
	"partridge", "quail", "pheasant", "grouse", "snipe", "curlew", "lapwing",
	"plover", "sandpiper", "dove", "pigeon", "parrot", "peacock",
}

// streetTypes are the road classifications an address dataset needs.
var streetTypes = []string{
	"street", "avenue", "boulevard", "road", "lane", "drive", "court", "place",
	"terrace", "way", "close", "crescent", "square", "circle", "parkway",
	"highway", "freeway", "motorway", "expressway", "bypass", "ring road",
	"alley", "passage", "path", "walk", "row", "gardens", "grove", "park",
	"hill", "rise", "vale", "mews", "wharf", "quay", "embankment", "bridge",
	"tunnel", "junction", "roundabout", "interchange", "service road",
	"dual carriageway", "cul-de-sac", "esplanade", "promenade", "causeway",
}

// diseases are common conditions by their everyday names. The coded form lives
// under KindICD10; this is the column a human-readable record uses.
var diseases = []string{
	"asthma", "bronchitis", "pneumonia", "influenza", "common cold", "sinusitis",
	"tonsillitis", "laryngitis", "tuberculosis", "emphysema", "COPD",
	"hypertension", "coronary artery disease", "heart failure", "arrhythmia",
	"atrial fibrillation", "angina", "myocardial infarction", "stroke",
	"deep vein thrombosis", "varicose veins", "anaemia", "leukaemia", "lymphoma",
	"type 1 diabetes", "type 2 diabetes", "hypothyroidism", "hyperthyroidism",
	"obesity", "gout", "osteoporosis", "osteoarthritis", "rheumatoid arthritis",
	"lupus", "fibromyalgia", "sciatica", "herniated disc", "scoliosis",
	"migraine", "epilepsy", "Parkinson's disease", "Alzheimer's disease",
	"multiple sclerosis", "neuropathy", "vertigo", "insomnia", "sleep apnoea",
	"depression", "anxiety disorder", "bipolar disorder", "schizophrenia",
	"ADHD", "autism spectrum disorder", "eating disorder",
	"gastritis", "peptic ulcer", "acid reflux", "irritable bowel syndrome",
	"Crohn's disease", "ulcerative colitis", "coeliac disease", "hepatitis",
	"cirrhosis", "gallstones", "pancreatitis", "appendicitis", "haemorrhoids",
	"kidney stones", "chronic kidney disease", "urinary tract infection",
	"prostatitis", "endometriosis", "polycystic ovary syndrome",
	"eczema", "psoriasis", "acne", "rosacea", "dermatitis", "urticaria",
	"cellulitis", "shingles", "chickenpox", "measles", "mumps", "rubella",
	"conjunctivitis", "glaucoma", "cataract", "macular degeneration",
	"otitis media", "tinnitus", "hearing loss", "allergic rhinitis",
	"food allergy", "anaphylaxis", "malaria", "dengue fever", "typhoid fever",
}
