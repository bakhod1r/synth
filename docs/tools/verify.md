# Audit an existing dataset (`synth verify`)

The rules that make Synth's output coherent are the rules worth checking on data
somebody else produced. `synth verify` is the generator run backwards:

```bash
synth verify -i orders.csv --ref user_id=users.csv:id
synth verify -i orders.csv -s orders.yaml -f json   # also re-check mined invariants
```

## What it reports

Failed check digits (Luhn, IBAN mod-97, EAN-13/UPC), unparseable emails, URLs
and IP addresses, dangling foreign keys, timestamp pairs in the wrong order, and
columns that carry no information. Parent tables are read from their own files —
nothing is queried.

## Exit codes

Exit code 1 on any error, 0 when only warnings were found, so it drops into CI
without a wrapper. A degenerate column is a *warning*, not an error: real data
sometimes looks like that, and a tool that cries wolf gets muted.

## Two rules that keep it honest

- A column is audited only when its values already mostly match what its name
  claims, so a column called `card` holding loyalty tiers is skipped rather than
  reported a million times.
- A clean dataset must produce an **empty** report — a check that fires on
  correct data is a false positive and a bug here, not something for you to
  filter out.

## k-anonymity

`verify` also answers whether an anonymized export still singles people out:

```sh
synth verify -i safe.csv --k 5 --qi age,zip,gender   # exit 1 if any group < 5
```

See [Mask](mask.md).
