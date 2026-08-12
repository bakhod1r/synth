package providers

import "strings"

// ASCII folding for mailbox local parts.
//
// A mailbox local part is ASCII unless the whole path speaks SMTPUTF8
// (RFC 6531), which most systems still do not: RFC 5321 defines the local part
// over ASCII, and a fixture carrying "денис.борисов@example.com" is rejected by
// the first validator it meets. Real addresses in those locales are written in
// Latin anyway — people transliterate their own names when they sign up.
//
// So a name is folded to ASCII before it becomes an address. What cannot be
// folded — Chinese, Japanese kanji, Thai, Arabic, Hebrew, Devanagari, where a
// correct romanisation needs a dictionary rather than a table — falls back to a
// Latin handle, which is what speakers of those languages commonly pick anyway.

// latinFold maps Latin letters with diacritics to their base letter. Generated
// by NFD-decomposing U+00C0–U+024F and U+1E00–U+1EFF and dropping the combining
// marks, then checking the result is ASCII.
var latinFold = map[rune]string{
	'À': "A", 'Á': "A", 'Â': "A", 'Ã': "A",
	'Ä': "A", 'Å': "A", 'Ç': "C", 'È': "E",
	'É': "E", 'Ê': "E", 'Ë': "E", 'Ì': "I",
	'Í': "I", 'Î': "I", 'Ï': "I", 'Ñ': "N",
	'Ò': "O", 'Ó': "O", 'Ô': "O", 'Õ': "O",
	'Ö': "O", 'Ù': "U", 'Ú': "U", 'Û': "U",
	'Ü': "U", 'Ý': "Y", 'à': "a", 'á': "a",
	'â': "a", 'ã': "a", 'ä': "a", 'å': "a",
	'ç': "c", 'è': "e", 'é': "e", 'ê': "e",
	'ë': "e", 'ì': "i", 'í': "i", 'î': "i",
	'ï': "i", 'ñ': "n", 'ò': "o", 'ó': "o",
	'ô': "o", 'õ': "o", 'ö': "o", 'ù': "u",
	'ú': "u", 'û': "u", 'ü': "u", 'ý': "y",
	'ÿ': "y", 'Ā': "A", 'ā': "a", 'Ă': "A",
	'ă': "a", 'Ą': "A", 'ą': "a", 'Ć': "C",
	'ć': "c", 'Ĉ': "C", 'ĉ': "c", 'Ċ': "C",
	'ċ': "c", 'Č': "C", 'č': "c", 'Ď': "D",
	'ď': "d", 'Ē': "E", 'ē': "e", 'Ĕ': "E",
	'ĕ': "e", 'Ė': "E", 'ė': "e", 'Ę': "E",
	'ę': "e", 'Ě': "E", 'ě': "e", 'Ĝ': "G",
	'ĝ': "g", 'Ğ': "G", 'ğ': "g", 'Ġ': "G",
	'ġ': "g", 'Ģ': "G", 'ģ': "g", 'Ĥ': "H",
	'ĥ': "h", 'Ĩ': "I", 'ĩ': "i", 'Ī': "I",
	'ī': "i", 'Ĭ': "I", 'ĭ': "i", 'Į': "I",
	'į': "i", 'İ': "I", 'Ĵ': "J", 'ĵ': "j",
	'Ķ': "K", 'ķ': "k", 'Ĺ': "L", 'ĺ': "l",
	'Ļ': "L", 'ļ': "l", 'Ľ': "L", 'ľ': "l",
	'Ń': "N", 'ń': "n", 'Ņ': "N", 'ņ': "n",
	'Ň': "N", 'ň': "n", 'Ō': "O", 'ō': "o",
	'Ŏ': "O", 'ŏ': "o", 'Ő': "O", 'ő': "o",
	'Ŕ': "R", 'ŕ': "r", 'Ŗ': "R", 'ŗ': "r",
	'Ř': "R", 'ř': "r", 'Ś': "S", 'ś': "s",
	'Ŝ': "S", 'ŝ': "s", 'Ş': "S", 'ş': "s",
	'Š': "S", 'š': "s", 'Ţ': "T", 'ţ': "t",
	'Ť': "T", 'ť': "t", 'Ũ': "U", 'ũ': "u",
	'Ū': "U", 'ū': "u", 'Ŭ': "U", 'ŭ': "u",
	'Ů': "U", 'ů': "u", 'Ű': "U", 'ű': "u",
	'Ų': "U", 'ų': "u", 'Ŵ': "W", 'ŵ': "w",
	'Ŷ': "Y", 'ŷ': "y", 'Ÿ': "Y", 'Ź': "Z",
	'ź': "z", 'Ż': "Z", 'ż': "z", 'Ž': "Z",
	'ž': "z", 'Ơ': "O", 'ơ': "o", 'Ư': "U",
	'ư': "u", 'Ǎ': "A", 'ǎ': "a", 'Ǐ': "I",
	'ǐ': "i", 'Ǒ': "O", 'ǒ': "o", 'Ǔ': "U",
	'ǔ': "u", 'Ǖ': "U", 'ǖ': "u", 'Ǘ': "U",
	'ǘ': "u", 'Ǚ': "U", 'ǚ': "u", 'Ǜ': "U",
	'ǜ': "u", 'Ǟ': "A", 'ǟ': "a", 'Ǡ': "A",
	'ǡ': "a", 'Ǧ': "G", 'ǧ': "g", 'Ǩ': "K",
	'ǩ': "k", 'Ǫ': "O", 'ǫ': "o", 'Ǭ': "O",
	'ǭ': "o", 'ǰ': "j", 'Ǵ': "G", 'ǵ': "g",
	'Ǹ': "N", 'ǹ': "n", 'Ǻ': "A", 'ǻ': "a",
	'Ȁ': "A", 'ȁ': "a", 'Ȃ': "A", 'ȃ': "a",
	'Ȅ': "E", 'ȅ': "e", 'Ȇ': "E", 'ȇ': "e",
	'Ȉ': "I", 'ȉ': "i", 'Ȋ': "I", 'ȋ': "i",
	'Ȍ': "O", 'ȍ': "o", 'Ȏ': "O", 'ȏ': "o",
	'Ȑ': "R", 'ȑ': "r", 'Ȓ': "R", 'ȓ': "r",
	'Ȕ': "U", 'ȕ': "u", 'Ȗ': "U", 'ȗ': "u",
	'Ș': "S", 'ș': "s", 'Ț': "T", 'ț': "t",
	'Ȟ': "H", 'ȟ': "h", 'Ȧ': "A", 'ȧ': "a",
	'Ȩ': "E", 'ȩ': "e", 'Ȫ': "O", 'ȫ': "o",
	'Ȭ': "O", 'ȭ': "o", 'Ȯ': "O", 'ȯ': "o",
	'Ȱ': "O", 'ȱ': "o", 'Ȳ': "Y", 'ȳ': "y",
	'Ḁ': "A", 'ḁ': "a", 'Ḃ': "B", 'ḃ': "b",
	'Ḅ': "B", 'ḅ': "b", 'Ḇ': "B", 'ḇ': "b",
	'Ḉ': "C", 'ḉ': "c", 'Ḋ': "D", 'ḋ': "d",
	'Ḍ': "D", 'ḍ': "d", 'Ḏ': "D", 'ḏ': "d",
	'Ḑ': "D", 'ḑ': "d", 'Ḓ': "D", 'ḓ': "d",
	'Ḕ': "E", 'ḕ': "e", 'Ḗ': "E", 'ḗ': "e",
	'Ḙ': "E", 'ḙ': "e", 'Ḛ': "E", 'ḛ': "e",
	'Ḝ': "E", 'ḝ': "e", 'Ḟ': "F", 'ḟ': "f",
	'Ḡ': "G", 'ḡ': "g", 'Ḣ': "H", 'ḣ': "h",
	'Ḥ': "H", 'ḥ': "h", 'Ḧ': "H", 'ḧ': "h",
	'Ḩ': "H", 'ḩ': "h", 'Ḫ': "H", 'ḫ': "h",
	'Ḭ': "I", 'ḭ': "i", 'Ḯ': "I", 'ḯ': "i",
	'Ḱ': "K", 'ḱ': "k", 'Ḳ': "K", 'ḳ': "k",
	'Ḵ': "K", 'ḵ': "k", 'Ḷ': "L", 'ḷ': "l",
	'Ḹ': "L", 'ḹ': "l", 'Ḻ': "L", 'ḻ': "l",
	'Ḽ': "L", 'ḽ': "l", 'Ḿ': "M", 'ḿ': "m",
	'Ṁ': "M", 'ṁ': "m", 'Ṃ': "M", 'ṃ': "m",
	'Ṅ': "N", 'ṅ': "n", 'Ṇ': "N", 'ṇ': "n",
	'Ṉ': "N", 'ṉ': "n", 'Ṋ': "N", 'ṋ': "n",
	'Ṍ': "O", 'ṍ': "o", 'Ṏ': "O", 'ṏ': "o",
	'Ṑ': "O", 'ṑ': "o", 'Ṓ': "O", 'ṓ': "o",
	'Ṕ': "P", 'ṕ': "p", 'Ṗ': "P", 'ṗ': "p",
	'Ṙ': "R", 'ṙ': "r", 'Ṛ': "R", 'ṛ': "r",
	'Ṝ': "R", 'ṝ': "r", 'Ṟ': "R", 'ṟ': "r",
	'Ṡ': "S", 'ṡ': "s", 'Ṣ': "S", 'ṣ': "s",
	'Ṥ': "S", 'ṥ': "s", 'Ṧ': "S", 'ṧ': "s",
	'Ṩ': "S", 'ṩ': "s", 'Ṫ': "T", 'ṫ': "t",
	'Ṭ': "T", 'ṭ': "t", 'Ṯ': "T", 'ṯ': "t",
	'Ṱ': "T", 'ṱ': "t", 'Ṳ': "U", 'ṳ': "u",
	'Ṵ': "U", 'ṵ': "u", 'Ṷ': "U", 'ṷ': "u",
	'Ṹ': "U", 'ṹ': "u", 'Ṻ': "U", 'ṻ': "u",
	'Ṽ': "V", 'ṽ': "v", 'Ṿ': "V", 'ṿ': "v",
	'Ẁ': "W", 'ẁ': "w", 'Ẃ': "W", 'ẃ': "w",
	'Ẅ': "W", 'ẅ': "w", 'Ẇ': "W", 'ẇ': "w",
	'Ẉ': "W", 'ẉ': "w", 'Ẋ': "X", 'ẋ': "x",
	'Ẍ': "X", 'ẍ': "x", 'Ẏ': "Y", 'ẏ': "y",
	'Ẑ': "Z", 'ẑ': "z", 'Ẓ': "Z", 'ẓ': "z",
	'Ẕ': "Z", 'ẕ': "z", 'ẖ': "h", 'ẗ': "t",
	'ẘ': "w", 'ẙ': "y", 'Ạ': "A", 'ạ': "a",
	'Ả': "A", 'ả': "a", 'Ấ': "A", 'ấ': "a",
	'Ầ': "A", 'ầ': "a", 'Ẩ': "A", 'ẩ': "a",
	'Ẫ': "A", 'ẫ': "a", 'Ậ': "A", 'ậ': "a",
	'Ắ': "A", 'ắ': "a", 'Ằ': "A", 'ằ': "a",
	'Ẳ': "A", 'ẳ': "a", 'Ẵ': "A", 'ẵ': "a",
	'Ặ': "A", 'ặ': "a", 'Ẹ': "E", 'ẹ': "e",
	'Ẻ': "E", 'ẻ': "e", 'Ẽ': "E", 'ẽ': "e",
	'Ế': "E", 'ế': "e", 'Ề': "E", 'ề': "e",
	'Ể': "E", 'ể': "e", 'Ễ': "E", 'ễ': "e",
	'Ệ': "E", 'ệ': "e", 'Ỉ': "I", 'ỉ': "i",
	'Ị': "I", 'ị': "i", 'Ọ': "O", 'ọ': "o",
	'Ỏ': "O", 'ỏ': "o", 'Ố': "O", 'ố': "o",
	'Ồ': "O", 'ồ': "o", 'Ổ': "O", 'ổ': "o",
	'Ỗ': "O", 'ỗ': "o", 'Ộ': "O", 'ộ': "o",
	'Ớ': "O", 'ớ': "o", 'Ờ': "O", 'ờ': "o",
	'Ở': "O", 'ở': "o", 'Ỡ': "O", 'ỡ': "o",
	'Ợ': "O", 'ợ': "o", 'Ụ': "U", 'ụ': "u",
	'Ủ': "U", 'ủ': "u", 'Ứ': "U", 'ứ': "u",
	'Ừ': "U", 'ừ': "u", 'Ử': "U", 'ử': "u",
	'Ữ': "U", 'ữ': "u", 'Ự': "U", 'ự': "u",
	'Ỳ': "Y", 'ỳ': "y", 'Ỵ': "Y", 'ỵ': "y",
	'Ỷ': "Y", 'ỷ': "y", 'Ỹ': "Y", 'ỹ': "y",
}

