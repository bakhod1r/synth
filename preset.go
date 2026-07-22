package synth

import (
	"fmt"

	"github.com/bakhodir/synth/yamlfe"
)

// Presets: ready-made schemas for the tables almost every project needs.
//
// "Give me 100 fake transactions" should be one call, not a schema-design
// exercise. Each preset is a plain YAML spec, so it is also a worked example:
// print it, edit it, commit it.
//
// Sensitive columns are masked by default. A card number in a fixture ends up
// in a ticket, a screenshot or a log sooner or later, and defaulting to the
// safe form means nobody has to remember.

// Preset names a built-in schema.
type Preset string

const (
	PresetUser        Preset = "user"
	PresetPayment     Preset = "payment"
	PresetTransaction Preset = "transaction"
	PresetOrder       Preset = "order"
	PresetProduct     Preset = "product"
	PresetEmployee    Preset = "employee"
	PresetPatient     Preset = "patient"
	PresetEvent       Preset = "event"
)

// Presets lists every built-in schema.
func Presets() []Preset {
	return []Preset{
		PresetUser, PresetPayment, PresetTransaction, PresetOrder,
		PresetProduct, PresetEmployee, PresetPatient, PresetEvent,
	}
}

// PresetSpec returns a preset's YAML, so it can be read, edited and committed
// rather than treated as a black box.
func PresetSpec(p Preset) (string, bool) {
	s, ok := presetSpecs[p]
	return s, ok
}

// Generate produces n records from a built-in schema.
//
//	rows, _ := synth.Generate(synth.PresetTransaction, 100)
//
// Card numbers and national identifiers come back masked. Pass Unmasked() when
// a test genuinely needs the raw value — for example to check that a validator
// accepts it.
func Generate(p Preset, n int, opts ...Option) ([]map[string]any, error) {
	spec, ok := presetSpecs[p]
	if !ok {
		return nil, fmt.Errorf("synth: unknown preset %q", p)
	}
	y, err := YAMLBytes([]byte(spec))
	if err != nil {
		return nil, err
	}
	return y.GenerateN(n, opts...)
}

// GenerateN generates exactly n records, overriding the spec's own count.
func (y *YAMLSpec) GenerateN(n int, opts ...Option) ([]map[string]any, error) {
	if n > 0 {
		y.spec.Count = n
	}
	return y.Generate(opts...)
}

// Unmasked strips the mask= setting from every field, returning raw values.
//
// The name is deliberate: at the call site it reads as a decision, not a
// default. Output from this option must not be pasted anywhere a real value
// would be unwelcome.
func Unmasked() Option {
	return func(c *config) { c.unmask = true }
}

// Users, Payments and the rest are shorthands for the common presets.
func Users(n int, opts ...Option) ([]map[string]any, error) {
	return Generate(PresetUser, n, opts...)
}

func Payments(n int, opts ...Option) ([]map[string]any, error) {
	return Generate(PresetPayment, n, opts...)
}

func Transactions(n int, opts ...Option) ([]map[string]any, error) {
	return Generate(PresetTransaction, n, opts...)
}

func Orders(n int, opts ...Option) ([]map[string]any, error) {
	return Generate(PresetOrder, n, opts...)
}

// Spec turns a preset into an editable spec, for when the built-in shape is
// close but not exact.
func Spec(p Preset) (*YAMLSpec, error) {
	s, ok := presetSpecs[p]
	if !ok {
		return nil, fmt.Errorf("synth: unknown preset %q", p)
	}
	return YAMLBytes([]byte(s))
}

var _ = yamlfe.Spec{} // keep the frontend import meaningful to readers

