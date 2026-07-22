package providers

import "github.com/bakhod1r/synth/schema"

// Catalog batch 5. Real lists only, for the same reason as batch 4: a fabricated
// programming language or university name is spotted instantly by the people who
// would use the fixture.

func init() {
	set(schema.KindStadium, stadiums)
	set(schema.KindMuseum, museums)
	set(schema.KindMountain, mountains)
	set(schema.KindRiver, rivers)
}

var mountains = []string{
	"Everest", "K2", "Kangchenjunga", "Lhotse", "Makalu", "Cho Oyu", "Dhaulagiri",
	"Manaslu", "Nanga Parbat", "Annapurna", "Gasherbrum I", "Broad Peak",
	"Shishapangma", "Nuptse", "Ama Dablam", "Machapuchare", "Pumori",
	"Muztagh Ata", "Kongur Tagh", "Ismoil Somoni Peak", "Lenin Peak",
	"Khan Tengri", "Jengish Chokusu", "Hazrat Sulton", "Beshtor Peak",
	"Elbrus", "Kazbek", "Dykh-Tau", "Shkhara", "Ararat", "Damavand", "Zard-Kuh",
	"Mont Blanc", "Matterhorn", "Monte Rosa", "Eiger", "Jungfrau", "Mönch",
	"Grossglockner", "Zugspitze", "Marmolada", "Gran Paradiso", "Triglav",
	"Mulhacén", "Aneto", "Teide", "Etna", "Vesuvius", "Olympus", "Musala",
	"Gerlachovský štít", "Rysy", "Moldoveanu", "Galdhøpiggen", "Kebnekaise",
	"Ben Nevis", "Snowdon", "Scafell Pike", "Carrauntoohil",
	"Denali", "Mount Logan", "Pico de Orizaba", "Popocatépetl", "Iztaccíhuatl",
	"Mount Whitney", "Mount Rainier", "Mount Shasta", "Mount Hood", "Mount Elbert",
	"Pikes Peak", "Grand Teton", "Mount Saint Helens", "Half Dome",
	"Aconcagua", "Ojos del Salado", "Monte Pissis", "Huascarán", "Chimborazo",
	"Cotopaxi", "Illimani", "Fitz Roy", "Cerro Torre", "Pico Bolívar",
	"Kilimanjaro", "Mount Kenya", "Mount Stanley", "Ras Dashen", "Mount Cameroon",
	"Toubkal", "Jebel Marra", "Table Mountain", "Drakensberg",
	"Mount Fuji", "Mount Hotaka", "Mount Aso", "Hallasan", "Baekdu Mountain",
	"Mount Kinabalu", "Puncak Jaya", "Mount Bromo", "Mount Rinjani", "Mount Apo",
	"Mount Kosciuszko", "Aoraki / Mount Cook", "Mount Ruapehu", "Mount Taranaki",
	"Vinson Massif", "Mount Erebus",
}

var rivers = []string{
	"Nile", "Amazon", "Yangtze", "Mississippi", "Missouri", "Yenisei", "Ob",
	"Yellow River", "Congo", "Amur", "Lena", "Mekong", "Mackenzie", "Niger",
	"Paraná", "Volga", "Danube", "Rhine", "Elbe", "Oder", "Vistula", "Seine",
	"Loire", "Rhône", "Garonne", "Thames", "Severn", "Shannon", "Ebro", "Tagus",
	"Douro", "Guadalquivir", "Po", "Tiber", "Arno", "Adige", "Meuse", "Scheldt",
	"Dnieper", "Dniester", "Don", "Kama", "Oka", "Ural", "Pechora", "Neva",
	"Vychegda", "Northern Dvina", "Western Dvina", "Neman", "Bug",
	"Amu Darya", "Syr Darya", "Zarafshan", "Chirchiq", "Naryn", "Vakhsh", "Panj",
	"Ili", "Irtysh", "Ishim", "Tobol", "Chu", "Talas", "Murghab", "Tejen",
	"Tigris", "Euphrates", "Jordan", "Orontes", "Kura", "Aras", "Karun",
	"Indus", "Ganges", "Brahmaputra", "Yamuna", "Godavari", "Krishna", "Narmada",
	"Kaveri", "Sutlej", "Chenab", "Jhelum", "Ravi", "Beas", "Kabul",
	"Irrawaddy", "Salween", "Chao Phraya", "Red River", "Pearl River", "Han River",
	"Songhua", "Liao", "Tarim", "Shinano", "Tone", "Ishikari",
	"Ohio", "Colorado", "Columbia", "Rio Grande", "Arkansas", "Snake", "Hudson",
	"Delaware", "Potomac", "Saint Lawrence", "Yukon", "Fraser", "Churchill",
	"Orinoco", "Madeira", "Negro", "Tapajós", "Xingu", "São Francisco",
	"Uruguay", "Paraguay", "Magdalena", "Marañón", "Ucayali", "Pilcomayo",
	"Zambezi", "Orange", "Limpopo", "Senegal", "Volta", "Blue Nile", "White Nile",
	"Okavango", "Juba", "Shabelle", "Murray", "Darling", "Waikato", "Clutha",
}