// latinExtra covers the Latin letters that have no decomposition to strip —
// their diacritic is part of the glyph — plus the ligatures and the letters
// that spell out as two.
var latinExtra = map[rune]string{
	'Æ': "AE", 'æ': "ae", 'Œ': "OE", 'œ': "oe", 'ß': "ss",
	'Ø': "O", 'ø': "o", 'Đ': "D", 'đ': "d", 'Ð': "D", 'ð': "d",
	'Ł': "L", 'ł': "l", 'Þ': "TH", 'þ': "th", 'Ħ': "H", 'ħ': "h",
	'Ŧ': "T", 'ŧ': "t", 'Ɖ': "D", 'Ə': "E", 'ə': "e", 'İ': "I", 'ı': "i",
}

// cyrillicFold is the common transliteration of Cyrillic, covering the Russian,
// Ukrainian, Belarusian, Bulgarian, Serbian, Macedonian and Kazakh letters the
// locales here use.
var cyrillicFold = map[rune]string{
	'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D", 'Е': "E", 'Ё': "Yo",
	'Ж': "Zh", 'З': "Z", 'И': "I", 'Й': "Y", 'К': "K", 'Л': "L", 'М': "M",
	'Н': "N", 'О': "O", 'П': "P", 'Р': "R", 'С': "S", 'Т': "T", 'У': "U",
	'Ф': "F", 'Х': "Kh", 'Ц': "Ts", 'Ч': "Ch", 'Ш': "Sh", 'Щ': "Shch",
	'Ъ': "", 'Ы': "Y", 'Ь': "", 'Э': "E", 'Ю': "Yu", 'Я': "Ya",
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "yo",
	'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "kh", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "shch",
	'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
	// Ukrainian, Belarusian, Serbian, Macedonian.
	'Є': "Ye", 'є': "ye", 'І': "I", 'і': "i", 'Ї': "Yi", 'ї': "yi",
	'Ґ': "G", 'ґ': "g", 'Ў': "U", 'ў': "u", 'Ђ': "Dj", 'ђ': "dj",
	'Ј': "J", 'ј': "j", 'Љ': "Lj", 'љ': "lj", 'Њ': "Nj", 'њ': "nj",
	'Ћ': "C", 'ћ': "c", 'Џ': "Dz", 'џ': "dz", 'Ѕ': "Dz", 'ѕ': "dz",
	'Ќ': "K", 'ќ': "k", 'Ѓ': "G", 'ѓ': "g",
	// Kazakh.
	'Ә': "A", 'ә': "a", 'Ғ': "G", 'ғ': "g", 'Қ': "Q", 'қ': "q",
	'Ң': "Ng", 'ң': "ng", 'Ө': "O", 'ө': "o", 'Ұ': "U", 'ұ': "u",
	'Ү': "U", 'ү': "u", 'Һ': "H", 'һ': "h",
}

