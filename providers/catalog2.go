package providers

import (
	"fmt"

	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/schema"
)

// Second catalog batch: more real/valid types (identity, finance, tech, misc).

func init() {
	set(schema.KindMiddleName, []string{"Lee", "Marie", "Ann", "James", "Rose", "John", "Grace", "Paul", "May", "Ray"})
	set(schema.KindNameSuffix, []string{"Jr.", "Sr.", "II", "III", "IV", "PhD", "MD", "Esq."})
	set(schema.KindMaritalStatus, []string{"Single", "Married", "Divorced", "Widowed", "Separated"})
	set(schema.KindEducation, []string{"High School", "Associate", "Bachelor", "Master", "PhD", "Diploma", "MBA"})
	set(schema.KindBankName, []string{"Chase", "Bank of America", "Wells Fargo", "Citibank", "HSBC", "Barclays", "Deutsche Bank", "Santander", "Ipak Yuli", "Kapitalbank"})
	set(schema.KindAccountType, []string{"Checking", "Savings", "Credit", "Business", "Money Market", "Investment"})
	set(schema.KindPaymentMethod, []string{"Credit Card", "Debit Card", "PayPal", "Bank Transfer", "Cash", "Apple Pay", "Google Pay", "Crypto"})
	set(schema.KindProtocol, []string{"http", "https", "ftp", "ssh", "smtp", "ws", "wss", "tcp", "udp"})
	setLocalized(schema.KindWeather, []string{"Sunny", "Cloudy", "Rainy", "Snowy", "Windy", "Foggy", "Stormy", "Clear"})
	setLocalized(schema.KindSeason, []string{"Spring", "Summer", "Autumn", "Winter"})
	set(schema.KindDirection, []string{"North", "South", "East", "West", "Northeast", "Northwest", "Southeast", "Southwest"})
	set(schema.KindElement, []string{"Hydrogen", "Helium", "Carbon", "Nitrogen", "Oxygen", "Iron", "Gold", "Silver", "Copper", "Silicon", "Uranium", "Neon"})
	set(schema.KindConstellation, []string{"Orion", "Ursa Major", "Cassiopeia", "Andromeda", "Leo", "Scorpius", "Cygnus", "Pegasus", "Draco", "Lyra"})
	set(schema.KindShape, []string{"Circle", "Square", "Triangle", "Rectangle", "Hexagon", "Pentagon", "Octagon", "Ellipse", "Rhombus", "Star"})
	set(schema.KindSocial, []string{"Twitter", "Instagram", "LinkedIn", "Facebook", "TikTok", "YouTube", "Reddit", "Telegram", "Discord", "GitHub"})
	set(schema.KindChessPiece, []string{"Pawn", "Knight", "Bishop", "Rook", "Queen", "King"})
	set(schema.KindUnit, []string{"kg", "g", "m", "cm", "km", "l", "ml", "°C", "kWh", "GB", "MB", "px"})
	set(schema.KindGitBranch, []string{"main", "develop", "feature/login", "release/1.0", "hotfix/bug", "staging", "test"})
	set(schema.KindHTMLTag, []string{"div", "span", "a", "p", "img", "ul", "li", "table", "form", "section", "header", "footer"})

	// Format-based / numeric generators.
	registry[schema.KindSwift] = func(c Ctx) any {
		return upperLetters(c.Rand, 4) + c.Locale.IBANCountry + upperLetters(c.Rand, 2) + upperAlnum(c.Rand, 3)
	}
	registry[schema.KindVIN] = func(c Ctx) any {
		const cs = "ABCDEFGHJKLMNPRSTUVWXYZ0123456789" // no I,O,Q
		b := make([]byte, 17)
		for i := range b {
			b[i] = cs[c.Rand.Pick(len(cs))]
		}
		return string(b)
	}
	registry[schema.KindLicensePlate] = func(c Ctx) any {
		return fmt.Sprintf("%s %s", c.Rand.Digits(2)+upperLetters(c.Rand, 1), c.Rand.Digits(3)+upperLetters(c.Rand, 2))
	}
	registry[schema.KindMD5] = func(c Ctx) any { return hexString(c.Rand, 32) }
	registry[schema.KindSHA256] = func(c Ctx) any { return hexString(c.Rand, 64) }
	registry[schema.KindGitCommit] = func(c Ctx) any { return hexString(c.Rand, 40) }
	registry[schema.KindEIN] = func(c Ctx) any { return fmt.Sprintf("%02d-%07d", c.Rand.IntRange(1, 99), c.Rand.IntRange(1, 9999999)) }
	registry[schema.KindPort] = func(c Ctx) any { return c.Rand.IntRange(1024, 65535) }
	registry[schema.KindPercentage] = func(c Ctx) any { return c.Rand.IntRange(0, 100) }
	registry[schema.KindRating] = func(c Ctx) any { return float64(c.Rand.IntRange(10, 50)) / 10 }
	registry[schema.KindTemperature] = func(c Ctx) any { return c.Rand.IntRange(-30, 45) }
	registry[schema.KindSalary] = func(c Ctx) any { return c.Rand.IntRange(20000, 250000) }
	registry[schema.KindSKU] = func(c Ctx) any {
		return upperLetters(c.Rand, 3) + "-" + c.Rand.Digits(4) + "-" + upperLetters(c.Rand, 2)
	}
	registry[schema.KindBase64] = func(c Ctx) any {
		const cs = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
		n := c.Rand.IntRange(16, 32)
		b := make([]byte, n)
		for i := range b {
			b[i] = cs[c.Rand.Pick(len(cs))]
		}
		return string(b) + "="
	}
	registry[schema.KindJWT] = func(c Ctx) any {
		seg := func(n int) string {
			const cs = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-"
			b := make([]byte, n)
			for i := range b {
				b[i] = cs[c.Rand.Pick(len(cs))]
			}
			return string(b)
		}
		return "eyJ" + seg(16) + "." + seg(32) + "." + seg(43)
	}
	registry[schema.KindFileName] = func(c Ctx) any {
		exts := []string{"pdf", "docx", "png", "csv", "json", "go", "sql", "txt"}
		return fmt.Sprintf("%s.%s", pick(c.Rand, loremWords), pick(c.Rand, exts))
	}
	registry[schema.KindFilePath] = func(c Ctx) any {
		return fmt.Sprintf("/%s/%s/%s.txt", pick(c.Rand, loremWords), pick(c.Rand, loremWords), pick(c.Rand, loremWords))
	}
}

func hexString(r *rng.Rand, n int) string {
	const hx = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hx[r.Pick(16)]
	}
	return string(b)
}

func upperLetters(r *rng.Rand, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte('A' + r.Intn(26))
	}
	return string(b)
}

func upperAlnum(r *rng.Rand, n int) string {
	const cs = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = cs[r.Pick(len(cs))]
	}
	return string(b)
}
