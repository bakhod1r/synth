// Package schema defines the intermediate representation (IR) that Synth's
// engine consumes. Frontends (reflection over structs, or later YAML) produce a
// Schema; the engine and encoders only ever see this — never Go structs.
package schema

// Kind is the semantic type of a field, independent of its Go type.
type Kind string

const (
	KindUUID      Kind = "uuid"
	KindName      Kind = "name"
	KindFirstName Kind = "firstname"
	KindLastName  Kind = "lastname"
	KindEmail     Kind = "email"
	KindPhone     Kind = "phone"
	KindCity      Kind = "city"
	KindRegion    Kind = "region"
	KindCountry   Kind = "country"
	KindPostcode  Kind = "postcode"
	KindInt       Kind = "int"
	KindFloat     Kind = "float"
	KindBool      Kind = "bool"
	KindTime      Kind = "time"
	KindLorem     Kind = "lorem"
	KindIBAN      Kind = "iban"
	KindCard      Kind = "card"
	KindPassport  Kind = "passport"
	KindCompany   Kind = "company"
	KindUsername  Kind = "username"
	KindIPv4      Kind = "ipv4"
	KindURL       Kind = "url"
	KindCurrency  Kind = "currency"
	KindAmount    Kind = "amount"
	KindEnum      Kind = "enum"
	KindStreet    Kind = "street"
	KindColor     Kind = "color"
	KindJob       Kind = "job"
	KindProduct   Kind = "product"
	KindMAC       Kind = "mac"
	KindHexColor  Kind = "hexcolor"
	KindGender    Kind = "gender"
	KindBook      Kind = "book"
	KindMovie     Kind = "movie"
	KindCelebrity Kind = "celebrity"
	KindBand      Kind = "band"
	KindBrand     Kind = "brand"
	// KindCountryName is a random famous country (not tied to the locale, unlike
	// KindCountry which returns the locale's own country).
	KindCountryName Kind = "countryname"
	KindFood        Kind = "food"
	KindAnimal      Kind = "animal"
	KindSport       Kind = "sport"
	KindPlanet      Kind = "planet"
	KindUniversity  Kind = "university"
	KindLanguage    Kind = "language"
	KindEmoji       Kind = "emoji"
	// go-faker parity additions.
	KindWord      Kind = "word"
	KindSentence  Kind = "sentence"
	KindParagraph Kind = "paragraph"
	KindIPv6      Kind = "ipv6"
	KindDomain    Kind = "domain"
	KindLatitude  Kind = "latitude"
	KindLongitude Kind = "longitude"
	KindUnixTime  Kind = "unixtime"
	KindMonth     Kind = "month"
	KindWeekday   Kind = "weekday"
	KindYear      Kind = "year"
	KindBloodType Kind = "bloodtype"
	KindUserAgent Kind = "useragent"
	KindTitle     Kind = "title"
	KindImageURL  Kind = "imageurl"
	KindSSN       Kind = "ssn"
	KindTimezone  Kind = "timezone"
	// Catalog types.
	KindVehicleMake     Kind = "vehiclemake"
	KindVehicleModel    Kind = "vehiclemodel"
	KindVehicleType     Kind = "vehicletype"
	KindDepartment      Kind = "department"
	KindJobArea         Kind = "jobarea"
	KindJobLevel        Kind = "joblevel"
	KindProductCategory Kind = "productcategory"
	KindProductMaterial Kind = "productmaterial"
	KindMusicGenre      Kind = "musicgenre"
	KindInstrument      Kind = "instrument"
	KindSportsTeam      Kind = "sportsteam"
	KindFramework       Kind = "framework"
	KindMimeType        Kind = "mimetype"
	KindHTTPMethod      Kind = "httpmethod"
	KindHTTPStatus      Kind = "httpstatus"
	KindOS              Kind = "os"
	KindBrowser         Kind = "browser"
	KindDevice          Kind = "device"
	KindAirport         Kind = "airport"
	KindAirline         Kind = "airline"
	KindStockTicker     Kind = "stockticker"
	KindCrypto          Kind = "crypto"
	KindContinent       Kind = "continent"
	KindLanguageName    Kind = "languagename"
	KindFruit           Kind = "fruit"
	KindVegetable       Kind = "vegetable"
	KindDrink           Kind = "drink"
	KindDogBreed        Kind = "dogbreed"
	KindCatBreed        Kind = "catbreed"
	KindFlower          Kind = "flower"
	KindGemstone        Kind = "gemstone"
	KindMetal           Kind = "metal"
	KindZodiac          Kind = "zodiac"
	KindCountryCode     Kind = "countrycode"
	KindCurrencyName    Kind = "currencyname"
	KindCurrencySymbol  Kind = "currencysymbol"
	KindCatchPhrase     Kind = "catchphrase"
	KindAppName         Kind = "appname"
	KindFileExt         Kind = "fileext"
	KindSemver          Kind = "semver"
	KindRGBColor        Kind = "rgbcolor"
	KindEAN13           Kind = "ean13"
	KindISBN            Kind = "isbn"
	KindPassword        Kind = "password"
	KindSlug            Kind = "slug"
	// Catalog batch 2.
	KindMiddleName    Kind = "middlename"
	KindNameSuffix    Kind = "namesuffix"
	KindMaritalStatus Kind = "maritalstatus"
	KindEducation     Kind = "education"
	KindBankName      Kind = "bankname"
	KindAccountType   Kind = "accounttype"
	KindPaymentMethod Kind = "paymentmethod"
	KindSwift         Kind = "swift"
	KindVIN           Kind = "vin"
	KindLicensePlate  Kind = "licenseplate"
	KindMD5           Kind = "md5"
	KindSHA256        Kind = "sha256"
	KindJWT           Kind = "jwt"
	KindGitCommit     Kind = "gitcommit"
	KindGitBranch     Kind = "gitbranch"
	KindFileName      Kind = "filename"
	KindFilePath      Kind = "filepath"
	KindPort          Kind = "port"
	KindProtocol      Kind = "protocol"
	KindHTMLTag       Kind = "htmltag"
	KindWeather       Kind = "weather"
	KindSeason        Kind = "season"
	KindDirection     Kind = "direction"
	KindElement       Kind = "element"
	KindConstellation Kind = "constellation"
	KindShape         Kind = "shape"
	KindSocial        Kind = "social"
	KindSKU           Kind = "sku"
	KindChessPiece    Kind = "chesspiece"
	KindUnit          Kind = "unit"
	KindTemperature   Kind = "temperature"
	KindPercentage    Kind = "percentage"
	KindRating        Kind = "rating"
	KindSalary        Kind = "salary"
	KindEIN           Kind = "ein"
	KindBase64        Kind = "base64"
	KindOrderStatus   Kind = "orderstatus"
	KindCouponCode    Kind = "couponcode"
	KindNickname      Kind = "nickname"
	// Catalog batch 3.
	KindCocktail       Kind = "cocktail"
	KindCoffee         Kind = "coffee"
	KindSuperhero      Kind = "superhero"
	KindPetName        Kind = "petname"
	KindLogLevel       Kind = "loglevel"
	KindEnvironment    Kind = "environment"
	KindAWSRegion      Kind = "awsregion"
	KindCloudProvider  Kind = "cloudprovider"
	KindContainerImage Kind = "containerimage"
	KindHTTPHeader     Kind = "httpheader"
	KindKeyboardKey    Kind = "keyboardkey"
	KindMusicNote      Kind = "musicnote"
	KindMedal          Kind = "medal"
	KindTShirtSize     Kind = "tshirtsize"
	KindPriority       Kind = "priority"

	// KindTimeSeries is a numeric value that follows a curve over time —
	// base + trend + seasonality + noise — read from a named timestamp axis.
	KindTimeSeries Kind = "timeseries"

	// Regulated-domain identifiers: healthcare, finance, networking.
	KindICD10        Kind = "icd10"
	KindNDC          Kind = "ndc"
	KindDrugName     Kind = "drugname"
	KindISIN         Kind = "isin"
	KindLEI          Kind = "lei"
	KindCUSIP        Kind = "cusip"
	KindCIDR         Kind = "cidr"
	KindASN          Kind = "asn"
	KindMACVendor    Kind = "macvendor"
	KindGeoJSONPoint Kind = "geojsonpoint"

	// KindProgrammingLanguage and KindHumanLanguage are the clear names for
	// what "language" and "languagename" mean. The old names stay valid, but
	// they were easy to mix up: one gives Go or Rust, the other Uzbek or
	// Japanese.
	KindProgrammingLanguage Kind = "programminglanguage"
	KindHumanLanguage       Kind = "humanlanguage"
	// KindBodyPart is anatomy in the record's own language.
	KindBodyPart Kind = "bodypart"
	// KindCardBrand is the payment brand of a card. With from=<card column> it
	// is read off that number's issuer prefix, so the two always agree.
	KindCardBrand Kind = "cardbrand"
	// KindCardExpiry is a card expiry date, MM/YY by default and always in the
	// future unless expired=true asks for a lapsed card.
	KindCardExpiry Kind = "cardexpiry"
	// KindCVV is a card security code, three digits except on American
	// Express, which uses four.
	KindCVV Kind = "cvv"
	// KindBalance is an account balance. Unlike KindAmount it may be negative,
	// because overdrafts exist and code that assumes otherwise breaks on them.
	KindBalance Kind = "balance"
	// KindPassphrase is a memorable word-based password.
	KindPassphrase Kind = "passphrase"
	// KindPasswordHash is a self-describing PBKDF2 hash of a generated
	// password, for seeding a credentials table.
	KindPasswordHash Kind = "passwordhash"

	// Named things: publications, vessels, aircraft, venues and hardware.
	KindArticleTitle Kind = "articletitle"
	KindMagazine     Kind = "magazine"
	KindShipName     Kind = "shipname"
	KindShipType     Kind = "shiptype"
	KindAircraft     Kind = "aircraft"
	KindTrainModel   Kind = "trainmodel"
	KindHotel        Kind = "hotel"
	KindComputer     Kind = "computer"
	// KindAirportName is the airport's full name. With from=<code column> it
	// names that airport rather than a different one.
	KindAirportName Kind = "airportname"

	// Entertainment and gaming.
	KindMovieCharacter Kind = "moviecharacter"
	KindGame           Kind = "game"
	KindGamertag       Kind = "gamertag"

	// KindCardToken is a payment token — the opaque reference a gateway hands
	// back in place of a card number. It is not derived from any card: that is
	// the whole point of tokenization.
	KindCardToken Kind = "cardtoken"

	// Roads and anatomy.
	KindStreetType Kind = "streettype"
	KindBird       Kind = "bird"
	KindDisease    Kind = "disease"

	// Catalog batch 4: media, nature, medicine, transport, civic life.
	KindTVShow           Kind = "tvshow"
	KindAnime            Kind = "anime"
	KindPodcast          Kind = "podcast"
	KindNewspaper        Kind = "newspaper"
	KindMusicAlbum       Kind = "musicalbum"
	KindBoardGame        Kind = "boardgame"
	KindConsole          Kind = "console"
	KindRestaurant       Kind = "restaurant"
	KindClothingBrand    Kind = "clothingbrand"
	KindWatchBrand       Kind = "watchbrand"
	KindCameraModel      Kind = "cameramodel"
	KindPhoneModel       Kind = "phonemodel"
	KindTree             Kind = "tree"
	KindInsect           Kind = "insect"
	KindFish             Kind = "fish"
	KindHerb             Kind = "herb"
	KindSpice            Kind = "spice"
	KindDinosaur         Kind = "dinosaur"
	KindMineral          Kind = "mineral"
	KindMedicalSpecialty Kind = "medicalspecialty"
	KindLabTest          Kind = "labtest"
	KindProcedure        Kind = "procedure"
	KindVaccine          Kind = "vaccine"
	KindAllergy          Kind = "allergy"
	KindSymptom          Kind = "symptom"
	KindMilitaryRank     Kind = "militaryrank"
	KindReligion         Kind = "religion"
	KindHoliday          Kind = "holiday"
	KindInsuranceType    Kind = "insurancetype"
	KindCrimeType        Kind = "crimetype"
	KindTaxType          Kind = "taxtype"
	KindSpacecraft       Kind = "spacecraft"
	KindMotorcycle       Kind = "motorcycle"
	KindBicycleType      Kind = "bicycletype"
	KindFuelType         Kind = "fueltype"
	KindNaturalDisaster  Kind = "naturaldisaster"
	KindMartialArt       Kind = "martialart"
	KindDance            Kind = "dance"
	KindChessOpening     Kind = "chessopening"
	KindWineVariety      Kind = "winevariety"
	KindCheese           Kind = "cheese"
	KindStadium          Kind = "stadium"
	KindMuseum           Kind = "museum"
	KindMountain         Kind = "mountain"
	KindBirthDate        Kind = "birthdate"
	KindPINFL            Kind = "pinfl"
	KindCVC              Kind = "cvc"
	KindCVV2             Kind = "cvv2"
	KindCVC2             Kind = "cvc2"
	KindCSC              Kind = "csc"
	KindCID              Kind = "cid"
	KindNationalID       Kind = "nationalid"
	KindTaxID            Kind = "taxid"
	KindAge              Kind = "age"
	KindRiver            Kind = "river"
	KindIMEI             Kind = "imei"
	KindUPC              Kind = "upc"
	KindRoutingNumber    Kind = "routingnumber"
	KindAccountNumber    Kind = "accountnumber"
	KindErrorCode        Kind = "errorcode"
	KindCron             Kind = "cron"
	KindFileSize         Kind = "filesize"
	KindDuration         Kind = "duration"
	KindGitTag           Kind = "gittag"
	// KindUnknown marks a field the frontend could not infer. The engine
	// leaves it at its zero value and records a Warning.
	KindUnknown Kind = ""
	// KindObject is a nested struct; its shape is in Field.Nested.
	KindObject Kind = "object"
	// KindArray is a slice; its element shape is in Field.Elem.
	KindArray Kind = "array"
)

