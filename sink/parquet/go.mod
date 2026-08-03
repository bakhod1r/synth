// Parquet writer module. It backs the CLI's `-f parquet` output and can be
// imported directly for generating from Go.
module github.com/bakhod1r/synth/sink/parquet

go 1.25

require (
	github.com/bakhod1r/synth v0.0.0
	github.com/parquet-go/parquet-go v0.30.1
)

require (
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.19.1 // indirect
	github.com/parquet-go/bitpack v1.0.0 // indirect
	github.com/parquet-go/jsonlite v1.0.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/twpayne/go-geom v1.6.1 // indirect
	golang.org/x/sys v0.38.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/bakhod1r/synth => ../..