var universities = []string{
	"Harvard University", "Stanford University", "Massachusetts Institute of Technology",
	"California Institute of Technology", "Princeton University", "Yale University",
	"Columbia University", "University of Chicago", "Cornell University",
	"Johns Hopkins University", "University of Pennsylvania", "Duke University",
	"Northwestern University", "Carnegie Mellon University", "New York University",
	"University of California, Berkeley", "University of California, Los Angeles",
	"University of Michigan", "University of Texas at Austin", "Georgia Institute of Technology",
	"University of Oxford", "University of Cambridge", "Imperial College London",
	"University College London", "London School of Economics", "University of Edinburgh",
	"University of Manchester", "King's College London", "University of Warwick",
	"ETH Zurich", "EPFL", "Technical University of Munich", "Heidelberg University",
	"Humboldt University of Berlin", "LMU Munich", "RWTH Aachen University",
	"Sorbonne University", "École Polytechnique", "Sciences Po", "PSL University",
	"Delft University of Technology", "University of Amsterdam", "Utrecht University",
	"KU Leuven", "Ghent University", "Uppsala University", "Lund University",
	"University of Copenhagen", "University of Oslo", "University of Helsinki",
	"University of Vienna", "Charles University", "University of Warsaw",
	"Lomonosov Moscow State University", "Saint Petersburg State University",
	"University of Toronto", "McGill University", "University of British Columbia",
	"University of Tokyo", "Kyoto University", "Osaka University", "Tohoku University",
	"Tsinghua University", "Peking University", "Fudan University", "Zhejiang University",
	"Shanghai Jiao Tong University", "Seoul National University", "KAIST",
	"National University of Singapore", "Nanyang Technological University",
	"University of Hong Kong", "Chinese University of Hong Kong",
	"Indian Institute of Technology Bombay", "Indian Institute of Science",
	"University of Melbourne", "Australian National University", "University of Sydney",
	"University of Queensland", "Monash University", "University of Auckland",
	"University of São Paulo", "University of Buenos Aires",
	"National Autonomous University of Mexico", "University of Cape Town",
	"Tel Aviv University", "Hebrew University of Jerusalem", "Technion",
	"National University of Uzbekistan", "Tashkent State Technical University",
	"Westminster International University in Tashkent", "Inha University in Tashkent",
	"Al-Farabi Kazakh National University", "Nazarbayev University",
	"Middle East Technical University", "Boğaziçi University", "Bilkent University",
	"Cairo University", "King Abdullah University of Science and Technology",
}