// greekFold is the standard transliteration of modern Greek. The accented
// vowels are folded by latinFold's counterpart here rather than separately.
var greekFold = map[rune]string{
	'Α': "A", 'Β': "V", 'Γ': "G", 'Δ': "D", 'Ε': "E", 'Ζ': "Z", 'Η': "I",
	'Θ': "Th", 'Ι': "I", 'Κ': "K", 'Λ': "L", 'Μ': "M", 'Ν': "N", 'Ξ': "X",
	'Ο': "O", 'Π': "P", 'Ρ': "R", 'Σ': "S", 'Τ': "T", 'Υ': "Y", 'Φ': "F",
	'Χ': "Ch", 'Ψ': "Ps", 'Ω': "O",
	'α': "a", 'β': "v", 'γ': "g", 'δ': "d", 'ε': "e", 'ζ': "z", 'η': "i",
	'θ': "th", 'ι': "i", 'κ': "k", 'λ': "l", 'μ': "m", 'ν': "n", 'ξ': "x",
	'ο': "o", 'π': "p", 'ρ': "r", 'σ': "s", 'ς': "s", 'τ': "t", 'υ': "y",
	'φ': "f", 'χ': "ch", 'ψ': "ps", 'ω': "o",
	'ά': "a", 'έ': "e", 'ή': "i", 'ί': "i", 'ό': "o", 'ύ': "y", 'ώ': "o",
	'ϊ': "i", 'ϋ': "y", 'ΐ': "i", 'ΰ': "y",
	'Ά': "A", 'Έ': "E", 'Ή': "I", 'Ί': "I", 'Ό': "O", 'Ύ': "Y", 'Ώ': "O",
	'Ϊ': "I", 'Ϋ': "Y",
}

