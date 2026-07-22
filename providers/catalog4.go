package providers

import "github.com/bakhod1r/synth/schema"

// Catalog batch 4: media, nature, medicine, transport and civic life.
//
// Every list here is real. A made-up dinosaur or a fictional medical specialty
// is spotted at once by anyone who works in the field, and that is the moment a
// fixture stops being trusted.

func init() {
	set(schema.KindTVShow, tvShows)
	set(schema.KindAnime, animeTitles)
	set(schema.KindPodcast, podcasts)
	set(schema.KindNewspaper, newspapers)
	set(schema.KindMusicAlbum, musicAlbums)
	set(schema.KindBoardGame, boardGames)
	set(schema.KindConsole, consoles)

	set(schema.KindRestaurant, restaurants)
	set(schema.KindClothingBrand, clothingBrands)
	set(schema.KindWatchBrand, watchBrands)
	set(schema.KindCameraModel, cameraModels)
	set(schema.KindPhoneModel, phoneModels)

	setLocalized(schema.KindTree, trees)
	setLocalized(schema.KindInsect, insects)
	setLocalized(schema.KindFish, fishes)
	setLocalized(schema.KindHerb, herbs)
	setLocalized(schema.KindSpice, spices)
	set(schema.KindDinosaur, dinosaurs)
	set(schema.KindMineral, minerals)

	set(schema.KindMedicalSpecialty, medicalSpecialties)
	set(schema.KindLabTest, labTests)
	set(schema.KindProcedure, procedures)
	set(schema.KindVaccine, vaccines)
	set(schema.KindAllergy, allergies)
	set(schema.KindSymptom, symptoms)

	set(schema.KindMilitaryRank, militaryRanks)
	set(schema.KindReligion, religions)
	set(schema.KindHoliday, holidays)
	set(schema.KindInsuranceType, insuranceTypes)
	set(schema.KindCrimeType, crimeTypes)
	set(schema.KindTaxType, taxTypes)

	set(schema.KindSpacecraft, spacecraft)
	set(schema.KindMotorcycle, motorcycles)
	set(schema.KindBicycleType, bicycleTypes)
	set(schema.KindFuelType, fuelTypes)
	set(schema.KindNaturalDisaster, naturalDisasters)
	set(schema.KindMartialArt, martialArts)
	set(schema.KindDance, dances)
	set(schema.KindChessOpening, chessOpenings)
	set(schema.KindWineVariety, wineVarieties)
	set(schema.KindCheese, cheeses)
}

var tvShows = []string{
	"Breaking Bad", "The Sopranos", "The Wire", "Mad Men", "The Crown",
	"Game of Thrones", "Stranger Things", "Better Call Saul", "Succession",
	"Chernobyl", "Band of Brothers", "The Office", "Friends", "Seinfeld",
	"Parks and Recreation", "Brooklyn Nine-Nine", "Arrested Development",
	"Fargo", "True Detective", "Sherlock", "Black Mirror", "Peaky Blinders",
	"The Mandalorian", "House of the Dragon", "The Last of Us", "Severance",
	"Ted Lasso", "The Bear", "Yellowstone", "Ozark", "Narcos", "Money Heist",
	"Dark", "Squid Game", "Lupin", "Call My Agent", "Borgen", "The Bridge",
	"Doctor Who", "Line of Duty", "Downton Abbey", "The Great British Bake Off",
	"Planet Earth", "Blue Planet", "Cosmos", "House", "Grey's Anatomy", "ER",
	"Law & Order", "CSI", "The X-Files", "Lost", "24", "Prison Break",
	"The Walking Dead", "Westworld", "Rick and Morty", "The Simpsons",
	"South Park", "Futurama", "BoJack Horseman", "Avatar: The Last Airbender",
}

var animeTitles = []string{
	"Naruto", "One Piece", "Bleach", "Dragon Ball Z", "Attack on Titan",
	"Death Note", "Fullmetal Alchemist: Brotherhood", "Steins;Gate", "Code Geass",
	"Cowboy Bebop", "Neon Genesis Evangelion", "Ghost in the Shell", "Akira",
	"Spirited Away", "My Neighbour Totoro", "Princess Mononoke", "Howl's Moving Castle",
	"Grave of the Fireflies", "Your Name", "Weathering with You", "Suzume",
	"Demon Slayer", "Jujutsu Kaisen", "My Hero Academia", "Chainsaw Man",
	"Spy × Family", "Vinland Saga", "Mob Psycho 100", "One Punch Man",
	"Hunter × Hunter", "Monster", "Berserk", "Samurai Champloo", "Trigun",
	"Gurren Lagann", "Made in Abyss", "Violet Evergarden", "A Silent Voice",
	"Clannad", "Anohana", "March Comes in Like a Lion", "Haikyuu!!",
	"Slam Dunk", "Yuri!!! on Ice", "Sailor Moon", "Cardcaptor Sakura",
	"Pokémon", "Digimon Adventure", "Doraemon", "Detective Conan", "Inuyasha",
}

