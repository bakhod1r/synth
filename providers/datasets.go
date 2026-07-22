package providers

import "github.com/bakhodir/synth/schema"

// Real-world reference datasets: actual book titles, films, famous people,
// bands and brands. These make generated data recognizable ("realniy dummy")
// instead of lorem noise — useful for demos, screenshots and UI testing.
// Lists are intentionally large so values rarely repeat across a dataset.

var (
	books = []string{
		"To Kill a Mockingbird", "1984", "Pride and Prejudice", "The Great Gatsby",
		"Moby-Dick", "War and Peace", "Crime and Punishment", "The Catcher in the Rye",
		"Brave New World", "The Hobbit", "Fahrenheit 451", "Jane Eyre",
		"The Odyssey", "Don Quixote", "One Hundred Years of Solitude", "Ulysses",
		"Anna Karenina", "The Brothers Karamazov", "Lolita", "Catch-22",
		"The Lord of the Rings", "Wuthering Heights", "Great Expectations", "The Grapes of Wrath",
		"Frankenstein", "Dracula", "The Picture of Dorian Gray", "Les Misérables",
		"Slaughterhouse-Five", "The Sun Also Rises", "Heart of Darkness", "A Tale of Two Cities",
		"The Count of Monte Cristo", "The Adventures of Huckleberry Finn", "Little Women", "Gone with the Wind",
		"The Name of the Rose", "Beloved", "Midnight's Children", "The Kite Runner",
		"Life of Pi", "The Road", "Never Let Me Go", "Cloud Atlas",
		"Norwegian Wood", "The Alchemist", "Dune", "Foundation",
	}
	movies = []string{
		"The Shawshank Redemption", "The Godfather", "Inception", "Pulp Fiction",
		"The Dark Knight", "Forrest Gump", "Interstellar", "Fight Club",
		"The Matrix", "Goodfellas", "Parasite", "Whiplash",
		"Gladiator", "Titanic", "Dune", "Oppenheimer",
		"Schindler's List", "The Lord of the Rings", "Se7en", "The Silence of the Lambs",
		"Saving Private Ryan", "The Green Mile", "Léon: The Professional", "The Prestige",
		"Casablanca", "Spirited Away", "Django Unchained", "The Departed",
		"Joker", "1917", "La La Land", "Mad Max: Fury Road",
		"No Country for Old Men", "There Will Be Blood", "The Grand Budapest Hotel", "Blade Runner 2049",
		"Everything Everywhere All at Once", "Get Out", "Arrival", "Her",
		"The Social Network", "Toy Story", "Coco", "Up",
		"Amélie", "City of God", "Oldboy", "Memento",
	}
	celebrities = []string{
		"Albert Einstein", "Leonardo da Vinci", "Marie Curie", "Nelson Mandela",
		"Stephen Hawking", "Ada Lovelace", "Nikola Tesla", "Frida Kahlo",
		"Steve Jobs", "Serena Williams", "Lionel Messi", "Taylor Swift",
		"Keanu Reeves", "Malala Yousafzai", "Elon Musk", "Oprah Winfrey",
		"Cristiano Ronaldo", "Beyoncé", "Barack Obama", "Greta Thunberg",
		"Denzel Washington", "Meryl Streep", "Morgan Freeman", "Emma Watson",
		"Roger Federer", "Usain Bolt", "Michael Jordan", "Kobe Bryant",
		"Freddie Mercury", "David Bowie", "Bob Dylan", "Aretha Franklin",
		"Alan Turing", "Grace Hopper", "Katherine Johnson", "Carl Sagan",
		"Isaac Newton", "Charles Darwin", "Rosa Parks", "Martin Luther King Jr.",
		"Audrey Hepburn", "Charlie Chaplin", "Pelé", "Muhammad Ali",
		"Scarlett Johansson", "Leonardo DiCaprio", "Zendaya", "Tom Hanks",
	}
	bands = []string{
		"The Beatles", "Queen", "Pink Floyd", "Led Zeppelin", "Nirvana",
		"Radiohead", "Metallica", "Daft Punk", "Coldplay", "The Rolling Stones",
		"AC/DC", "U2", "Red Hot Chili Peppers", "Arctic Monkeys", "The Weeknd",
		"Michael Jackson", "Madonna", "Elton John", "David Guetta", "Eminem",
		"Kendrick Lamar", "Adele", "Ed Sheeran", "Billie Eilish", "Dua Lipa",
		"Linkin Park", "Green Day", "Foo Fighters", "Imagine Dragons", "Gorillaz",
	}
	brands = []string{
		"Apple", "Google", "Nike", "Coca-Cola", "Toyota", "Samsung",
		"Amazon", "Netflix", "Adidas", "Sony", "Tesla", "Spotify",
		"Microsoft", "Meta", "Intel", "IBM", "Oracle", "Nvidia",
		"McDonald's", "Starbucks", "Visa", "Mastercard", "PayPal", "Uber",
		"Airbnb", "Disney", "BMW", "Mercedes-Benz", "Volkswagen", "Ferrari",
		"Louis Vuitton", "Gucci", "Rolex", "Lego", "Ikea", "Canon",
	}
	countriesList = []string{
		"United States", "Japan", "Germany", "Brazil", "India", "France",
		"Canada", "Australia", "Italy", "Spain", "Mexico", "South Korea",
		"United Kingdom", "China", "Russia", "Turkey", "Egypt", "Nigeria",
		"Argentina", "Indonesia", "Netherlands", "Sweden", "Switzerland", "Norway",
		"Poland", "Portugal", "Greece", "Ireland", "Austria", "Belgium",
		"Uzbekistan", "Kazakhstan", "Vietnam", "Thailand", "Malaysia", "Singapore",
	}
	foods = []string{
		"Pizza", "Sushi", "Burger", "Pasta", "Tacos", "Ramen", "Plov",
		"Samsa", "Dumplings", "Curry", "Falafel", "Steak", "Pancakes", "Lagman",
		"Paella", "Lasagna", "Pho", "Kebab", "Biryani", "Gnocchi",
		"Croissant", "Bibimbap", "Manti", "Shawarma", "Risotto", "Gyoza",
		"Enchiladas", "Goulash", "Pad Thai", "Ceviche", "Borscht", "Tiramisu",
		"Baklava", "Hummus", "Poutine", "Katsu", "Empanada", "Dim Sum",
	}
	animals = []string{
		"Lion", "Tiger", "Elephant", "Panda", "Dolphin", "Eagle", "Wolf",
		"Fox", "Penguin", "Koala", "Giraffe", "Octopus", "Owl", "Cheetah",
		"Kangaroo", "Zebra", "Rhinoceros", "Hippopotamus", "Leopard", "Bear",
		"Otter", "Raccoon", "Hedgehog", "Falcon", "Peacock", "Flamingo",
		"Crocodile", "Chameleon", "Sloth", "Lynx", "Bison", "Camel",
		"Dolphin", "Whale", "Shark", "Seahorse", "Jaguar", "Gorilla",
	}
	sports = []string{
		"Football", "Basketball", "Tennis", "Cricket", "Boxing", "Swimming",
		"Volleyball", "Wrestling", "Judo", "Cycling", "Golf", "Baseball",
		"Rugby", "Hockey", "Badminton", "Table Tennis", "Archery", "Fencing",
		"Rowing", "Karate", "Taekwondo", "Skiing", "Snowboarding", "Surfing",
		"Climbing", "Marathon", "Handball", "Water Polo", "Gymnastics", "Weightlifting",
	}
	langs = []string{
		"Go", "Rust", "Python", "JavaScript", "TypeScript", "Java", "C++",
		"Kotlin", "Swift", "Ruby", "Elixir", "Zig",
		"C", "C#", "PHP", "Scala", "Haskell", "Clojure",
		"Dart", "Lua", "Perl", "R", "Julia", "OCaml",
		"Erlang", "F#", "Groovy", "Nim", "Crystal", "Objective-C",
	}
	emojis = []string{
		"😀", "🔥", "🚀", "🎉", "💡", "⭐", "❤️", "👍", "🌍", "🍕", "⚽", "🎧",
		"🎮", "📚", "☕", "🌈", "🐱", "🐶", "🎨", "🏆", "💰", "📱", "✈️", "🎂",
		"🍎", "🌟", "⚡", "🎯", "🧠", "💻", "🔒", "🌸", "🍀", "🎸", "📷", "🥳",
	}
)

