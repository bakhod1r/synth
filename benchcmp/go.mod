// Separate module so the comparison benchmarks' dependencies (go-faker,
// jaswdr) never enter the main library's dependency graph.
module github.com/bakhod1r/synth/benchcmp

go 1.26.2

require (
	github.com/bakhod1r/synth v0.0.0
	github.com/go-faker/faker/v4 v4.10.0
	github.com/jaswdr/faker/v2 v2.9.1
)

require (
	github.com/bakhod1r/devicex v0.2.1 // indirect
	github.com/bakhod1r/emailx v0.4.0 // indirect
	github.com/bakhod1r/phonex v0.1.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/bakhod1r/synth => ..
