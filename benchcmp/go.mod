// Separate module so the comparison benchmarks' dependencies (go-faker,
// jaswdr) never enter the main library's dependency graph.
module github.com/bakhodir/synth/benchcmp

go 1.26.0

require (
	github.com/bakhodir/synth v0.0.0
	github.com/go-faker/faker/v4 v4.10.0
	github.com/jaswdr/faker/v2 v2.9.1
)

require (
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/bakhodir/synth => ..
