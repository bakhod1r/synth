package synth

import "github.com/bakhod1r/synth/mask"

// Masker anonymizes a real data export, replacing personal data with synthetic
// values of the same format. See the mask package for the guarantees.
type Masker = mask.Masker

// MaskRule binds a column to a masking strategy.
type MaskRule = mask.Rule

// MaskReport summarizes what a masking run changed.
type MaskReport = mask.Report

// Masking strategies.
const (
	MaskKeep   = mask.Keep
	MaskFake   = mask.Fake
	MaskRedact = mask.Redact
	MaskDrop   = mask.Drop
	MaskDP     = mask.DP
)

// NewMasker returns a Masker. The key makes replacements deterministic: mask
// related dumps with the same key so foreign keys still join, or use a fresh
// key to make two runs unlinkable. Columns that look personal are faked by
// default even without an explicit rule.
func NewMasker(key, localeName string) *Masker {
	if localeName == "" {
		localeName = "en_US"
	}
	return mask.New(key, localeName)
}
