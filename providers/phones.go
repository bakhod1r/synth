package providers

import (
	"sort"
	"strings"
	"sync"

	"github.com/bakhod1r/phonex"
	"github.com/bakhod1r/synth/schema"
)

// Phone numbers valid under their region's numbering plan, from the phonex
// metadata — the same ranges Google's libphonenumber uses.
//
// KindPhone above is the older, weaker type: a dialling code, the place's
// operator digits and seven random digits. It has the shape of a phone number
// and no numbering plan accepts it. These types pass IsValid.
//
//	Houston   +17134359505    (713) 435-9505    fixed_line_or_mobile
//	Andijon   +998944359505   94 435 95 05      mobile
//	London    +442043595056   020 4359 5056     fixed_line
//	Roma      +390618563453   06 1856 3453      fixed_line
//
// A number valid under a numbering plan may well be somebody's. These are for
// fixtures, tests and demos — never dial or message them.

func init() {
	registry[schema.KindPhoneE164] = func(c Ctx) any {
		return formatPhone(c, (*phonex.Phone).E164, echoPhone)
	}
	registry[schema.KindPhoneNational] = func(c Ctx) any {
		return formatPhone(c, (*phonex.Phone).National, echoPhone)
	}
	registry[schema.KindPhoneInternational] = func(c Ctx) any {
		return formatPhone(c, (*phonex.Phone).International, echoPhone)
	}
	registry[schema.KindPhoneType] = func(c Ctx) any {
		// A number that will not parse has no knowable type, and echoing the
		// digits into a type column would be worse than saying so.
		return formatPhone(c, lineType, func(string) string { return "unknown" })
	}
}

// phoneAttempts bounds the search for digits that land inside a range. The
// leading digits are kept, so a handful of attempts is normally enough; the
// bound only matters for ranges that are sparse within their prefix.
const phoneAttempts = 50

// anyPhoneType asks buildPhone for a number in any range at all. Unknown serves
// as the sentinel because a valid number never reports it — it is what phonex
// says when no range matches.
const anyPhoneType = phonex.Unknown

// echoPhone hands back a from= value that could not be parsed. It is the user's
// own field, not one of ours: reformatting is impossible and inventing a second
// number would put two different numbers in one record.
func echoPhone(s string) string { return s }

// formatPhone produces the number this field describes and renders it. It
// prefers the number named by from=, so the formatted fields of a record agree;
// with no from= it draws its own. unparsed renders a from= value phonex rejects.
func formatPhone(c Ctx, render func(*phonex.Phone) string, unparsed func(string) string) any {
	if s, _ := c.Sibling("__from__").(string); s != "" {
		p, err := phonex.Parse(s)
		if err != nil {
			return unparsed(s)
		}
		return render(p)
	}
	p, ok := generatePhone(c)
	if !ok {
		// The metadata does not cover this locale. The built-in phone is not a
		// valid number, which is exactly why these types exist — but a missing
		// field would be worse.
		return phone(c)
	}
	return render(p)
}

// generatePhone returns a valid number for the record's locale, drawn from the
// record's own stream so the same seed yields the same number.
//
// It tries the record's own place first, so the number is issued where the
// record lives. Where the plan will not take that code, the region's example
// number supplies the leading digits instead.
//
// phonex.Generate does the second half of this and cannot be used directly: it
// draws from the global math/rand, and a Synth record must be reproducible from
// its seed.
func generatePhone(c Ctx) (*phonex.Phone, bool) {
	region, ok := phoneRegion(c)
	if !ok {
		return nil, false
	}
	example, ok := phonex.ExampleNumberForType(region, phonex.Mobile)
	if !ok {
		// Not every region separates mobile from fixed line.
		if example, ok = phonex.ExampleNumber(region); !ok {
			return nil, false
		}
	}
	meta := example.Metadata()
	if meta == nil {
		return example, true
	}
	nsn := example.NSN()

	if p, ok := localPhone(c, region, meta, len(nsn)); ok {
		return p, true
	}

	// Failing that, keep the leading digits of the example, which carry an
	// operator or area code known to be live, and randomise the subscriber
	// part. Randomising more than that would mostly leave the range.
	vary := len(nsn) / 2
	if vary > 6 {
		vary = 6
	}
	if vary < 2 {
		// Too short to randomise without leaving the range.
		return example, true
	}
	// The candidate must stay in the same range as the example. Comparing
	// against the example's own type rather than against Mobile matters where a
	// plan does not separate the two: North American numbers report
	// FixedLineOrMobile, and demanding Mobile there would reject every candidate
	// and leave the example — the same number in every record.
	if p, ok := buildPhone(c, region, meta.DialCode, nsn[:len(nsn)-vary], vary, example.Type()); ok {
		return p, true
	}

	// Nothing better was found, so hand back the known-good example.
	return example, true
}

// phoneShape is what a place's area code has to be padded to before the plan
// will accept it: the digits to lead with, and the national number's length.
type phoneShape struct {
	prefix string
	length int
}

// unusableShape marks a place code no shape made work, so the search runs once
// rather than once per record.
var unusableShape = phoneShape{}

// phoneShapes caches the shape found per region and place code. Without it, a
// locale whose codes the plan rejects would re-run the whole search for every
// record.
var phoneShapes sync.Map // region + "|" + place → phoneShape

