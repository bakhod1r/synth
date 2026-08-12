package providers

import (
	"sync"

	"github.com/bakhod1r/devicex"
	"github.com/bakhod1r/synth/schema"
)

// Android device fields, answered from the devicex catalogue rather than from a
// list kept here. A model code is evidence a User-Agent actually carries, and
// the handset it names is not derivable from it: Samsung assigned SM-G973F to
// the Galaxy S10 and no pattern recovers that. Hence the catalogue.
//
// device_code is the field that draws. device_brand and device_name read it
// through from= and look it up, so the three describe one handset:
//
//	type Session struct {
//	    Code  string `synth:"device_code"`
//	    Brand string `synth:"device_brand,from=Code"`
//	    Model string `synth:"device_name,from=Code"`
//	}
//	// SM-G973F  Samsung  Galaxy S10
//
// Declared without from=, a brand or name field draws its own device, and the
// row then names one handset by its code and another by its name.

func init() {
	registry[schema.KindDeviceCode] = func(c Ctx) any { return drawDevice(c).Code }
	registry[schema.KindDeviceBrand] = func(c Ctx) any {
		return resolveDevice(c, func(d devicex.Device) string { return d.Brand })
	}
	registry[schema.KindDeviceName] = func(c Ctx) any {
		return resolveDevice(c, func(d devicex.Device) string { return d.Name })
	}
}

// catalogue is the catalogue flattened once into a slice, because generating a
// record needs to pick by index and devicex offers only iteration. The copy is
// a little over a megabyte and is built on first use, so a program that never
// generates a device field never pays for it.
var catalogue = sync.OnceValue(func() []devicex.Device {
	all := make([]devicex.Device, 0, devicex.Len())
	devicex.All(func(d devicex.Device) bool {
		all = append(all, d)
		return true
	})
	return all
})

// drawDevice picks one device using the record's own stream, so the same seed
// yields the same handset.
func drawDevice(c Ctx) devicex.Device {
	all := catalogue()
	if len(all) == 0 {
		return devicex.Device{}
	}
	return all[c.Rand.Intn(len(all))]
}

// resolveDevice answers for the device named by the field's from=, falling back
// to a fresh draw when the field has no from= or the code is not in the
// catalogue.
//
// A code the catalogue does not hold is not guessed at. devicex reports false
// rather than approximating, because a wrong device name is worse than none: it
// is a fact-shaped value that a caller will store, aggregate and report.
func resolveDevice(c Ctx, field func(devicex.Device) string) any {
	code, _ := c.Sibling("__from__").(string)
	if code == "" {
		return field(drawDevice(c))
	}
	d, ok := devicex.Lookup(code)
	if !ok {
		return field(drawDevice(c))
	}
	return field(d)
}
