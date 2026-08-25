# Anonymize a production dump (GDPR)

Hand Synth a real export and get one that is safe to share: personal data is
replaced with synthetic values of the same format, everything else is left
alone.

```go
m := synth.NewMasker("team-key", "en_US")
m.Rule(synth.MaskRule{Column: "notes", Strategy: synth.MaskRedact})
report, _ := m.File("dump.csv", "safe.csv")
```

```sh
synth mask -i prod_export.csv -o safe.csv --key "$MASK_KEY"
```

## Properties

- **Consistent** — the same input value always maps to the same replacement, so
  joins and foreign keys still line up (use the same key across related dumps).
- **Format-preserving** — an email stays an email, a card stays 16 digits.
- **Irreversible** — replacements come from a keyed hash, not an encoding.
- **Thorough** — PII is caught by column name, by value format, and inside free
  text (an email buried in a `notes` field is scrubbed too).

`synth mask` refuses to run without `--key` (an unkeyed run is not reproducible)
and refuses to write over its own input.

## k-anonymity

Two measures go further than replacement. **k-anonymity** checks that no
combination of quasi-identifiers singles anyone out — direct identifiers gone is
not enough if one person is the only 99-year-old in their ZIP:

```sh
synth verify -i safe.csv --k 5 --qi age,zip,gender   # exit 1 if any group < 5
```

## Differential-privacy noise

**Differential-privacy noise** perturbs a numeric column with the Laplace
mechanism, so a released number cannot be pinned to one record:

```sh
synth mask -i dump.csv -o safe.csv --key team-key --dp salary:1.0:10000
```

`salary:1.0:10000` is column, epsilon (smaller = more noise), and sensitivity.

!!! warning "Scope of the DP claim"
    The noise is reproducible under the key — this is input perturbation for
    fixtures, not query-time differential privacy, and it carries no privacy
    budget across releases.