var podcasts = []string{
	"Serial", "This American Life", "Radiolab", "99% Invisible", "Planet Money",
	"Freakonomics Radio", "Hardcore History", "Revolutions", "The Daily",
	"Reply All", "Darknet Diaries", "Syntax", "Software Engineering Daily",
	"The Changelog", "Go Time", "Talk Python To Me", "CoRecursive",
	"Lex Fridman Podcast", "The Tim Ferriss Show", "How I Built This",
	"Masters of Scale", "a16z Podcast", "Acquired", "Business Wars",
	"The Rest Is History", "In Our Time", "The Infinite Monkey Cage",
	"No Such Thing As A Fish", "Stuff You Should Know", "Science Vs",
	"Ologies", "The Naked Scientists", "Nature Podcast", "Short Wave",
	"My Favorite Murder", "Criminal", "Casefile", "S-Town", "Dirty John",
	"The Anthropocene Reviewed", "Song Exploder", "Switched on Pop",
	"Desert Island Discs", "The Moth", "Sleep With Me", "Philosophize This",
}

var newspapers = []string{
	"The New York Times", "The Washington Post", "The Wall Street Journal",
	"Los Angeles Times", "Chicago Tribune", "USA Today", "The Boston Globe",
	"The Guardian", "The Times", "The Daily Telegraph", "The Independent",
	"Financial Times", "The Observer", "Le Monde", "Le Figaro", "Libération",
	"Frankfurter Allgemeine Zeitung", "Süddeutsche Zeitung", "Die Zeit", "Die Welt",
	"Corriere della Sera", "La Repubblica", "La Stampa", "El País", "El Mundo",
	"La Vanguardia", "ABC", "Público", "Folha de S.Paulo", "O Globo",
	"Clarín", "La Nación", "El Universal", "The Globe and Mail", "Toronto Star",
	"The Sydney Morning Herald", "The Age", "The Australian",
	"The Times of India", "The Hindu", "Hindustan Times", "Dawn",
	"Asahi Shimbun", "Yomiuri Shimbun", "The Japan Times", "South China Morning Post",
	"The Straits Times", "Gazeta Wyborcza", "Rzeczpospolita", "Hürriyet",
	"Milliyet", "Kommersant", "Vedomosti", "Xalq so'zi", "Aftenposten",
	"Dagens Nyheter", "Helsingin Sanomat", "Politiken", "NRC Handelsblad",
}

var musicAlbums = []string{
	"The Dark Side of the Moon", "The Wall", "Wish You Were Here", "Abbey Road",
	"Revolver", "Sgt. Pepper's Lonely Hearts Club Band", "Rubber Soul",
	"Pet Sounds", "Highway 61 Revisited", "Blonde on Blonde", "Blood on the Tracks",
	"Led Zeppelin IV", "Physical Graffiti", "Exile on Main St.", "Sticky Fingers",
	"Who's Next", "A Night at the Opera", "Rumours", "Hotel California",
	"Born to Run", "Nebraska", "Thriller", "Off the Wall", "Purple Rain",
	"What's Going On", "Songs in the Key of Life", "Innervisions", "Kind of Blue",
	"A Love Supreme", "Time Out", "Giant Steps", "Blue Train", "Mingus Ah Um",
	"The Velvet Underground & Nico", "Horses", "London Calling", "Never Mind the Bollocks",
	"Unknown Pleasures", "Closer", "Remain in Light", "Talking Heads: 77",
	"Nevermind", "In Utero", "OK Computer", "The Bends", "Kid A",
	"Definitely Maybe", "(What's the Story) Morning Glory?", "Parachutes",
	"Illmatic", "The Chronic", "Ready to Die", "To Pimp a Butterfly",
	"good kid, m.A.A.d city", "The Miseducation of Lauryn Hill", "Aquemini",
	"Discovery", "Random Access Memories", "Selected Ambient Works 85-92",
	"Music Has the Right to Children", "Loveless", "Doolittle", "Surfer Rosa",
}

var boardGames = []string{
	"Chess", "Go", "Backgammon", "Draughts", "Shogi", "Xiangqi", "Mancala",
	"Monopoly", "Scrabble", "Cluedo", "Risk", "Trivial Pursuit", "Battleship",
	"Connect Four", "Othello", "Stratego", "Catan", "Carcassonne", "Ticket to Ride",
	"Pandemic", "7 Wonders", "Dominion", "Agricola", "Puerto Rico", "Power Grid",
	"Terraforming Mars", "Wingspan", "Brass: Birmingham", "Gloomhaven",
	"Twilight Struggle", "Scythe", "Root", "Spirit Island", "Everdell",
	"Azul", "Splendor", "Codenames", "Dixit", "Hanabi", "The Crew",
	"Blood Rage", "Betrayal at House on the Hill", "Arkham Horror",
	"Dead of Winter", "Small World", "Kingdomino", "Patchwork", "Jaipur",
	"Lost Cities", "Sushi Go!", "Love Letter", "Coup", "The Resistance",
}

var consoles = []string{
	"PlayStation 5", "PlayStation 5 Pro", "PlayStation 4", "PlayStation 3",
	"PlayStation 2", "PlayStation Portable", "PlayStation Vita",
	"Xbox Series X", "Xbox Series S", "Xbox One", "Xbox 360", "Xbox",
	"Nintendo Switch", "Nintendo Switch OLED", "Nintendo Switch Lite",
	"Wii U", "Wii", "GameCube", "Nintendo 64", "Super Nintendo",
	"Nintendo Entertainment System", "Game Boy", "Game Boy Advance",
	"Nintendo DS", "Nintendo 3DS", "Sega Genesis", "Sega Saturn", "Dreamcast",
	"Atari 2600", "Neo Geo", "Steam Deck", "ROG Ally", "Analogue Pocket",
}

