package providers

import "github.com/bakhodir/synth/internal/rng"

// Combinatorial generation: instead of hand-listing 10k+ literal strings, we
// compose values from real word banks using templates. The reachable space is
// far larger than 10,000 distinct values, so repetition stays very low across
// big datasets while every value still reads as a plausible title/name/brand.

var (
	adjectives = []string{
		"Silent", "Golden", "Broken", "Hidden", "Ancient", "Crimson", "Eternal", "Frozen",
		"Wandering", "Burning", "Silver", "Shattered", "Distant", "Endless", "Forgotten", "Rising",
		"Midnight", "Velvet", "Iron", "Sacred", "Wild", "Lonely", "Electric", "Crystal",
		"Savage", "Gentle", "Restless", "Radiant", "Hollow", "Bright", "Dark", "Quiet",
		"Fearless", "Secret", "Emerald", "Scarlet", "Northern", "Southern", "Lost", "Final",
	}
	nouns = []string{
		"Kingdom", "River", "Shadow", "Empire", "Garden", "Storm", "Mountain", "Ocean",
		"Dream", "Mirror", "Crown", "Winter", "Summer", "Forest", "Desert", "City",
		"Wolf", "Dragon", "Phoenix", "Serpent", "Star", "Moon", "Sun", "Comet",
		"Tower", "Bridge", "Harbor", "Valley", "Island", "Temple", "Throne", "Legacy",
		"Prophecy", "Whisper", "Promise", "Journey", "Secret", "Machine", "Engine", "Circuit",
		"Requiem", "Symphony", "Ballad", "Legend", "Chronicle", "Saga", "Odyssey", "Voyage",
	}
	titleWords = []string{
		"Light", "Night", "Fire", "Water", "Blood", "Time", "Glass", "Stone",
		"Gold", "Silver", "Ash", "Dust", "Rain", "Snow", "Wind", "Flame",
	}
	nameParticles = []string{
		"Corp", "Labs", "Systems", "Technologies", "Industries", "Group", "Holdings",
		"Solutions", "Digital", "Global", "Ventures", "Networks", "Dynamics", "Works",
	}
	techPrefixes = []string{
		"Nova", "Quantum", "Hyper", "Cyber", "Meta", "Neo", "Apex", "Vertex",
		"Zenith", "Nexus", "Flux", "Pulse", "Echo", "Orbit", "Prism", "Vector",
		"Cloud", "Data", "Bit", "Byte", "Code", "Logic", "Core", "Grid",
	}
)

// combineTitle builds a title-like string with a large reachable space.
// Templates × word banks give well over 10k distinct outputs.
func combineTitle(r *rng.Rand) string {
	switch r.Intn(4) {
	case 0: // "Adjective Noun"       ~ 40*48 = 1920
		return pick(r, adjectives) + " " + pick(r, nouns)
	case 1: // "The Noun of Noun"     ~ 48*48 = 2304
		return "The " + pick(r, nouns) + " of " + pick(r, nouns)
	case 2: // "Noun and Noun"        ~ 48*48 = 2304
		return pick(r, nouns) + " and " + pick(r, titleWords)
	default: // "Adjective Word Noun" ~ 40*16*48 = 30720
		return pick(r, adjectives) + " " + pick(r, titleWords) + " " + pick(r, nouns)
	}
}

// combineCompany builds a company name: prefix + particle, ~24*14 plus the
// brand-style single words → thousands of distinct values.
func combineCompany(r *rng.Rand) string {
	return pick(r, techPrefixes) + " " + pick(r, nameParticles)
}
