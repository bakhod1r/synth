package providers

import "github.com/bakhod1r/synth/schema"

// Body parts, plus clearer aliases for the two "language" kinds.
//
// "language" has always meant a programming language here and "languagename" a
// spoken one, which is a distinction nobody should have to remember from the
// name alone. The aliases below say what they mean; the old names keep working.

func init() {
	setLocalized(schema.KindBodyPart, bodyParts)

	// Aliases delegate at call time rather than copying the provider now.
	// Package init runs file by file in filename order, and this file sorts
	// before catalog.go — copying here would capture a nil provider.
	registry[schema.KindProgrammingLanguage] = alias(schema.KindLanguage)
	registry[schema.KindHumanLanguage] = alias(schema.KindLanguageName)
}

// alias returns a provider that forwards to another kind, so the two names
// share one dataset and can never drift apart.
func alias(target schema.Kind) Provider {
	return func(c Ctx) any { return registry[target](c) }
}

// bodyParts is the English fallback. Locale versions live in the locale
// catalog, because this is exactly the kind of vocabulary that looks wrong in
// the wrong language.
var bodyParts = []string{
	"head", "hair", "forehead", "eye", "eyebrow", "eyelash", "ear", "nose",
	"cheek", "mouth", "lip", "tooth", "tongue", "chin", "jaw", "neck",
	"throat", "shoulder", "arm", "elbow", "wrist", "hand", "palm", "finger",
	"thumb", "nail", "chest", "back", "waist", "hip", "stomach", "leg",
	"thigh", "knee", "shin", "calf", "ankle", "foot", "heel", "toe",
	"skin", "bone", "muscle", "heart", "lung", "liver", "kidney", "brain",
}