var restaurants = []string{
	"McDonald's", "Burger King", "KFC", "Subway", "Starbucks", "Domino's Pizza",
	"Pizza Hut", "Papa John's", "Wendy's", "Taco Bell", "Chipotle", "Five Guys",
	"Shake Shack", "In-N-Out Burger", "Dunkin'", "Costa Coffee", "Pret a Manger",
	"Nando's", "Wagamama", "Pizza Express", "Greggs", "Tim Hortons",
	"Krispy Kreme", "Baskin-Robbins", "Cinnabon", "Panera Bread", "Chick-fil-A",
	"Popeyes", "Jollibee", "Yoshinoya", "Mos Burger", "CoCo Ichibanya",
	"Din Tai Fung", "Haidilao", "Vapiano", "L'Osteria", "Paul", "Ladurée",
	"Le Pain Quotidien", "Hard Rock Cafe", "TGI Fridays", "Outback Steakhouse",
	"Olive Garden", "Red Lobster", "IHOP", "Denny's", "Waffle House",
}

var clothingBrands = []string{
	"Nike", "Adidas", "Puma", "Reebok", "New Balance", "Asics", "Under Armour",
	"Converse", "Vans", "Fila", "Champion", "The North Face", "Patagonia",
	"Columbia", "Arc'teryx", "Salomon", "Levi's", "Wrangler", "Lee", "Diesel",
	"Zara", "H&M", "Uniqlo", "Mango", "Bershka", "Pull&Bear", "COS", "Arket",
	"Gap", "Old Navy", "Banana Republic", "J.Crew", "Ralph Lauren",
	"Tommy Hilfiger", "Calvin Klein", "Lacoste", "Hugo Boss", "Armani",
	"Gucci", "Prada", "Versace", "Balenciaga", "Burberry", "Louis Vuitton",
	"Hermès", "Chanel", "Dior", "Fendi", "Moncler", "Stone Island",
	"Supreme", "Carhartt", "Dickies", "Timberland", "Dr. Martens", "Clarks",
}

var watchBrands = []string{
	"Rolex", "Omega", "Patek Philippe", "Audemars Piguet", "Vacheron Constantin",
	"Jaeger-LeCoultre", "IWC Schaffhausen", "Breitling", "TAG Heuer", "Cartier",
	"Panerai", "Hublot", "Zenith", "Blancpain", "Breguet", "A. Lange & Söhne",
	"Glashütte Original", "Nomos Glashütte", "Grand Seiko", "Seiko", "Citizen",
	"Casio", "G-Shock", "Orient", "Tissot", "Longines", "Rado", "Oris",
	"Hamilton", "Certina", "Mido", "Frederique Constant", "Bell & Ross",
	"Sinn", "Junghans", "Swatch", "Fossil", "Timex", "Garmin", "Suunto",
	"Polar", "Apple Watch", "Withings", "Tudor", "Bremont", "Christopher Ward",
}

var cameraModels = []string{
	"Canon EOS R5", "Canon EOS R6 Mark II", "Canon EOS R8", "Canon EOS 5D Mark IV",
	"Canon EOS 90D", "Nikon Z8", "Nikon Z9", "Nikon Z6 III", "Nikon Z fc",
	"Nikon D850", "Sony A7 IV", "Sony A7R V", "Sony A7S III", "Sony A9 III",
	"Sony A6700", "Sony ZV-E1", "Fujifilm X-T5", "Fujifilm X-H2S",
	"Fujifilm X100VI", "Fujifilm GFX 100 II", "Panasonic Lumix S5 II",
	"Panasonic Lumix GH6", "Panasonic Lumix G9 II", "Olympus OM-1",
	"OM System OM-5", "Leica M11", "Leica Q3", "Leica SL3", "Pentax K-3 III",
	"Ricoh GR III", "Hasselblad X2D 100C", "Phase One XF IQ4",
	"GoPro HERO12 Black", "DJI Osmo Pocket 3", "DJI Air 3", "Insta360 X4",
	"Blackmagic Pocket Cinema 6K", "RED Komodo 6K", "ARRI Alexa 35",
	"Sigma fp L", "Canon PowerShot G7 X Mark III", "Sony RX100 VII",
}

var phoneModels = []string{
	"iPhone 15 Pro Max", "iPhone 15 Pro", "iPhone 15", "iPhone 14", "iPhone SE",
	"Samsung Galaxy S24 Ultra", "Samsung Galaxy S24+", "Samsung Galaxy S24",
	"Samsung Galaxy Z Fold5", "Samsung Galaxy Z Flip5", "Samsung Galaxy A54",
	"Google Pixel 8 Pro", "Google Pixel 8", "Google Pixel 8a", "Google Pixel Fold",
	"OnePlus 12", "OnePlus Nord 3", "Xiaomi 14 Ultra", "Xiaomi 14",
	"Xiaomi Redmi Note 13 Pro", "Poco X6 Pro", "Oppo Find X7 Ultra",
	"Oppo Reno11", "Vivo X100 Pro", "Honor Magic6 Pro", "Huawei P60 Pro",
	"Huawei Mate 60 Pro", "Nothing Phone (2)", "Nothing Phone (2a)",
	"Motorola Edge 50 Pro", "Motorola Razr 40 Ultra", "Sony Xperia 1 V",
	"Asus ROG Phone 8", "Asus Zenfone 10", "Fairphone 5", "Nokia XR21",
}

