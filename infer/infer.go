// Package infer turns an untagged field (its name + Go type) into a
// schema.Kind. This is the "just works without tags" path: a synonym table
// first, then a Go-type fallback, then a warning.
package infer

import (
	"strings"

	"github.com/bakhodir/synth/schema"
)

// synonyms maps normalized field-name aliases to a Kind. Extendable via Alias.
var synonyms = map[string]schema.Kind{
	"id": schema.KindUUID, "uuid": schema.KindUUID, "guid": schema.KindUUID,
	"name": schema.KindName, "fullname": schema.KindName, "displayname": schema.KindName,
	"firstname": schema.KindFirstName, "givenname": schema.KindFirstName,
	"lastname": schema.KindLastName, "surname": schema.KindLastName, "familyname": schema.KindLastName,
	"email": schema.KindEmail, "mail": schema.KindEmail, "emailaddress": schema.KindEmail,
	"phone": schema.KindPhone, "tel": schema.KindPhone, "mobile": schema.KindPhone, "phonenumber": schema.KindPhone,
	"city": schema.KindCity, "town": schema.KindCity,
	"region": schema.KindRegion, "state": schema.KindRegion, "province": schema.KindRegion, "viloyat": schema.KindRegion,
	"country": schema.KindCountry, "davlat": schema.KindCountry, "nation": schema.KindCountry,
	"postcode": schema.KindPostcode, "zip": schema.KindPostcode, "zipcode": schema.KindPostcode, "postal": schema.KindPostcode,
	"iban": schema.KindIBAN,
	"card": schema.KindCard, "cardnumber": schema.KindCard, "pan": schema.KindCard,
	"passport": schema.KindPassport,
	"age":      schema.KindInt,
	"company":  schema.KindCompany, "employer": schema.KindCompany, "organization": schema.KindCompany,
	"username": schema.KindUsername, "login": schema.KindUsername, "handle": schema.KindUsername,
	"ip": schema.KindIPv4, "ipaddress": schema.KindIPv4, "ipv4": schema.KindIPv4,
	"url": schema.KindURL, "website": schema.KindURL, "link": schema.KindURL,
	"currency": schema.KindCurrency, "ccy": schema.KindCurrency,
	"amount": schema.KindAmount, "price": schema.KindAmount, "total": schema.KindAmount,
	"bio": schema.KindLorem, "description": schema.KindLorem, "notes": schema.KindLorem,
	"street": schema.KindStreet, "streetaddress": schema.KindStreet, "address": schema.KindStreet,
	"color": schema.KindColor, "colour": schema.KindColor,
	"hexcolor": schema.KindHexColor, "hex": schema.KindHexColor,
	"job": schema.KindJob, "jobtitle": schema.KindJob, "position": schema.KindJob, "title": schema.KindJob,
	"product": schema.KindProduct, "item": schema.KindProduct,
	"mac": schema.KindMAC, "macaddress": schema.KindMAC,
	"gender": schema.KindGender, "sex": schema.KindGender,
	"book": schema.KindBook, "booktitle": schema.KindBook, "novel": schema.KindBook,
	"movie": schema.KindMovie, "film": schema.KindMovie, "cinema": schema.KindMovie,
	"celebrity": schema.KindCelebrity, "famousperson": schema.KindCelebrity, "star": schema.KindCelebrity,
	"band": schema.KindBand, "artist": schema.KindBand, "musician": schema.KindBand,
	"brand": schema.KindBrand,
	"food":  schema.KindFood, "dish": schema.KindFood, "meal": schema.KindFood,
	"animal": schema.KindAnimal, "pet": schema.KindAnimal,
	"sport":      schema.KindSport,
	"planet":     schema.KindPlanet,
	"university": schema.KindUniversity, "college": schema.KindUniversity, "school": schema.KindUniversity,
	"language": schema.KindLanguage, "proglang": schema.KindLanguage,
	"programminglanguage": schema.KindProgrammingLanguage, "codinglanguage": schema.KindProgrammingLanguage,
	"humanlanguage": schema.KindHumanLanguage, "spokenlanguage": schema.KindHumanLanguage,
	"nativelanguage": schema.KindHumanLanguage, "motherTongue": schema.KindHumanLanguage,
	"cardexpiry": schema.KindCardExpiry, "expiry": schema.KindCardExpiry,
	"expirydate": schema.KindCardExpiry, "expdate": schema.KindCardExpiry,
	"expiresat": schema.KindCardExpiry, "validthru": schema.KindCardExpiry,
	"cvv": schema.KindCVV, "cvc": schema.KindCVV, "securitycode": schema.KindCVV,
	"balance": schema.KindBalance, "accountbalance": schema.KindBalance,
	"passphrase":   schema.KindPassphrase,
	"passwordhash": schema.KindPasswordHash, "pwhash": schema.KindPasswordHash,
	"hashedpassword": schema.KindPasswordHash,
	"cardbrand":      schema.KindCardBrand, "cardtype": schema.KindCardBrand,
	"paymentbrand": schema.KindCardBrand, "cardscheme": schema.KindCardBrand,
	"bodypart": schema.KindBodyPart, "organ": schema.KindBodyPart, "anatomy": schema.KindBodyPart,
	"emoji":     schema.KindEmoji,
	"word":      schema.KindWord,
	"sentence":  schema.KindSentence,
	"paragraph": schema.KindParagraph, "text": schema.KindParagraph, "content": schema.KindParagraph,
	"ipv6":   schema.KindIPv6,
	"domain": schema.KindDomain, "domainname": schema.KindDomain,
	"latitude": schema.KindLatitude, "lat": schema.KindLatitude,
	"longitude": schema.KindLongitude, "lng": schema.KindLongitude, "lon": schema.KindLongitude,
	"unixtime": schema.KindUnixTime, "timestamp": schema.KindUnixTime,
	"month":   schema.KindMonth,
	"weekday": schema.KindWeekday, "dayofweek": schema.KindWeekday,
	"year":      schema.KindYear,
	"bloodtype": schema.KindBloodType, "blood": schema.KindBloodType,
	"useragent": schema.KindUserAgent, "ua": schema.KindUserAgent,
	"salutation": schema.KindTitle, "honorific": schema.KindTitle,
	"imageurl": schema.KindImageURL, "avatar": schema.KindImageURL, "image": schema.KindImageURL,
	"ssn":      schema.KindSSN,
	"timezone": schema.KindTimezone, "tz": schema.KindTimezone,
	"vehiclemake": schema.KindVehicleMake, "carmake": schema.KindVehicleMake, "make": schema.KindVehicleMake,
	"vehiclemodel": schema.KindVehicleModel, "carmodel": schema.KindVehicleModel, "model": schema.KindVehicleModel,
	"vehicletype": schema.KindVehicleType,
	"department":  schema.KindDepartment, "dept": schema.KindDepartment,
	"jobarea":  schema.KindJobArea,
	"joblevel": schema.KindJobLevel, "seniority": schema.KindJobLevel,
	"productcategory": schema.KindProductCategory, "category": schema.KindProductCategory,
	"productmaterial": schema.KindProductMaterial, "material": schema.KindProductMaterial,
	"musicgenre": schema.KindMusicGenre, "genre": schema.KindMusicGenre,
	"instrument": schema.KindInstrument,
	"team":       schema.KindSportsTeam, "sportsteam": schema.KindSportsTeam,
	"framework": schema.KindFramework,
	"mimetype":  schema.KindMimeType, "contenttype": schema.KindMimeType,
	"httpmethod": schema.KindHTTPMethod, "method": schema.KindHTTPMethod,
	"httpstatus": schema.KindHTTPStatus, "statuscode": schema.KindHTTPStatus,
	"os": schema.KindOS, "operatingsystem": schema.KindOS,
	"browser": schema.KindBrowser,
	"device":  schema.KindDevice,
	"airport": schema.KindAirport, "airportcode": schema.KindAirport,
	"airline":     schema.KindAirline,
	"stockticker": schema.KindStockTicker, "ticker": schema.KindStockTicker, "symbol": schema.KindStockTicker,
	"crypto": schema.KindCrypto, "cryptocurrency": schema.KindCrypto,
	"continent":    schema.KindContinent,
	"languagename": schema.KindLanguageName,
	"fruit":        schema.KindFruit,
	"vegetable":    schema.KindVegetable,
	"drink":        schema.KindDrink, "beverage": schema.KindDrink,
	"dogbreed": schema.KindDogBreed,
	"catbreed": schema.KindCatBreed,
	"flower":   schema.KindFlower,
	"gemstone": schema.KindGemstone, "gem": schema.KindGemstone,
	"metal":  schema.KindMetal,
	"zodiac": schema.KindZodiac, "starsign": schema.KindZodiac, "horoscope": schema.KindZodiac,
	"countrycode": schema.KindCountryCode, "iso": schema.KindCountryCode,
	"currencyname":   schema.KindCurrencyName,
	"currencysymbol": schema.KindCurrencySymbol,
	"catchphrase":    schema.KindCatchPhrase, "slogan": schema.KindCatchPhrase, "tagline": schema.KindCatchPhrase,
	"appname": schema.KindAppName, "app": schema.KindAppName,
	"fileext": schema.KindFileExt, "extension": schema.KindFileExt,
	"semver": schema.KindSemver, "version": schema.KindSemver,
	"rgbcolor": schema.KindRGBColor, "rgb": schema.KindRGBColor,
	"ean": schema.KindEAN13, "ean13": schema.KindEAN13, "barcode": schema.KindEAN13,
	"isbn":     schema.KindISBN,
	"password": schema.KindPassword, "pwd": schema.KindPassword, "pass": schema.KindPassword,
	"slug": schema.KindSlug, "permalink": schema.KindSlug,
	"middlename": schema.KindMiddleName,
	"namesuffix": schema.KindNameSuffix, "suffix": schema.KindNameSuffix,
	"maritalstatus": schema.KindMaritalStatus, "marital": schema.KindMaritalStatus,
	"education": schema.KindEducation, "degree": schema.KindEducation,
	"bankname": schema.KindBankName, "bank": schema.KindBankName,
	"accounttype":   schema.KindAccountType,
	"paymentmethod": schema.KindPaymentMethod, "payment": schema.KindPaymentMethod,
	"swift": schema.KindSwift, "bic": schema.KindSwift,
	"vin":          schema.KindVIN,
	"licenseplate": schema.KindLicensePlate, "plate": schema.KindLicensePlate,
	"md5":    schema.KindMD5,
	"sha256": schema.KindSHA256, "sha": schema.KindSHA256, "hash": schema.KindSHA256,
	"jwt": schema.KindJWT, "token": schema.KindJWT,
	"gitcommit": schema.KindGitCommit, "commit": schema.KindGitCommit, "sha1": schema.KindGitCommit,
	"gitbranch": schema.KindGitBranch, "branch": schema.KindGitBranch,
	"filename": schema.KindFileName,
	"filepath": schema.KindFilePath, "path": schema.KindFilePath,
	"port":     schema.KindPort,
	"protocol": schema.KindProtocol, "scheme": schema.KindProtocol,
	"htmltag": schema.KindHTMLTag, "tag": schema.KindHTMLTag,
	"weather":   schema.KindWeather,
	"season":    schema.KindSeason,
	"direction": schema.KindDirection,
	"element":   schema.KindElement, "chemicalelement": schema.KindElement,
	"constellation": schema.KindConstellation,
	"shape":         schema.KindShape,
	"social":        schema.KindSocial, "socialplatform": schema.KindSocial, "platform": schema.KindSocial,
	"sku":         schema.KindSKU,
	"chesspiece":  schema.KindChessPiece,
	"unit":        schema.KindUnit,
	"temperature": schema.KindTemperature, "temp": schema.KindTemperature,
	"percentage": schema.KindPercentage, "percent": schema.KindPercentage,
	"rating": schema.KindRating, "score": schema.KindRating, "stars": schema.KindRating,
	"salary": schema.KindSalary, "wage": schema.KindSalary, "income": schema.KindSalary,
	"ein": schema.KindEIN, "taxid": schema.KindEIN,
	"base64":      schema.KindBase64,
	"orderstatus": schema.KindOrderStatus, "status": schema.KindOrderStatus,
	"couponcode": schema.KindCouponCode, "coupon": schema.KindCouponCode, "promocode": schema.KindCouponCode, "discountcode": schema.KindCouponCode,
	"nickname": schema.KindNickname, "nick": schema.KindNickname, "alias": schema.KindNickname,
	"cocktail":  schema.KindCocktail,
	"coffee":    schema.KindCoffee,
	"superhero": schema.KindSuperhero, "hero": schema.KindSuperhero,
	"petname":  schema.KindPetName,
	"loglevel": schema.KindLogLevel, "level": schema.KindLogLevel,
	"environment": schema.KindEnvironment, "env": schema.KindEnvironment, "stage": schema.KindEnvironment,
	"awsregion":     schema.KindAWSRegion,
	"cloudprovider": schema.KindCloudProvider, "cloud": schema.KindCloudProvider,
	"containerimage": schema.KindContainerImage, "dockerimage": schema.KindContainerImage,
	"httpheader": schema.KindHTTPHeader, "header": schema.KindHTTPHeader,
	"keyboardkey": schema.KindKeyboardKey, "key": schema.KindKeyboardKey,
	"musicnote": schema.KindMusicNote, "note": schema.KindMusicNote,
	"medal":      schema.KindMedal,
	"tshirtsize": schema.KindTShirtSize, "size": schema.KindTShirtSize,
	"priority":      schema.KindPriority,
	"imei":          schema.KindIMEI,
	"upc":           schema.KindUPC,
	"routingnumber": schema.KindRoutingNumber, "routing": schema.KindRoutingNumber, "aba": schema.KindRoutingNumber,
	"accountnumber": schema.KindAccountNumber, "account": schema.KindAccountNumber,
	"errorcode": schema.KindErrorCode,
	"cron":      schema.KindCron, "cronexpression": schema.KindCron, "schedule": schema.KindCron,
	"filesize": schema.KindFileSize,
	"duration": schema.KindDuration,
	"gittag":   schema.KindGitTag, "tag2": schema.KindGitTag,
	// Regulated-domain identifiers.
	"icd10": schema.KindICD10, "icd": schema.KindICD10,
	"diagnosiscode": schema.KindICD10, "diagnosis": schema.KindICD10,
	"ndc": schema.KindNDC, "drugcode": schema.KindNDC,
	"drugname": schema.KindDrugName, "medication": schema.KindDrugName,
	"drug":  schema.KindDrugName,
	"isin":  schema.KindISIN,
	"lei":   schema.KindLEI,
	"cusip": schema.KindCUSIP,
	"cidr":  schema.KindCIDR, "subnet": schema.KindCIDR, "network": schema.KindCIDR,
	"asn":       schema.KindASN,
	"macvendor": schema.KindMACVendor, "oui": schema.KindMACVendor,
	"geojson": schema.KindGeoJSONPoint, "geojsonpoint": schema.KindGeoJSONPoint,
	"geometry": schema.KindGeoJSONPoint,
}

