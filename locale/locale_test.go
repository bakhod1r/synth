package locale

import (
	"reflect"
	"sort"
	"testing"
)

// Map order is random, so an unsorted Names() lists the locales differently on
// every run: the picker becomes unusable and anything naming them stops being
// reproducible.
func TestNamesAreSorted(t *testing.T) {
	got := Names()
	want := append([]string(nil), got...)
	sort.Strings(want)
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("Names() is not sorted: %q before %q", got[i], want[i])
		}
	}
	for i := 0; i < 5; i++ {
		if again := Names(); !reflect.DeepEqual(again, got) {
			t.Fatal("Names() returned a different order on a second call")
		}
	}
}
