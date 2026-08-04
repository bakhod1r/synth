module github.com/bakhod1r/synth

go 1.25

require github.com/google/uuid v1.6.0

require gopkg.in/yaml.v3 v3.0.1

// v1.4.2 does not compile: a test function was declared twice in the root
// package. Fixed in v1.4.3.
retract v1.4.2