// Alias registers an extra field-name synonym (e.g. Uzbek "ismi" → name).
func Alias(fieldName string, kind schema.Kind) {
	synonyms[normalize(fieldName)] = kind
}

// normalize lowercases and strips separators: "Full_Name" → "fullname".
func normalize(s string) string {
	s = strings.ToLower(s)
	r := strings.NewReplacer("_", "", "-", "", " ", "")
	return r.Replace(s)
}

// Kind infers a field's kind from its name, then its Go type.
// Returns (kind, matchedByName). KindUnknown means the caller should warn.
func Kind(fieldName, goType string) (schema.Kind, bool) {
	if k, ok := synonyms[normalize(fieldName)]; ok {
		return k, true
	}
	switch goType {
	case "time.Time":
		return schema.KindTime, false
	case "uuid.UUID":
		return schema.KindUUID, false
	case "bool":
		return schema.KindBool, false
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return schema.KindInt, false
	case "float32", "float64":
		return schema.KindFloat, false
	case "string":
		return schema.KindLorem, false
	}
	return schema.KindUnknown, false
}

// LinkDependencies wires up automatic coherence between inferred fields:
// name→email (from), city↔postcode↔region share a Place, and multiple time
// fields get an ordering hint by name semantics.
func LinkDependencies(s *schema.Schema) {
	var nameField, timeCreated, cardField, airportField string
	for i := range s.Fields {
		switch s.Fields[i].Kind {
		case schema.KindName, schema.KindFirstName:
			if nameField == "" {
				nameField = s.Fields[i].Name
			}
		case schema.KindCard:
			if cardField == "" {
				cardField = s.Fields[i].Name
			}
		case schema.KindAirport:
			if airportField == "" {
				airportField = s.Fields[i].Name
			}
		}
	}
	// An airport name beside an airport code must be that airport.
	if airportField != "" {
		for i := range s.Fields {
			f := &s.Fields[i]
			if f.Kind == schema.KindAirportName && f.From == "" {
				f.From = airportField
			}
		}
	}
	// A card brand next to a card number must describe that number. Deriving
	// it beats drawing it: a "MasterCard" label on a 4539… number is the kind
	// of incoherence that makes test data useless for payment code.
	if cardField != "" {
		for i := range s.Fields {
			f := &s.Fields[i]
			if (f.Kind == schema.KindCardBrand || f.Kind == schema.KindCVV) && f.From == "" {
				f.From = cardField
			}
		}
	}
	// email derives from the name field, when the email has no explicit from.
	for i := range s.Fields {
		f := &s.Fields[i]
		if f.Kind == schema.KindEmail && f.From == "" && nameField != "" {
			f.From = nameField
		}
	}
	// time ordering: a "created" field becomes the anchor; "updated"/
	// "deleted" come after it.
	for i := range s.Fields {
		f := &s.Fields[i]
		if f.Kind != schema.KindTime {
			continue
		}
		n := normalize(f.Name)
		if strings.Contains(n, "creat") && timeCreated == "" {
			timeCreated = f.Name
		}
	}
	if timeCreated != "" {
		for i := range s.Fields {
			f := &s.Fields[i]
			if f.Kind != schema.KindTime || f.Name == timeCreated {
				continue
			}
			n := normalize(f.Name)
			if strings.Contains(n, "updat") || strings.Contains(n, "delet") || strings.Contains(n, "modif") {
				f.From = timeCreated // engine reads From on time as "after"
			}
		}
	}
}