var presetSpecs = map[Preset]string{
	PresetUser: `name: users
count: 100
fields:
  id:            { kind: uuid, pk: true }
  full_name:     { kind: name }
  first_name:    { kind: firstname }
  last_name:     { kind: lastname }
  gender:        { kind: gender }
  email:         { kind: email, unique: true }
  username:      { kind: username, unique: true }
  phone:         { kind: phone }
  password_hash: { kind: passwordhash }
  country:       { kind: country }
  city:          { kind: city }
  street:        { kind: street }
  postcode:      { kind: postcode }
  date_of_birth: { kind: time, min: 1960-01-01, max: 2006-12-31 }
  national_id:   { kind: ssn, mask: partial }
  is_active:     { kind: bool }
  created_at:    { kind: time }
  updated_at:    { kind: time }
  last_login_at: { kind: time, blank: 20 }
`,

	PresetPayment: `name: payments
count: 100
fields:
  id:            { kind: uuid, pk: true }
  user_id:       { kind: uuid }
  card_number:   { kind: card, mask: partial }
  card_brand:    { kind: cardbrand }
  card_expiry:   { kind: cardexpiry }
  cvv:           { kind: cvv, mask: redact }
  card_token:    { kind: cardtoken }
  holder_name:   { kind: name }
  amount:        { kind: amount, min: 1, max: 5000 }
  currency:      { kind: currency }
  method:        { kind: paymentmethod }
  status:        { kind: enum, choices: [pending, authorized, captured, failed, refunded], weights: [0.05, 0.1, 0.7, 0.1, 0.05] }
  created_at:    { kind: time }
  captured_at:   { kind: time, blank: 15 }
`,

	PresetTransaction: `name: transactions
count: 100
fields:
  id:              { kind: uuid, pk: true }
  account_id:      { kind: uuid }
  counterparty:    { kind: company }
  card_number:     { kind: card, mask: partial }
  card_brand:      { kind: cardbrand }
  national_id:     { kind: ssn, mask: hash }
  iban:            { kind: iban, mask: partial }
  amount:          { kind: amount, min: 1, max: 10000 }
  currency:        { kind: currency }
  balance_after:   { kind: balance }
  direction:       { kind: enum, choices: [debit, credit], weights: [0.7, 0.3] }
  category:        { kind: enum, choices: [groceries, transport, utilities, salary, transfer, entertainment, health, rent] }
  status:          { kind: enum, choices: [posted, pending, reversed], weights: [0.85, 0.1, 0.05] }
  reference:       { kind: couponcode }
  description:     { kind: lorem }
  merchant_city:   { kind: city }
  merchant_country: { kind: countrycode }
  created_at:      { kind: time }
  posted_at:       { kind: time }
`,

	PresetOrder: `name: orders
count: 100
fields:
  id:            { kind: uuid, pk: true }
  customer_id:   { kind: uuid }
  customer_name: { kind: name }
  email:         { kind: email }
  product:       { kind: product }
  category:      { kind: productcategory }
  sku:           { kind: sku }
  quantity:      { kind: int, min: 1, max: 10 }
  unit_price:    { kind: amount, min: 5, max: 900 }
  total:         { kind: amount, min: 5, max: 9000 }
  currency:      { kind: currency }
  status:        { kind: orderstatus }
  shipping_city: { kind: city }
  shipping_postcode: { kind: postcode }
  created_at:    { kind: time }
  shipped_at:    { kind: time, blank: 30 }
constraints:
  - {kind: ordering, left: created_at, right: shipped_at}
`,

	PresetProduct: `name: products
count: 100
fields:
  id:          { kind: uuid, pk: true }
  name:        { kind: product }
  sku:         { kind: sku, unique: true }
  barcode:     { kind: ean13, unique: true }
  category:    { kind: productcategory }
  material:    { kind: productmaterial }
  brand:       { kind: brand }
  color:       { kind: color }
  size:        { kind: tshirtsize }
  price:       { kind: amount, min: 1, max: 2000 }
  currency:    { kind: currency }
  stock:       { kind: int, min: 0, max: 500 }
  rating:      { kind: rating }
  description: { kind: paragraph }
  is_active:   { kind: bool }
  created_at:  { kind: time }
`,

	PresetEmployee: `name: employees
count: 100
fields:
  id:            { kind: uuid, pk: true }
  full_name:     { kind: name }
  gender:        { kind: gender }
  email:         { kind: email, unique: true }
  phone:         { kind: phone }
  national_id:   { kind: ssn, mask: partial }
  iban:          { kind: iban, mask: partial }
  job_title:     { kind: job }
  department:    { kind: department }
  level:         { kind: joblevel }
  salary:        { kind: salary }
  currency:      { kind: currency }
  city:          { kind: city }
  hired_at:      { kind: time }
  terminated_at: { kind: time, blank: 80 }
  is_active:     { kind: bool }
`,

	PresetPatient: `name: patients
count: 100
fields:
  id:           { kind: uuid, pk: true }
  full_name:    { kind: name }
  gender:       { kind: gender }
  date_of_birth: { kind: time, min: 1940-01-01, max: 2015-12-31 }
  national_id:  { kind: ssn, mask: hash }
  phone:        { kind: phone, mask: partial }
  blood_type:   { kind: bloodtype }
  allergy:      { kind: allergy, blank: 60 }
  diagnosis:    { kind: disease }
  diagnosis_code: { kind: icd10 }
  symptom:      { kind: symptom }
  specialty:    { kind: medicalspecialty }
  medication:   { kind: drugname }
  lab_test:     { kind: labtest }
  admitted_at:  { kind: time }
  discharged_at: { kind: time, blank: 25 }
constraints:
  - {kind: ordering, left: admitted_at, right: discharged_at}
`,

	PresetEvent: `name: events
count: 100
fields:
  id:         { kind: uuid, pk: true }
  user_id:    { kind: uuid }
  session_id: { kind: uuid }
  name:       { kind: enum, choices: [page_view, click, signup, login, logout, purchase, search, share, error] }
  url:        { kind: url }
  referrer:   { kind: url, blank: 40 }
  ip:         { kind: ipv4, mask: partial }
  user_agent: { kind: useragent }
  browser:    { kind: browser }
  os:         { kind: os }
  device:     { kind: device }
  country:    { kind: countrycode }
  city:       { kind: city }
  duration_ms: { kind: int, min: 10, max: 30000 }
  status_code: { kind: httpstatus }
  created_at: { kind: time }
`,
}
