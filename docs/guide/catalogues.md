# Catalogue-backed types

The built-in `phone` has E.164 shape; the built-in `device` picks from a short
hand-written list. Where that is not enough, these types answer from published
reference data instead — Google's libphonenumber ranges, and a catalogue of
25,000 Android handsets. No import, no registration:

```go
type Session struct {
    Phone    string `synth:"phone_e164"`
    Display  string `synth:"phone_national,from=Phone"`
    LineType string `synth:"phone_type,from=Phone"`

    Code  string `synth:"device_code"`
    Brand string `synth:"device_brand,from=Code"`
    Model string `synth:"device_name,from=Code"`
}
// +998912341024  91 234 10 24  mobile   SM-T500  Samsung  Galaxy Tab A7
```

## Phones

Valid under the region's numbering plan, carrying the area code of the record's
own city where the plan allows it: a Houston row gets `+1 713`, an Andijon row
`+998 94`, a Roma row `+39 06`.

Types: `phone_e164`, `phone_national`, `phone_international`, `phone_type`.

## Devices

Model codes as they appear in a User-Agent, with the brand and handset name they
actually belong to.

Types: `device_code`, `device_brand`, `device_name`.

## Mail domains

The provider that owns an address, its canonical form, and addresses at the
8,000-odd throwaway services.

Types: `email_provider`, `email_normalized`, `email_disposable`.

## One draw, many columns

In each group, one field draws and the others read it through `from=`, so the
record describes one phone and one handset rather than several. Drop the `from=`
and the columns describe unrelated things that happen to sit in the same row.

!!! warning "Generated numbers may be somebody's"
    A number valid under a numbering plan may well be in use. Generated numbers
    are for fixtures and demos — never dial or message them.