// georgianFold is the national transliteration of the Georgian alphabet, which
// is unicameral: one case, one mapping.
var georgianFold = map[rune]string{
	'ა': "a", 'ბ': "b", 'გ': "g", 'დ': "d", 'ე': "e", 'ვ': "v", 'ზ': "z",
	'თ': "t", 'ი': "i", 'კ': "k", 'ლ': "l", 'მ': "m", 'ნ': "n", 'ო': "o",
	'პ': "p", 'ჟ': "zh", 'რ': "r", 'ს': "s", 'ტ': "t", 'უ': "u", 'ფ': "p",
	'ქ': "k", 'ღ': "gh", 'ყ': "q", 'შ': "sh", 'ჩ': "ch", 'ც': "ts",
	'ძ': "dz", 'წ': "ts", 'ჭ': "ch", 'ხ': "kh", 'ჯ': "j", 'ჰ': "h",
}

// greekDigraphs are the Greek vowel pairs that do not transliterate letter by
// letter: upsilon is "y" on its own but the second half of a diphthong
// otherwise, so Θεοδώρου is Theodorou and not Theodoroy.
// The accented forms are listed too: the accent sits on the pair's first
// letter (Παύλος), so matching only the unaccented pair would miss half of
// them.
var greekDigraphs = strings.NewReplacer(
	"ου", "ou", "Ου", "Ou", "ΟΥ", "OU", "ού", "ou", "Ού", "Ou",
	"αυ", "au", "Αυ", "Au", "ΑΥ", "AU", "αύ", "au", "Αύ", "Au",
	"ευ", "eu", "Ευ", "Eu", "ΕΥ", "EU", "εύ", "eu", "Εύ", "Eu",
	"ηυ", "iu", "Ηυ", "Iu", "ΗΥ", "IU", "ηύ", "iu", "Ηύ", "Iu",
)

// foldASCII rewrites a name into ASCII letters and digits. Anything it has no
// mapping for is dropped, so the caller must handle an empty result: a Chinese
// or Thai name folds to nothing.
func foldASCII(s string) string {
	s = greekDigraphs.Replace(s)
	var b []byte
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b = append(b, byte(r))
			continue
		case r < 128:
			continue // punctuation and spaces are not local-part material
		}
		for _, table := range []map[rune]string{latinFold, latinExtra, cyrillicFold, greekFold, georgianFold} {
			if v, ok := table[r]; ok {
				b = append(b, v...)
				break
			}
		}
	}
	return string(b)
}
