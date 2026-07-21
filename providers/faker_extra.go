package providers

import (
	"fmt"
	"strings"
	"time"

	"github.com/bakhodir/synth/schema"
)

// Additional atomic types matching common go-faker tags, so Synth is a drop-in
// superset. All values are real/valid: routable-looking IPs, checksum-free but
// well-formed SSNs, real timezone names, etc.

var (
	bloodTypes = []string{"A+", "A-", "B+", "B-", "AB+", "AB-", "O+", "O-"}
	titles     = []string{"Mr", "Mrs", "Ms", "Dr", "Prof", "Miss"}
	months     = []string{"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"}
	weekdays  = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	timezones = []string{"UTC", "America/New_York", "Europe/London", "Europe/Berlin",
		"Asia/Tashkent", "Asia/Tokyo", "Asia/Shanghai", "Australia/Sydney", "America/Los_Angeles"}
	userAgents = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Safari/605.1.15",
		"Mozilla/5.0 (X11; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148 Safari/604.1",
	}
	loremWords = strings.Fields("lorem ipsum dolor sit amet consectetur adipiscing elit sed do " +
		"eiusmod tempor incididunt ut labore et dolore magna aliqua enim ad minim veniam quis nostrud")
)

func init() {
	registry[schema.KindWord] = func(c Ctx) any { return pick(c.Rand, loremWords) }
	registry[schema.KindSentence] = func(c Ctx) any { return sentence(c) }
	registry[schema.KindParagraph] = func(c Ctx) any {
		n := c.Rand.IntRange(3, 6)
		out := make([]string, n)
		for i := range out {
			out[i] = sentence(c)
		}
		return strings.Join(out, " ")
	}
	registry[schema.KindIPv6] = func(c Ctx) any {
		p := make([]string, 8)
		for i := range p {
			p[i] = fmt.Sprintf("%x", c.Rand.Intn(65536))
		}
		return strings.Join(p, ":")
	}
	registry[schema.KindDomain] = func(c Ctx) any {
		tlds := []string{"com", "org", "net", "io", "dev"}
		return strings.ToLower(strings.ReplaceAll(pick(c.Rand, c.Locale.Companies), " ", "")) + "." + pick(c.Rand, tlds)
	}
	registry[schema.KindLatitude] = func(c Ctx) any { return -90 + c.Rand.Float64()*180 }
	registry[schema.KindLongitude] = func(c Ctx) any { return -180 + c.Rand.Float64()*360 }
	registry[schema.KindUnixTime] = func(c Ctx) any { return int(timeProvider(c).(time.Time).Unix()) }
	registry[schema.KindMonth] = func(c Ctx) any { return localized(c, schema.KindMonth, months) }
	registry[schema.KindWeekday] = func(c Ctx) any { return localized(c, schema.KindWeekday, weekdays) }
	registry[schema.KindYear] = func(c Ctx) any { return c.Rand.IntRange(1970, 2030) }
	registry[schema.KindBloodType] = func(c Ctx) any { return pick(c.Rand, bloodTypes) }
	registry[schema.KindUserAgent] = func(c Ctx) any { return pick(c.Rand, userAgents) }
	registry[schema.KindTitle] = func(c Ctx) any { return pick(c.Rand, titles) }
	registry[schema.KindImageURL] = func(c Ctx) any {
		w, h := (c.Rand.Intn(8)+1)*100, (c.Rand.Intn(8)+1)*100
		return fmt.Sprintf("https://picsum.photos/%d/%d", w, h)
	}
	registry[schema.KindSSN] = func(c Ctx) any {
		return fmt.Sprintf("%03d-%02d-%04d", c.Rand.IntRange(1, 899), c.Rand.IntRange(1, 99), c.Rand.IntRange(1, 9999))
	}
	registry[schema.KindTimezone] = func(c Ctx) any { return pick(c.Rand, timezones) }
}

func sentence(c Ctx) string {
	n := c.Rand.IntRange(5, 12)
	w := make([]string, n)
	for i := range w {
		w[i] = pick(c.Rand, loremWords)
	}
	s := strings.Join(w, " ")
	return strings.ToUpper(s[:1]) + s[1:] + "."
}