func init() {
	// Only real, recognizable titles — no synthesized values.
	registry[schema.KindBook] = func(c Ctx) any { return pick(c.Rand, books) }
	registry[schema.KindMovie] = func(c Ctx) any { return pick(c.Rand, movies) }
	registry[schema.KindCelebrity] = func(c Ctx) any { return pick(c.Rand, celebrities) }
	registry[schema.KindBand] = func(c Ctx) any { return pick(c.Rand, bands) }
	registry[schema.KindBrand] = func(c Ctx) any { return pick(c.Rand, brands) }
	registry[schema.KindCountryName] = func(c Ctx) any { return pick(c.Rand, countriesList) }
	registry[schema.KindFood] = func(c Ctx) any { return localized(c, schema.KindFood, foods) }
	registry[schema.KindAnimal] = func(c Ctx) any { return localized(c, schema.KindAnimal, animals) }
	registry[schema.KindSport] = func(c Ctx) any { return pick(c.Rand, sports) }
	registry[schema.KindPlanet] = func(c Ctx) any { return pick(c.Rand, planets) }
	registry[schema.KindUniversity] = func(c Ctx) any { return pick(c.Rand, universities) }
	registry[schema.KindLanguage] = func(c Ctx) any { return pick(c.Rand, langs) }
	registry[schema.KindEmoji] = func(c Ctx) any { return pick(c.Rand, emojis) }
}