var trees = []string{
	"oak", "beech", "birch", "maple", "ash", "elm", "lime", "hornbeam",
	"sycamore", "chestnut", "walnut", "hazel", "alder", "willow", "poplar",
	"aspen", "rowan", "hawthorn", "blackthorn", "cherry", "apple", "pear",
	"plum", "apricot", "peach", "quince", "fig", "mulberry", "olive",
	"pine", "spruce", "fir", "larch", "cedar", "juniper", "yew", "cypress",
	"redwood", "sequoia", "eucalyptus", "acacia", "baobab", "banyan",
	"palm", "date palm", "coconut palm", "bamboo", "magnolia", "catalpa",
	"tulip tree", "ginkgo", "plane tree", "horse chestnut", "elder", "holly",
}

var insects = []string{
	"ant", "bee", "bumblebee", "wasp", "hornet", "butterfly", "moth",
	"dragonfly", "damselfly", "mayfly", "grasshopper", "cricket", "locust",
	"cicada", "beetle", "ladybird", "stag beetle", "dung beetle", "weevil",
	"firefly", "cockroach", "termite", "mantis", "stick insect", "earwig",
	"aphid", "mosquito", "midge", "housefly", "hoverfly", "horsefly",
	"flea", "louse", "silverfish", "springtail", "lacewing", "caddisfly",
	"scarab", "leafhopper", "shield bug", "water strider", "diving beetle",
}

var fishes = []string{
	"salmon", "trout", "carp", "pike", "perch", "bream", "roach", "tench",
	"catfish", "sturgeon", "eel", "herring", "sardine", "anchovy", "mackerel",
	"tuna", "cod", "haddock", "pollock", "hake", "halibut", "plaice", "sole",
	"turbot", "sea bass", "sea bream", "mullet", "snapper", "grouper",
	"barracuda", "swordfish", "marlin", "shark", "ray", "skate", "flounder",
	"sprat", "whitebait", "grayling", "char", "zander", "burbot", "gudgeon",
	"rudd", "chub", "barbel", "dace", "smelt", "lamprey", "monkfish",
}

var herbs = []string{
	"basil", "parsley", "coriander", "dill", "mint", "oregano", "thyme",
	"rosemary", "sage", "tarragon", "chervil", "marjoram", "bay leaf",
	"lemon balm", "lovage", "savory", "sorrel", "chives", "fennel",
	"lemongrass", "curry leaf", "kaffir lime leaf", "shiso", "epazote",
	"borage", "hyssop", "rue", "wormwood", "st john's wort", "chamomile",
	"lavender", "nettle", "plantain", "yarrow", "elderflower", "linden",
}

var spices = []string{
	"black pepper", "white pepper", "cumin", "coriander seed", "turmeric",
	"paprika", "chilli", "cayenne", "cinnamon", "cassia", "clove", "nutmeg",
	"mace", "cardamom", "star anise", "anise", "fennel seed", "caraway",
	"mustard seed", "fenugreek", "ginger", "galangal", "saffron", "sumac",
	"za'atar", "ras el hanout", "garam masala", "curry powder", "juniper berry",
	"allspice", "vanilla", "asafoetida", "nigella seed", "poppy seed",
	"sesame", "peppercorn", "long pepper", "grains of paradise", "annatto",
}

var dinosaurs = []string{
	"Tyrannosaurus", "Triceratops", "Stegosaurus", "Velociraptor", "Brachiosaurus",
	"Diplodocus", "Apatosaurus", "Allosaurus", "Ankylosaurus", "Spinosaurus",
	"Parasaurolophus", "Iguanodon", "Pachycephalosaurus", "Archaeopteryx",
	"Compsognathus", "Deinonychus", "Utahraptor", "Giganotosaurus",
	"Carnotaurus", "Therizinosaurus", "Oviraptor", "Gallimimus", "Struthiomimus",
	"Protoceratops", "Styracosaurus", "Torosaurus", "Edmontosaurus",
	"Maiasaura", "Corythosaurus", "Argentinosaurus", "Titanosaurus",
	"Mamenchisaurus", "Camarasaurus", "Dilophosaurus", "Ceratosaurus",
	"Megalosaurus", "Baryonyx", "Suchomimus", "Microraptor", "Sinosauropteryx",
	"Pteranodon", "Quetzalcoatlus", "Plesiosaurus", "Mosasaurus", "Ichthyosaurus",
}

