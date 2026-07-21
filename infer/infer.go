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
	"amount": schema.KindAmount, "price": schema.KindAmount, "total": schema.KindAmount, "balance": schema.KindAmount,
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
	var nameField, timeCreated string
	for i := range s.Fields {
		switch s.Fields[i].Kind {
		case schema.KindName, schema.KindFirstName:
			if nameField == "" {
				nameField = s.Fields[i].Name
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
