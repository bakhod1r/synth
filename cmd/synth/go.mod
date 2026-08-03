// The CLI is its own module so its output-format dependencies — the Parquet
// writer and the zstd/gzip compressor — stay out of the core library's graph.
// The library itself needs only google/uuid and yaml.v3; installing the binary
// pulls the heavier set, but importing the library never does.
module github.com/bakhod1r/synth/cmd/synth

go 1.25

require (
	github.com/bakhod1r/synth v0.0.0
	github.com/bakhod1r/synth/sink/parquet v0.0.0
	github.com/klauspost/compress v1.19.1
)

require (
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/parquet-go/bitpack v1.0.0 // indirect
	github.com/parquet-go/jsonlite v1.0.0 // indirect
	github.com/parquet-go/parquet-go v0.30.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/twpayne/go-geom v1.6.1 // indirect
	golang.org/x/sys v0.38.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/bakhod1r/synth => ../..

replace github.com/bakhod1r/synth/sink/parquet => ../../sink/parquet
