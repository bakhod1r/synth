package locale

import (
	"sync"
	"testing"
)

// Get used to fill in IPBlocks on first call, writing to a *Locale shared by
// every caller. A worker pool that generated records concurrently tripped the
// race detector on the first record. This test only fails under -race.
func TestGetIsConcurrentSafe(t *testing.T) {
	names := Names()
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, n := range names {
				l := Get(n)
				_ = len(l.IPBlocks)
			}
		}()
	}
	wg.Wait()
}

// Every registered locale has IP blocks before any Get call, so ipv4 generation
// never depends on who called Get first.
func TestEveryLocaleHasIPBlocks(t *testing.T) {
	for _, n := range Names() {
		if len(registry[n].IPBlocks) == 0 {
			t.Errorf("locale %s has no IPBlocks", n)
		}
	}
}

// A locale with no entry in the block map falls back to the en_US ranges, and
// one that declares its own keeps them. init cannot be re-run, so applyIPBlocks
// is called directly on locales registered for this test.
func TestApplyIPBlocksFallbackAndOwnBlocks(t *testing.T) {
	unknown := &Locale{Name: "zz_UNKNOWN"}
	own := &Locale{Name: "zz_OWN", IPBlocks: []int{7}}
	registry["zz_UNKNOWN"] = unknown
	registry["zz_OWN"] = own
	t.Cleanup(func() {
		delete(registry, "zz_UNKNOWN")
		delete(registry, "zz_OWN")
	})

	applyIPBlocks(countryIPBlocks)

	if got, want := len(unknown.IPBlocks), len(countryIPBlocks["en_US"]); got != want {
		t.Errorf("unknown locale got %d blocks, want the %d en_US blocks", got, want)
	}
	if len(own.IPBlocks) != 1 || own.IPBlocks[0] != 7 {
		t.Errorf("own blocks overwritten: %v", own.IPBlocks)
	}
}
