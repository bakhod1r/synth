package providers

import (
	"fmt"

	"github.com/bakhod1r/synth/schema"
)

// This catalog adds many real-world "pick from a curated set" types, pushing
// Synth's type count toward the breadth of large fakers. Each is registered as
// an ordinary kind so it works via tag or field-name inference.

// set registers a provider that picks uniformly from a fixed list.
func set(k schema.Kind, values []string) {
	registry[k] = func(c Ctx) any { return pick(c.Rand, values) }
}

func init() {
	set(schema.KindVehicleMake, []string{"Toyota", "Ford", "BMW", "Honda", "Mercedes-Benz", "Volkswagen", "Audi", "Tesla", "Hyundai", "Kia", "Nissan", "Chevrolet"})
	set(schema.KindVehicleModel, []string{"Corolla", "Civic", "Model 3", "F-150", "Camry", "Golf", "A4", "X5", "Mustang", "Sonata", "Cybertruck", "Rav4"})
	set(schema.KindVehicleType, []string{"Sedan", "SUV", "Hatchback", "Coupe", "Pickup", "Van", "Convertible", "Wagon"})
	set(schema.KindDepartment, []string{"Engineering", "Sales", "Marketing", "Finance", "Human Resources", "Operations", "Support", "Legal", "Product", "Design"})
	set(schema.KindJobArea, []string{"Backend", "Frontend", "Data", "Infrastructure", "Security", "Growth", "Research", "Quality"})
	set(schema.KindJobLevel, []string{"Intern", "Junior", "Mid", "Senior", "Lead", "Principal", "Staff", "Director", "VP"})
	set(schema.KindProductCategory, []string{"Electronics", "Clothing", "Home", "Books", "Toys", "Beauty", "Sports", "Grocery", "Automotive", "Garden"})
	set(schema.KindProductMaterial, []string{"Cotton", "Steel", "Wood", "Plastic", "Leather", "Glass", "Aluminum", "Ceramic", "Rubber", "Silk"})
	set(schema.KindMusicGenre, []string{"Rock", "Pop", "Jazz", "Hip Hop", "Classical", "Electronic", "Reggae", "Blues", "Metal", "Folk", "R&B", "Country"})
	set(schema.KindInstrument, []string{"Guitar", "Piano", "Violin", "Drums", "Saxophone", "Trumpet", "Cello", "Flute", "Bass", "Clarinet"})
	set(schema.KindSportsTeam, []string{"Real Madrid", "Barcelona", "Manchester United", "Bayern Munich", "Liverpool", "Juventus", "Lakers", "Warriors", "Yankees", "Celtics"})
	set(schema.KindFramework, []string{"React", "Angular", "Vue", "Django", "Rails", "Spring", "Laravel", "Express", "Flask", "Next.js", "Svelte", "Gin"})
	set(schema.KindMimeType, []string{"application/json", "text/html", "image/png", "image/jpeg", "application/pdf", "text/csv", "video/mp4", "application/xml", "text/plain", "audio/mpeg"})
	set(schema.KindHTTPMethod, []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"})
	set(schema.KindOS, []string{"Windows 11", "macOS Sonoma", "Ubuntu 22.04", "Android 14", "iOS 17", "Debian 12", "Fedora 39"})
	set(schema.KindBrowser, []string{"Chrome", "Firefox", "Safari", "Edge", "Opera", "Brave"})
	set(schema.KindDevice, []string{"iPhone 15", "Galaxy S24", "Pixel 8", "MacBook Pro", "iPad Air", "Surface Pro", "ThinkPad X1"})
	// Codes come from the paired airport table in things.go, so a code and a
	// name column can describe the same airport.
	set(schema.KindAirport, airportCodes)
	set(schema.KindAirline, []string{"Emirates", "Qatar Airways", "Lufthansa", "Delta", "Singapore Airlines", "Turkish Airlines", "Uzbekistan Airways", "ANA"})
	set(schema.KindStockTicker, []string{"AAPL", "MSFT", "GOOGL", "AMZN", "TSLA", "NVDA", "META", "NFLX", "AMD", "INTC"})
	set(schema.KindCrypto, []string{"Bitcoin", "Ethereum", "Solana", "Cardano", "Polkadot", "Litecoin", "Chainlink", "Avalanche"})
	set(schema.KindContinent, []string{"Africa", "Antarctica", "Asia", "Europe", "North America", "Oceania", "South America"})
	set(schema.KindLanguageName, []string{"English", "Spanish", "Mandarin", "Hindi", "Arabic", "French", "Russian", "Uzbek", "Japanese", "German", "Portuguese", "Korean"})
	setLocalized(schema.KindFruit, []string{"Apple", "Banana", "Orange", "Grape", "Mango", "Strawberry", "Pineapple", "Watermelon", "Peach", "Cherry", "Pomegranate"})
	setLocalized(schema.KindVegetable, []string{"Carrot", "Potato", "Tomato", "Cucumber", "Onion", "Pepper", "Broccoli", "Spinach", "Eggplant", "Cabbage"})
	setLocalized(schema.KindDrink, []string{"Coffee", "Tea", "Water", "Juice", "Cola", "Beer", "Wine", "Lemonade", "Smoothie", "Ayran"})
	set(schema.KindDogBreed, []string{"Labrador", "German Shepherd", "Bulldog", "Poodle", "Beagle", "Rottweiler", "Husky", "Corgi", "Dachshund", "Chihuahua"})
	set(schema.KindCatBreed, []string{"Persian", "Maine Coon", "Siamese", "Bengal", "Ragdoll", "Sphynx", "British Shorthair", "Scottish Fold"})
	set(schema.KindFlower, []string{"Rose", "Tulip", "Lily", "Orchid", "Sunflower", "Daisy", "Iris", "Lavender", "Peony", "Jasmine"})
	set(schema.KindGemstone, []string{"Diamond", "Ruby", "Emerald", "Sapphire", "Amethyst", "Topaz", "Opal", "Pearl", "Garnet", "Turquoise"})
	set(schema.KindMetal, []string{"Gold", "Silver", "Platinum", "Copper", "Iron", "Titanium", "Bronze", "Nickel", "Zinc", "Palladium"})
	set(schema.KindZodiac, []string{"Aries", "Taurus", "Gemini", "Cancer", "Leo", "Virgo", "Libra", "Scorpio", "Sagittarius", "Capricorn", "Aquarius", "Pisces"})
	set(schema.KindCountryCode, []string{"US", "GB", "DE", "FR", "JP", "CN", "IN", "BR", "RU", "UZ", "TR", "KR"})
	set(schema.KindCurrencyName, []string{"US Dollar", "Euro", "Japanese Yen", "British Pound", "Uzbek Som", "Russian Ruble", "Chinese Yuan", "Indian Rupee"})
	set(schema.KindCurrencySymbol, []string{"$", "€", "¥", "£", "₽", "₩", "₹", "₴", "so'm"})
	set(schema.KindCatchPhrase, []string{"Empowering the future", "Innovate and grow", "Simplify everything", "Data-driven decisions", "Built for scale", "Think different"})
	set(schema.KindAppName, []string{"CloudSync", "TaskFlow", "DataHub", "PayFast", "QuickChat", "SmartDesk", "GreenLedger", "NovaMail"})
	set(schema.KindFileExt, []string{"pdf", "docx", "xlsx", "png", "jpg", "csv", "json", "zip", "mp4", "txt", "go", "sql"})

	// Format-based generators.
	registry[schema.KindSemver] = func(c Ctx) any {
		return fmt.Sprintf("%d.%d.%d", c.Rand.Intn(10), c.Rand.Intn(20), c.Rand.Intn(50))
	}
	registry[schema.KindRGBColor] = func(c Ctx) any {
		return fmt.Sprintf("rgb(%d, %d, %d)", c.Rand.Intn(256), c.Rand.Intn(256), c.Rand.Intn(256))
	}
	registry[schema.KindEAN13] = func(c Ctx) any {
		body := c.Rand.Digits(12)
		return body + string(ean13Check(body))
	}
	registry[schema.KindISBN] = func(c Ctx) any {
		body := "978" + c.Rand.Digits(9)
		return "978-" + body[3:] + string(ean13Check(body))
	}
	registry[schema.KindHTTPStatus] = func(c Ctx) any {
		codes := []int{200, 201, 204, 301, 302, 400, 401, 403, 404, 409, 422, 429, 500, 502, 503}
		return codes[c.Rand.Pick(len(codes))]
	}
	registry[schema.KindSlug] = func(c Ctx) any {
		return fmt.Sprintf("%s-%s-%s", pick(c.Rand, loremWords), pick(c.Rand, loremWords), c.Rand.Digits(4))
	}
}

// ean13Check returns the EAN-13 check digit for a 12-digit body.
func ean13Check(body string) byte {
	sum := 0
	for i := 0; i < len(body); i++ {
		d := int(body[i] - '0')
		if i%2 == 1 {
			d *= 3
		}
		sum += d
	}
	return byte('0' + (10-sum%10)%10)
}