var stadiums = []string{
	"Camp Nou", "Santiago Bernabéu", "Metropolitano", "Mestalla", "San Mamés",
	"Ramón Sánchez-Pizjuán", "Benito Villamarín", "Old Trafford", "Anfield",
	"Emirates Stadium", "Stamford Bridge", "Etihad Stadium", "Tottenham Hotspur Stadium",
	"Villa Park", "Goodison Park", "St James' Park", "Elland Road", "Craven Cottage",
	"Wembley Stadium", "Hampden Park", "Celtic Park", "Ibrox Stadium",
	"Allianz Arena", "Signal Iduna Park", "Veltins-Arena", "Olympiastadion",
	"Deutsche Bank Park", "Mercedes-Benz Arena", "Red Bull Arena", "Volksparkstadion",
	"San Siro", "Stadio Olimpico", "Allianz Stadium", "Diego Armando Maradona",
	"Stadio Artemio Franchi", "Gewiss Stadium", "Parc des Princes", "Stade Vélodrome",
	"Groupama Stadium", "Stade Pierre-Mauroy", "Stade de France", "Estádio da Luz",
	"Estádio José Alvalade", "Estádio do Dragão", "Johan Cruyff Arena",
	"De Kuip", "Philips Stadion", "Türk Telekom Stadium", "Şükrü Saracoğlu",
	"Vodafone Park", "Luzhniki Stadium", "Gazprom Arena", "Maracanã",
	"Estádio do Morumbi", "Allianz Parque", "Arena Corinthians", "Mineirão",
	"Estadio Monumental", "La Bombonera", "Estadio Azteca", "Estadio BBVA",
	"Rose Bowl", "MetLife Stadium", "SoFi Stadium", "AT&T Stadium", "Lambeau Field",
	"Arrowhead Stadium", "Soldier Field", "Mercedes-Benz Stadium", "Levi's Stadium",
	"Yankee Stadium", "Fenway Park", "Wrigley Field", "Dodger Stadium",
	"Madison Square Garden", "Melbourne Cricket Ground", "Eden Park",
	"Nissan Stadium", "Saitama Stadium 2002", "Tokyo Dome", "Seoul World Cup Stadium",
	"Rungrado 1st of May Stadium", "Beijing National Stadium", "Narendra Modi Stadium",
	"Eden Gardens", "Wankhede Stadium", "Lord's", "The Oval",
	"FNB Stadium", "Cairo International Stadium", "Stade Mohammed V",
	"Lusail Stadium", "Khalifa International Stadium", "King Fahd Stadium",
	"Milliy Stadium", "Bunyodkor Stadium", "Pakhtakor Markaziy Stadium",
	"Astana Arena", "Olimpiyskiy National Sports Complex", "Dinamo Stadium",
}

var museums = []string{
	"Louvre", "Musée d'Orsay", "Centre Pompidou", "Musée Rodin", "Musée Picasso",
	"Palace of Versailles", "British Museum", "National Gallery", "Tate Modern",
	"Tate Britain", "Victoria and Albert Museum", "Natural History Museum",
	"Science Museum", "Imperial War Museum", "National Portrait Gallery",
	"Metropolitan Museum of Art", "Museum of Modern Art", "Guggenheim Museum",
	"Whitney Museum of American Art", "American Museum of Natural History",
	"Smithsonian National Air and Space Museum", "National Gallery of Art",
	"Art Institute of Chicago", "Museum of Fine Arts, Boston",
	"J. Paul Getty Museum", "Los Angeles County Museum of Art", "de Young Museum",
	"Prado Museum", "Reina Sofía", "Thyssen-Bornemisza Museum", "Guggenheim Bilbao",
	"Uffizi Gallery", "Galleria dell'Accademia", "Vatican Museums", "Borghese Gallery",
	"Capitoline Museums", "Pinacoteca di Brera", "Peggy Guggenheim Collection",
	"Rijksmuseum", "Van Gogh Museum", "Mauritshuis", "Stedelijk Museum",
	"Kunsthistorisches Museum", "Albertina", "Belvedere", "Leopold Museum",
	"Pergamon Museum", "Neues Museum", "Alte Nationalgalerie", "Gemäldegalerie",
	"Deutsches Museum", "Städel Museum", "Kunsthaus Zürich", "Fondation Beyeler",
	"State Hermitage Museum", "State Russian Museum", "Tretyakov Gallery",
	"Pushkin Museum", "Acropolis Museum", "National Archaeological Museum, Athens",
	"Topkapi Palace Museum", "Hagia Sophia", "Istanbul Archaeology Museums",
	"Egyptian Museum", "Grand Egyptian Museum", "Israel Museum", "Yad Vashem",
	"National Museum of China", "Palace Museum", "Shanghai Museum",
	"Tokyo National Museum", "National Museum of Western Art", "Mori Art Museum",
	"National Museum of Korea", "National Palace Museum", "ArtScience Museum",
	"National Museum of India", "Chhatrapati Shivaji Maharaj Vastu Sangrahalaya",
	"Royal Ontario Museum", "National Gallery of Canada", "Museo Nacional de Antropología",
	"Frida Kahlo Museum", "Museu Nacional de Belas Artes", "Museo Larco",
	"National Gallery of Victoria", "Australian Museum", "Te Papa Tongarewa",
	"State Museum of History of Uzbekistan", "State Museum of Arts of Uzbekistan",
	"Savitsky Museum", "Amir Timur Museum", "Afrasiyab Museum",
}

// planets covers the Solar System and the dwarf planets. It stops there
// deliberately: exoplanet catalogue designations are identifiers, not names,
// and belong in a code type rather than here.
var planets = []string{
	"Mercury", "Venus", "Earth", "Mars", "Jupiter", "Saturn", "Uranus", "Neptune",
	"Pluto", "Ceres", "Eris", "Haumea", "Makemake",
}
