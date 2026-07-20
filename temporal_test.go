package synth_test

import (
	"testing"
	"time"

	"github.com/bakhodir/synth"
)

func TestTemporalCausality(t *testing.T) {
	type Order struct {
		ID          int
		CreatedAt   time.Time
		PaidAt      time.Time `synth:"time,after=CreatedAt,gap=1h..48h"`
		ShippedAt   time.Time `synth:"time,after=PaidAt,gap=1h..72h"`
		DeliveredAt time.Time `synth:"time,after=ShippedAt,gap=1h..120h"`
	}
	for _, o := range synth.Make[Order](2000, synth.WithSeed(1)) {
		if !(o.CreatedAt.Before(o.PaidAt) &&
			o.PaidAt.Before(o.ShippedAt) &&
			o.ShippedAt.Before(o.DeliveredAt)) {
			t.Fatalf("lifecycle out of order: %v %v %v %v",
				o.CreatedAt, o.PaidAt, o.ShippedAt, o.DeliveredAt)
		}
		// gap between created and paid must be within 1h..48h.
		gap := o.PaidAt.Sub(o.CreatedAt)
		if gap < time.Hour || gap > 48*time.Hour {
			t.Fatalf("paid gap %v outside 1h..48h", gap)
		}
	}
}