var minerals = []string{
	"quartz", "feldspar", "mica", "calcite", "dolomite", "gypsum", "halite",
	"fluorite", "apatite", "pyrite", "magnetite", "hematite", "galena",
	"sphalerite", "chalcopyrite", "bauxite", "cassiterite", "malachite",
	"azurite", "olivine", "garnet", "tourmaline", "beryl", "topaz", "zircon",
	"corundum", "spinel", "kyanite", "andalusite", "staurolite", "epidote",
	"talc", "serpentine", "chlorite", "kaolinite", "montmorillonite",
	"barite", "celestine", "anhydrite", "graphite", "sulphur", "cinnabar",
	"molybdenite", "wolframite", "scheelite", "rutile", "ilmenite", "chromite",
}

var medicalSpecialties = []string{
	"anaesthesiology", "cardiology", "cardiothoracic surgery", "dermatology",
	"emergency medicine", "endocrinology", "gastroenterology", "general practice",
	"general surgery", "geriatrics", "haematology", "hepatology",
	"immunology", "infectious diseases", "internal medicine", "nephrology",
	"neurology", "neurosurgery", "obstetrics and gynaecology", "oncology",
	"ophthalmology", "orthopaedics", "otolaryngology", "paediatrics",
	"pathology", "plastic surgery", "psychiatry", "pulmonology", "radiology",
	"rheumatology", "sports medicine", "urology", "vascular surgery",
	"palliative care", "occupational medicine", "clinical genetics",
	"nuclear medicine", "intensive care medicine", "public health",
}

var labTests = []string{
	"complete blood count", "basic metabolic panel", "comprehensive metabolic panel",
	"lipid panel", "liver function tests", "kidney function tests",
	"thyroid function tests", "haemoglobin A1c", "fasting glucose",
	"oral glucose tolerance test", "C-reactive protein",
	"erythrocyte sedimentation rate", "prothrombin time", "INR",
	"D-dimer", "troponin", "BNP", "creatine kinase", "ferritin",
	"vitamin B12", "folate", "vitamin D", "iron studies", "urinalysis",
	"urine culture", "blood culture", "stool culture", "throat swab",
	"COVID-19 PCR", "influenza rapid test", "HIV antibody test",
	"hepatitis B surface antigen", "hepatitis C antibody", "syphilis serology",
	"PSA", "CA-125", "CEA", "alpha-fetoprotein", "cortisol", "testosterone",
	"oestradiol", "FSH", "LH", "parathyroid hormone", "arterial blood gas",
}

var procedures = []string{
	"appendectomy", "cholecystectomy", "hernia repair", "hip replacement",
	"knee replacement", "coronary artery bypass graft", "angioplasty",
	"pacemaker implantation", "cataract surgery", "LASIK", "tonsillectomy",
	"adenoidectomy", "septoplasty", "thyroidectomy", "mastectomy",
	"lumpectomy", "prostatectomy", "hysterectomy", "caesarean section",
	"colonoscopy", "gastroscopy", "bronchoscopy", "cystoscopy", "arthroscopy",
	"laparoscopy", "biopsy", "endoscopic retrograde cholangiopancreatography",
	"dialysis", "kidney transplant", "liver transplant", "bone marrow transplant",
	"skin graft", "spinal fusion", "laminectomy", "craniotomy", "tracheostomy",
	"gastric bypass", "sleeve gastrectomy", "vasectomy", "tubal ligation",
}

var vaccines = []string{
	"BCG", "hepatitis B", "hepatitis A", "DTaP", "Tdap", "polio (IPV)",
	"Haemophilus influenzae type b", "pneumococcal conjugate",
	"pneumococcal polysaccharide", "rotavirus", "measles, mumps and rubella",
	"varicella", "influenza", "COVID-19 mRNA", "human papillomavirus",
	"meningococcal ACWY", "meningococcal B", "typhoid", "yellow fever",
	"rabies", "Japanese encephalitis", "tick-borne encephalitis", "cholera",
	"shingles", "RSV", "dengue", "malaria (RTS,S)", "smallpox",
}

var allergies = []string{
	"peanut", "tree nut", "milk", "egg", "soy", "wheat", "gluten", "fish",
	"shellfish", "sesame", "mustard", "celery", "lupin", "sulphites",
	"pollen", "grass pollen", "tree pollen", "ragweed", "dust mite",
	"cat dander", "dog dander", "mould", "latex", "bee sting", "wasp sting",
	"penicillin", "sulfonamides", "aspirin", "NSAIDs", "iodine contrast",
	"nickel", "fragrance", "formaldehyde", "chlorhexidine",
}

var symptoms = []string{
	"fever", "chills", "fatigue", "headache", "dizziness", "nausea", "vomiting",
	"diarrhoea", "constipation", "abdominal pain", "chest pain", "palpitations",
	"shortness of breath", "cough", "sore throat", "runny nose", "sneezing",
	"loss of smell", "loss of taste", "muscle ache", "joint pain", "back pain",
	"rash", "itching", "swelling", "bruising", "night sweats", "weight loss",
	"weight gain", "loss of appetite", "insomnia", "confusion", "memory loss",
	"blurred vision", "double vision", "hearing loss", "tinnitus", "numbness",
	"tingling", "weakness", "tremor", "seizure", "fainting", "jaundice",
	"blood in stool", "blood in urine", "frequent urination", "painful urination",
}