// localPhone builds a number on the record's own place code — "713" for
// Houston, "94" for Andijon — so the number is issued where the record lives.
//
// Neither half of the shape is something an address knows. The length varies
// within a country: London's 20 takes eight more digits where most UK codes
// take seven. And some plans count the trunk digit as part of the national
// number, so Rome is 06 to libphonenumber and 6 to the atlas the locale's
// places came from. Both are searched.
//
// Lengths are tried nearest the region's example number first. A plan's longest
// permitted length is not its ordinary one — Germany allows fifteen digits and
// issues eleven — and a technically valid Berlin number with seventeen digits in
// it is not the number anyone is testing against.
func localPhone(c Ctx, region string, meta *phonex.Metadata, typical int) (*phonex.Phone, bool) {
	if c.Place == nil || c.Place.PhonePrefix == "" {
		return nil, false
	}
	place := c.Place.PhonePrefix
	key := region + "|" + place

	attempt := func(s phoneShape) (*phonex.Phone, bool) {
		if s.length <= len(s.prefix) {
			return nil, false
		}
		return buildPhone(c, region, meta.DialCode, s.prefix, s.length-len(s.prefix), anyPhoneType)
	}

	if v, ok := phoneShapes.Load(key); ok {
		s := v.(phoneShape)
		if s == unusableShape {
			return nil, false
		}
		// A shape that worked before but not for this record's digits is not
		// evidence that the code is unusable, so the cache stands.
		return attempt(s)
	}

	// The trunk-digit variant. Where a plan counts the trunk digit as part of
	// the national number — Italy is the clearest case, where Rome really is 06
	// and not 6 — the place code as an atlas gives it is one digit short.
	prefixes := []string{place}
	trunk := meta.NationalPrefix
	if trunk == "" {
		trunk = "0"
	}
	if !strings.HasPrefix(place, trunk) {
		prefixes = append(prefixes, trunk+place)
	}

	lengths := make([]int, 0, len(meta.General.Lengths))
	for _, l := range meta.General.Lengths {
		lengths = append(lengths, int(l))
	}
	sort.SliceStable(lengths, func(i, j int) bool {
		di, dj := absInt(lengths[i]-typical), absInt(lengths[j]-typical)
		if di != dj {
			return di < dj
		}
		return lengths[i] > lengths[j]
	})

	for _, length := range lengths {
		for _, prefix := range prefixes {
			s := phoneShape{prefix: prefix, length: length}
			if p, ok := attempt(s); ok {
				phoneShapes.Store(key, s)
				return p, true
			}
		}
	}
	phoneShapes.Store(key, unusableShape)
	return nil, false
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// buildPhone appends n random digits to prefix until the result is a number the
// region accepts. want restricts it to one range; anyPhoneType accepts any valid
// number, which is what the place-code attempt wants — a city code may
// legitimately land on a fixed line.
func buildPhone(c Ctx, region, dialCode, prefix string, n int, want phonex.PhoneType) (*phonex.Phone, bool) {
	var b strings.Builder
	var candidate phonex.Phone
	for i := 0; i < phoneAttempts; i++ {
		b.Reset()
		b.WriteByte('+')
		b.WriteString(dialCode)
		b.WriteString(prefix)
		b.WriteString(c.Rand.Digits(n))
		if candidate.Parse(b.String()) != nil {
			continue
		}
		if !candidate.IsValidForRegion(region) {
			continue
		}
		if want != anyPhoneType && candidate.Type() != want {
			continue
		}
		return candidate.Clone(), true
	}
	return nil, false
}

// phoneRegion names the ISO 3166-1 alpha-2 region the metadata is keyed by.
//
// The locale's own name answers it exactly — en_CA is Canada — and a dialling
// code does not: +1 is nineteen countries and +7 is two, so asking the code
// would validate Toronto's 416 against the United States and Astana's 7172
// against Russia, and reject both. The dialling code is the fallback for
// locales whose name carries no region.
func phoneRegion(c Ctx) (string, bool) {
	if c.Locale == nil {
		return "", false
	}
	if _, iso, ok := strings.Cut(c.Locale.Name, "_"); ok {
		if _, found := phonex.Country(iso); found {
			return strings.ToUpper(iso), true
		}
	}
	code := strings.TrimPrefix(c.Locale.CountryCode, "+")
	if code == "" {
		return "", false
	}
	meta, ok := phonex.CountryByDialCode(code)
	if !ok {
		return "", false
	}
	return meta.ISO2, true
}

// lineType names the range a number falls in. It reports "fixed_line_or_mobile"
// rather than picking one where the numbering plan does not separate them —
// North America is the common case — because the plan genuinely does not say.
func lineType(p *phonex.Phone) string {
	switch p.Type() {
	case phonex.Mobile:
		return "mobile"
	case phonex.FixedLine:
		return "fixed_line"
	case phonex.FixedLineOrMobile:
		return "fixed_line_or_mobile"
	case phonex.TollFree:
		return "toll_free"
	case phonex.PremiumRate:
		return "premium_rate"
	case phonex.SharedCost:
		return "shared_cost"
	case phonex.VoIP:
		return "voip"
	case phonex.PersonalNumber:
		return "personal_number"
	case phonex.Pager:
		return "pager"
	case phonex.UAN:
		return "uan"
	case phonex.Voicemail:
		return "voicemail"
	default:
		return "unknown"
	}
}
