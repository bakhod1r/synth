package locale

import "testing"

// A name bank for a locale that was never registered is skipped rather than
// creating one: the registry is the list of locales Synth supports, and a bank
// added ahead of its locale must not quietly extend it.
func TestApplyNameBanksSkipsUnregisteredLocales(t *testing.T) {
	before := len(registry)
	applyNameBanks(map[string]nameBank{
		"zz_ZZ": {maleFirst: []string{"a"}, femaleFirst: []string{"b"}, last: []string{"c"}},
	})
	if len(registry) != before {
		t.Fatalf("registry grew from %d to %d; an unregistered bank must not add a locale", before, len(registry))
	}
	if Has("zz_ZZ") {
		t.Error("zz_ZZ became a registered locale")
	}
}

// The genderless surname bank fills both gendered lists, so a locale that does
// not inflect surnames still answers a gendered request.
func TestApplyNameBanksFillsBothGendersFromASharedSurnameList(t *testing.T) {
	l := &Locale{}
	registry["zz_TEST"] = l
	t.Cleanup(func() { delete(registry, "zz_TEST") })

	applyNameBanks(map[string]nameBank{
		"zz_TEST": {maleFirst: []string{"m"}, femaleFirst: []string{"f"}, last: []string{"s"}},
	})
	if len(l.MaleLast) != 1 || len(l.FemaleLast) != 1 || len(l.LastNames) != 1 {
		t.Fatalf("surname lists = %v/%v/%v, want the shared list on all three", l.MaleLast, l.FemaleLast, l.LastNames)
	}
	if len(l.FirstNames) != 2 {
		t.Errorf("FirstNames = %v, want both gendered pools merged", l.FirstNames)
	}
}