var militaryRanks = []string{
	"Private", "Private First Class", "Lance Corporal", "Corporal", "Sergeant",
	"Staff Sergeant", "Sergeant First Class", "Master Sergeant", "First Sergeant",
	"Sergeant Major", "Warrant Officer", "Chief Warrant Officer",
	"Second Lieutenant", "First Lieutenant", "Captain", "Major",
	"Lieutenant Colonel", "Colonel", "Brigadier General", "Major General",
	"Lieutenant General", "General", "Field Marshal",
	"Seaman", "Petty Officer", "Chief Petty Officer", "Ensign",
	"Lieutenant Commander", "Commander", "Rear Admiral", "Vice Admiral",
	"Admiral", "Fleet Admiral", "Airman", "Flight Sergeant", "Flying Officer",
	"Squadron Leader", "Wing Commander", "Group Captain", "Air Commodore",
	"Air Vice-Marshal", "Air Marshal", "Air Chief Marshal",
}

var religions = []string{
	"Christianity", "Catholicism", "Protestantism", "Orthodox Christianity",
	"Anglicanism", "Lutheranism", "Methodism", "Baptism", "Presbyterianism",
	"Islam", "Sunni Islam", "Shia Islam", "Sufism", "Judaism",
	"Orthodox Judaism", "Reform Judaism", "Hinduism", "Buddhism",
	"Theravada Buddhism", "Mahayana Buddhism", "Zen Buddhism", "Sikhism",
	"Jainism", "Taoism", "Confucianism", "Shinto", "Zoroastrianism",
	"Baháʼí Faith", "Rastafari", "Paganism", "Animism", "Agnosticism",
	"Atheism", "Humanism", "Unitarian Universalism", "Quakerism",
	"Eastern Orthodoxy", "Oriental Orthodoxy", "Coptic Christianity",
	"Armenian Apostolic Church", "Georgian Orthodox Church", "Assyrian Church of the East",
	"Maronite Church", "Greek Catholic Church", "Pentecostalism", "Evangelicalism",
	"Adventism", "Mormonism", "Jehovah's Witnesses", "Amish", "Mennonitism",
	"Ibadi Islam", "Alevism", "Ahmadiyya", "Druze", "Yazidism", "Alawism",
	"Naqshbandi Sufism", "Yasawi Sufism", "Conservative Judaism", "Karaite Judaism",
	"Hasidic Judaism", "Vaishnavism", "Shaivism", "Shaktism", "Smartism",
	"Vajrayana Buddhism", "Tibetan Buddhism", "Pure Land Buddhism",
	"Nichiren Buddhism", "Tendai", "Shingon", "Cao Dai", "Hoa Hao",
	"Tenrikyo", "Cheondoism", "Chondogyo", "Falun Gong", "Wicca", "Druidry",
	"Ásatrú", "Tengrism", "Shamanism", "Candomblé", "Santería", "Vodou",
	"Sikh Khalsa", "Ravidassia", "Meivazhi", "Ayyavazhi", "Manichaeism",
	"Mandaeism", "Samaritanism", "Deism", "Pantheism", "Spiritualism",
	"New Age", "Scientology", "Raëlism", "Eckankar", "Theosophy",
	"Traditional African religions", "Native American Church", "Bon",
	"Irreligion", "Secular Humanism", "Agnostic Atheism",
}

var holidays = []string{
	"New Year's Day", "Epiphany", "Valentine's Day", "Carnival", "Shrove Tuesday",
	"Ash Wednesday", "St Patrick's Day", "Nowruz", "Good Friday", "Easter",
	"Passover", "Ramadan", "Eid al-Fitr", "Eid al-Adha", "Labour Day",
	"Victory Day", "Mother's Day", "Father's Day", "Midsummer", "Independence Day",
	"Bastille Day", "Assumption Day", "Rosh Hashanah", "Yom Kippur", "Sukkot",
	"Diwali", "Halloween", "All Saints' Day", "Day of the Dead", "Guy Fawkes Night",
	"Thanksgiving", "Black Friday", "Hanukkah", "Saint Nicholas Day",
	"Christmas Eve", "Christmas Day", "Boxing Day", "New Year's Eve",
	"Lunar New Year", "Mid-Autumn Festival", "Songkran", "Obon", "Vesak",
	"Navruz", "Kurban Hayit", "Constitution Day", "Republic Day",
}

var insuranceTypes = []string{
	"life insurance", "term life insurance", "whole life insurance",
	"health insurance", "dental insurance", "vision insurance",
	"critical illness cover", "income protection", "disability insurance",
	"travel insurance", "car insurance", "third-party liability",
	"comprehensive motor insurance", "motorcycle insurance", "home insurance",
	"buildings insurance", "contents insurance", "landlord insurance",
	"tenant insurance", "pet insurance", "business insurance",
	"public liability insurance", "professional indemnity insurance",
	"employers' liability insurance", "cyber insurance", "marine cargo insurance",
	"crop insurance", "credit insurance", "title insurance", "reinsurance",
}

var crimeTypes = []string{
	"theft", "burglary", "robbery", "shoplifting", "pickpocketing", "fraud",
	"identity theft", "credit card fraud", "money laundering", "embezzlement",
	"tax evasion", "bribery", "corruption", "extortion", "blackmail",
	"forgery", "counterfeiting", "smuggling", "trafficking", "arson",
	"vandalism", "criminal damage", "trespassing", "assault", "battery",
	"harassment", "stalking", "kidnapping", "false imprisonment", "homicide",
	"manslaughter", "drink driving", "dangerous driving", "hit and run",
	"drug possession", "drug trafficking", "illegal firearm possession",
	"cybercrime", "hacking", "phishing", "ransomware", "copyright infringement",
	"perjury", "obstruction of justice", "public disorder", "loitering",
}

