module github.com/bakhod1r/synth

go 1.26.2

require github.com/google/uuid v1.6.0

require (
	github.com/bakhod1r/devicex v0.2.1
	github.com/bakhod1r/phonex v0.1.0
	gopkg.in/yaml.v3 v3.0.1
)

require github.com/bakhod1r/emailx v0.4.0

// v1.4.2 does not compile: a test function was declared twice in the root
// package. Fixed in v1.4.3.
retract v1.4.2
