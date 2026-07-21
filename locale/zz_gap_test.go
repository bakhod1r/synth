package locale_test

import (
	"fmt"
	"testing"

	"github.com/bakhodir/synth/locale"
)

func TestGapReport(t *testing.T) {
	for _, code := range locale.Names() {
		l := locale.Get(code)
		m := len(l.FirstNamesFor("male")) * len(l.LastNamesFor("male"))
		f := len(l.FirstNamesFor("female")) * len(l.LastNamesFor("female"))
		if m < 1000 || f < 1000 {
			fmt.Printf("%s m=%d f=%d\n", code, m, f)
		}
	}
}