var taxTypes = []string{
	"income tax", "corporate tax", "capital gains tax", "dividend tax",
	"value added tax", "goods and services tax", "sales tax", "excise duty",
	"customs duty", "import tariff", "property tax", "land tax", "council tax",
	"stamp duty", "inheritance tax", "estate tax", "gift tax", "wealth tax",
	"payroll tax", "social security contribution", "national insurance",
	"withholding tax", "environmental tax", "carbon tax", "fuel duty",
	"vehicle excise duty", "road tax", "tourist tax", "sugar tax",
	"alcohol duty", "tobacco duty", "gambling duty", "digital services tax",
}

var spacecraft = []string{
	"Sputnik 1", "Vostok 1", "Mercury-Redstone 3", "Gemini 4", "Apollo 11",
	"Apollo 13", "Apollo 17", "Soyuz", "Salyut 1", "Skylab", "Mir",
	"International Space Station", "Tiangong", "Space Shuttle Columbia",
	"Space Shuttle Challenger", "Space Shuttle Discovery", "Space Shuttle Atlantis",
	"Space Shuttle Endeavour", "Buran", "Voyager 1", "Voyager 2", "Pioneer 10",
	"Pioneer 11", "New Horizons", "Cassini-Huygens", "Galileo", "Juno",
	"Magellan", "Viking 1", "Viking 2", "Mars Pathfinder", "Spirit",
	"Opportunity", "Curiosity", "Perseverance", "Ingenuity", "InSight",
	"Zhurong", "Chandrayaan-3", "Chang'e 5", "Luna 9", "Hayabusa2",
	"OSIRIS-REx", "Rosetta", "Philae", "Parker Solar Probe", "SOHO",
	"Hubble Space Telescope", "James Webb Space Telescope", "Kepler", "TESS",
	"Crew Dragon", "Cargo Dragon", "Starship", "Falcon 9", "Falcon Heavy",
	"Ariane 5", "Ariane 6", "Soyuz-2", "Proton-M", "Long March 5", "H-IIA",
}

var motorcycles = []string{
	"Harley-Davidson Sportster", "Harley-Davidson Fat Boy", "Harley-Davidson Road King",
	"Honda CB500F", "Honda CBR1000RR-R", "Honda Africa Twin", "Honda Gold Wing",
	"Honda Rebel 500", "Yamaha MT-07", "Yamaha MT-09", "Yamaha YZF-R1",
	"Yamaha Ténéré 700", "Yamaha Tracer 9", "Kawasaki Ninja ZX-10R",
	"Kawasaki Z900", "Kawasaki Versys 650", "Kawasaki Vulcan S",
	"Suzuki GSX-R1000", "Suzuki V-Strom 650", "Suzuki SV650", "Suzuki Hayabusa",
	"BMW R 1250 GS", "BMW S 1000 RR", "BMW F 900 R", "BMW R nineT",
	"Ducati Panigale V4", "Ducati Monster", "Ducati Multistrada V4",
	"Ducati Scrambler", "KTM 390 Duke", "KTM 890 Adventure", "KTM 1290 Super Duke R",
	"Triumph Bonneville T120", "Triumph Street Triple", "Triumph Tiger 900",
	"Royal Enfield Classic 350", "Royal Enfield Himalayan", "Aprilia RS 660",
	"Moto Guzzi V7", "MV Agusta Brutale", "Indian Scout", "Zero SR/F",
}

var bicycleTypes = []string{
	"road bike", "endurance road bike", "aero road bike", "time trial bike",
	"triathlon bike", "gravel bike", "cyclocross bike", "touring bike",
	"randonneur", "commuter bike", "hybrid bike", "city bike", "dutch bike",
	"folding bike", "cargo bike", "longtail cargo bike", "mountain bike",
	"hardtail mountain bike", "full-suspension mountain bike", "cross-country bike",
	"trail bike", "enduro bike", "downhill bike", "fat bike", "dirt jump bike",
	"BMX", "track bike", "fixed-gear bike", "single-speed bike", "recumbent",
	"tandem", "e-bike", "e-mountain bike", "e-cargo bike", "children's bike",
	"balance bike", "unicycle", "trike",
}

var fuelTypes = []string{
	"petrol", "unleaded 95", "unleaded 98", "super unleaded", "diesel",
	"premium diesel", "biodiesel", "B7 diesel", "E5 petrol", "E10 petrol",
	"E85 ethanol", "compressed natural gas", "liquefied natural gas",
	"liquefied petroleum gas", "autogas", "hydrogen", "electric",
	"plug-in hybrid", "mild hybrid", "full hybrid", "kerosene", "jet A-1",
	"aviation gasoline", "marine gas oil", "heavy fuel oil", "methanol",
	"synthetic fuel", "hydrotreated vegetable oil",
}

