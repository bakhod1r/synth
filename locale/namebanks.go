package locale

// Gendered name banks.
//
// A locale needs more than a handful of names before generated data stops
// looking repetitive: 32 first names per gender against 32 surnames gives 1024
// distinct full names, which is enough that a 10,000-row table does not read
// as the same dozen people over and over.
//
// Every name here is a real, common name in its language, and the male and
// female lists are kept apart so a male first name gets a male surname form.
// That matters in Slavic and Baltic languages, where a surname inflects for
// gender: Novák/Nováková, Ivanov/Ivanova, Bērziņš/Bērziņa. Getting it wrong is
// immediately obvious to a native speaker, which is exactly the kind of detail
// that makes generated data look fake.
//
// Ordering note: this file's init runs after locale.go and locales_ext.go
// because Go initializes a package's files in filename order, and "namebanks"
// sorts after both. The banks below deliberately replace whatever those files
// registered, so that order is required, not incidental.
type nameBank struct {
	maleFirst, femaleFirst []string
	// last applies to both genders. For languages that inflect surnames by
	// gender, maleLast and femaleLast are set instead.
	last                 []string
	maleLast, femaleLast []string
}

// nameBanks is every bank, assembled from the per-region files. Splitting the
// data by region keeps each file reviewable by someone who reads those
// languages; splitting it by anything else would not.
var nameBanks = func() map[string]nameBank {
	all := map[string]nameBank{}
	for _, group := range []map[string]nameBank{banksWest, banksSlavic, banksAsia, banksEuro} {
		for code, b := range group {
			all[code] = b
		}
	}
	return all
}()

func init() { applyNameBanks(nameBanks) }

// applyNameBanks copies each name bank onto the locale it belongs to. It is
// separate from init so the skip — a bank whose locale is not registered, which
// is how a bank added ahead of its locale fails — can be exercised; init itself
// runs once, before any test can set up the case.
func applyNameBanks(banks map[string]nameBank) {
	for code, b := range banks {
		l, ok := registry[code]
		if !ok {
			continue
		}
		l.MaleFirst = b.maleFirst
		l.FemaleFirst = b.femaleFirst
		if len(b.maleLast) > 0 {
			l.MaleLast = b.maleLast
			l.FemaleLast = b.femaleLast
			l.LastNames = b.maleLast
		} else {
			l.MaleLast = b.last
			l.FemaleLast = b.last
			l.LastNames = b.last
		}
		// FirstNames is the ungendered pool used where gender is unknown.
		l.FirstNames = append(append([]string{}, b.maleFirst...), b.femaleFirst...)
	}
}