// Field is one column of a record.
type Field struct {
	// Name is the Go field name (also the CSV/SQL column header).
	Name string
	Kind Kind
	// GoType is the reflect kind hint ("string", "int", "time.Time"...),
	// used by encoders and by min/max coercion.
	GoType string
	// Params holds parsed tag options, e.g. {"min":"18","max":"65"}.
	Params map[string]string
	// From names another field this one derives from (email from=Name).
	From string
	// Match names another field this one must stay coherent with
	// (postcode match=City).
	Match string
	// Unique requests distinct values across the generated set.
	Unique bool
	// PK marks the primary key, used as the join target for Ref.
	PK bool
	// FromRef, when set, means this field is a foreign key filled from a
	// referenced parent slice (see synth.Ref). Holds the parent PK values.
	FromRef []any
	// RefCardinality controls how FromRef values are distributed
	// (0 means uniform-random).
	RefMin, RefMax int
	// Choices holds the value set for KindEnum.
	Choices []string
	// Weights, if set, are relative probabilities for Choices (same length).
	// If empty, Choices are picked uniformly (or by Zipf if Params["dist"]
	// == "zipf").
	Weights []float64
	// Nested is the sub-schema for a KindObject field.
	Nested *Schema
	// Elem describes the element of a KindArray field (its Kind, and for
	// arrays of structs, its own Nested schema).
	Elem *Field
	// ArrMin/ArrMax bound the length of a KindArray field (default 1..3).
	ArrMin, ArrMax int
}

// Schema is an ordered list of fields describing one record type.
type Schema struct {
	Fields []Field
}

// FieldByName returns the field with the given name, or nil.
func (s *Schema) FieldByName(name string) *Field {
	for i := range s.Fields {
		if s.Fields[i].Name == name {
			return &s.Fields[i]
		}
	}
	return nil
}

// Warning records a field the frontend could not fully handle.
type Warning struct {
	Field  string
	Reason string
}
