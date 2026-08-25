# Format-valid values

Every generated identifier passes the check a real system would run on it:

- **Credit cards** — Luhn-valid, issued from real BIN ranges per brand.
- **IBANs** — mod-97 checksum, correct per-country length and BBAN layout.
- **National IDs, VAT numbers, tax IDs** — country-specific check digits.
- **Emails** — RFC 5322 conformant in every locale, including the ones whose
  names are not written in Latin: a mailbox is ASCII unless the whole mail path
  speaks SMTPUTF8, so names are transliterated the way their owners
  transliterate them.
- **URLs, phone numbers** — RFC / E.164 conformant.

That last one matters more than it reads. A generated Uzbek or Japanese name
does not become a legal mailbox by being written in its own script; Synth
transliterates rather than emitting an address most SMTP paths would reject.

## Checking somebody else's data

The same rules run backwards over a dataset you did not generate — failed check
digits, unparseable emails and URLs, dangling foreign keys:

```sh
synth verify -i orders.csv
```

See [Verify](../tools/verify.md).