var naturalDisasters = []string{
	"earthquake", "aftershock", "tsunami", "volcanic eruption", "lahar",
	"pyroclastic flow", "landslide", "mudslide", "rockfall", "avalanche",
	"sinkhole", "flood", "flash flood", "river flood", "coastal flood",
	"storm surge", "hurricane", "typhoon", "cyclone", "tornado", "waterspout",
	"derecho", "thunderstorm", "hailstorm", "blizzard", "ice storm",
	"drought", "heatwave", "cold wave", "wildfire", "bushfire", "dust storm",
	"sandstorm", "famine", "epidemic", "pandemic", "locust swarm",
	"solar flare", "geomagnetic storm", "meteorite impact",
}

var martialArts = []string{
	"karate", "judo", "aikido", "kendo", "kyudo", "sumo", "jujutsu",
	"Brazilian jiu-jitsu", "taekwondo", "hapkido", "kung fu", "wing chun",
	"tai chi", "shaolin kung fu", "wushu", "sanda", "muay thai", "kickboxing",
	"boxing", "savate", "capoeira", "krav maga", "systema", "sambo",
	"wrestling", "freestyle wrestling", "Greco-Roman wrestling", "kurash",
	"pahlavani", "silat", "escrima", "arnis", "kalaripayattu", "bokator",
	"lethwei", "mixed martial arts", "pankration", "glima", "bartitsu",
}

var dances = []string{
	"waltz", "viennese waltz", "tango", "argentine tango", "foxtrot",
	"quickstep", "cha-cha-cha", "rumba", "samba", "jive", "paso doble",
	"salsa", "bachata", "merengue", "cumbia", "mambo", "swing", "lindy hop",
	"charleston", "jitterbug", "boogie-woogie", "ballet", "contemporary",
	"modern dance", "jazz dance", "tap dance", "hip hop", "breaking",
	"popping", "locking", "krump", "house dance", "voguing", "flamenco",
	"sevillanas", "irish stepdance", "highland fling", "polka", "mazurka",
	"csárdás", "hora", "sirtaki", "kalinka", "lezginka", "lazgi", "belly dance",
	"bhangra", "kathak", "bharatanatyam", "kabuki dance", "haka",
}

var chessOpenings = []string{
	"Ruy Lopez", "Italian Game", "Scotch Game", "Vienna Game", "King's Gambit",
	"Evans Gambit", "Four Knights Game", "Petrov's Defence", "Philidor Defence",
	"Sicilian Defence", "Sicilian Najdorf", "Sicilian Dragon", "Sicilian Sveshnikov",
	"Sicilian Scheveningen", "Sicilian Accelerated Dragon", "French Defence",
	"Caro-Kann Defence", "Pirc Defence", "Modern Defence", "Alekhine's Defence",
	"Scandinavian Defence", "Queen's Gambit Declined", "Queen's Gambit Accepted",
	"Slav Defence", "Semi-Slav Defence", "Nimzo-Indian Defence",
	"Queen's Indian Defence", "King's Indian Defence", "Grünfeld Defence",
	"Benoni Defence", "Benko Gambit", "Dutch Defence", "Catalan Opening",
	"London System", "Colle System", "Torre Attack", "Trompowsky Attack",
	"English Opening", "Réti Opening", "Bird's Opening", "Larsen's Opening",
	"King's Indian Attack", "Danish Gambit", "Latvian Gambit", "Budapest Gambit",
}

var wineVarieties = []string{
	"Cabernet Sauvignon", "Merlot", "Pinot Noir", "Syrah", "Shiraz", "Malbec",
	"Grenache", "Sangiovese", "Nebbiolo", "Barbera", "Tempranillo", "Garnacha",
	"Zinfandel", "Primitivo", "Carménère", "Petit Verdot", "Cabernet Franc",
	"Gamay", "Mourvèdre", "Touriga Nacional", "Aglianico", "Montepulciano",
	"Chardonnay", "Sauvignon Blanc", "Riesling", "Pinot Grigio", "Pinot Gris",
	"Gewürztraminer", "Chenin Blanc", "Viognier", "Sémillon", "Albariño",
	"Verdejo", "Grüner Veltliner", "Vermentino", "Trebbiano", "Muscat",
	"Furmint", "Assyrtiko", "Torrontés", "Marsanne", "Roussanne", "Silvaner",
}

var cheeses = []string{
	"Cheddar", "Gouda", "Edam", "Emmental", "Gruyère", "Comté", "Beaufort",
	"Parmigiano-Reggiano", "Grana Padano", "Pecorino Romano", "Manchego",
	"Mozzarella", "Burrata", "Ricotta", "Mascarpone", "Provolone", "Taleggio",
	"Gorgonzola", "Roquefort", "Stilton", "Danish Blue", "Cambozola",
	"Brie", "Camembert", "Chaource", "Reblochon", "Munster", "Époisses",
	"Chèvre", "Feta", "Halloumi", "Paneer", "Queso Fresco", "Cotija",
	"Monterey Jack", "Colby", "Havarti", "Jarlsberg", "Raclette", "Fontina",
	"Asiago", "Caciocavallo", "Scamorza", "Tilsit", "Cheshire", "Wensleydale",
	"Red Leicester", "Double Gloucester", "Caerphilly", "Suluguni", "Brynza",
}
