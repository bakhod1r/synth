package synth

import (
	"time"

	"github.com/bakhodir/synth/gen"
	"github.com/bakhodir/synth/internal/rng"
	"github.com/bakhodir/synth/protofe"
	"github.com/bakhodir/synth/schema"
)

// ProtoMessage is a message parsed from a .proto file, ready to generate rows.
type ProtoMessage struct {
	msg *protofe.Message
}

// LoadProto parses .proto source from a file. Synth reads it as text — no
// protoc, no code generation, no schema registry.
func LoadProto(path string) ([]*ProtoMessage, error) {
	ms, err := protofe.Load(path)
	if err != nil {
		return nil, err
	}
	return wrapProto(ms), nil
}

// ProtoBytes parses .proto source from memory.
func ProtoBytes(src []byte) ([]*ProtoMessage, error) {
	ms, err := protofe.Parse(string(src))
	if err != nil {
		return nil, err
	}
	return wrapProto(ms), nil
}

func wrapProto(ms []*protofe.Message) []*ProtoMessage {
	out := make([]*ProtoMessage, len(ms))
	for i, m := range ms {
		out[i] = &ProtoMessage{msg: m}
	}
	return out
}

// Name returns the message name.
func (p *ProtoMessage) Name() string { return p.msg.Name }

// Columns returns field names in declaration order.
func (p *ProtoMessage) Columns() []string { return p.msg.Order }

// Generate produces n records matching the message.
func (p *ProtoMessage) Generate(n int, opts ...Option) ([]map[string]any, error) {
	cfg := config{seed: uint64(time.Now().UnixNano()), locale: "en_US"}
	for _, o := range opts {
		o(&cfg)
	}
	s := &schema.Schema{Fields: append([]schema.Field(nil), p.msg.Schema.Fields...)}
	applyWeighted(s, cfg.weighted)
	eng, err := gen.Compile(s, cfg.locale)
	if err != nil {
		return nil, err
	}
	eng.Chaos = cfg.chaos
	base := rng.New(cfg.seed)
	out := make([]map[string]any, n)
	for i := 0; i < n; i++ {
		out[i] = eng.Record(base, i)
	}
	return out, nil
}
