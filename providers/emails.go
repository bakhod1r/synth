package providers

import (
	"strings"
	"sync"

	"github.com/bakhod1r/emailx"
	"github.com/bakhod1r/synth/schema"
)

// Mail-domain fields, answered from the emailx catalogues rather than from a
// list kept here: the domains throwaway mail services run, and the domains the
// public providers own.
//
// email_provider and email_normalized read an address through from=, so they
// describe the record's own address rather than a second one:
//
//	type Account struct {
//	    Email    string `synth:"email"`
//	    Provider string `synth:"email_provider,from=Email"`
//	    Canonical string `synth:"email_normalized,from=Email"`
//	}
//	// bekzat.muqanov@gmail.com  gmail  bekzatmuqanov@gmail.com

func init() {
	registry[schema.KindEmailDisposable] = disposableEmail
	registry[schema.KindEmailProvider] = emailProvider
	registry[schema.KindEmailNormalized] = normalizedEmail
}

// disposableDomains is the throwaway-service list flattened once, because
// generating a record needs to pick by index. It is built on first use, so a
// program that never asks for one never pays for it.
var disposableDomains = sync.OnceValue(emailx.DisposableDomains)

// disposableEmail builds an address at a throwaway service. The local part
// comes from the same shapes an ordinary address uses — a disposable mailbox is
// distinguished by its domain, not by looking odd.
func disposableEmail(c Ctx) any {
	domains := disposableDomains()
	if len(domains) == 0 {
		return email(c)
	}
	addr, _ := email(c).(string)
	local, _, ok := strings.Cut(addr, "@")
	if !ok || local == "" {
		return addr
	}
	return local + "@" + domains[c.Rand.Intn(len(domains))]
}

// emailProvider names the mail service an address belongs to. It reports an
// empty string for a domain no public provider owns — a company's own mail
// server, a throwaway service — because naming one would be a guess.
func emailProvider(c Ctx) any {
	if addr, _ := c.Sibling("__from__").(string); addr != "" {
		e, err := emailx.Parse(addr)
		if err != nil {
			return ""
		}
		return e.ProviderID()
	}
	// With no address to read, draw a provider and let the field stand alone.
	providers := emailx.Providers()
	if len(providers) == 0 {
		return ""
	}
	return providers[c.Rand.Intn(len(providers))].ID
}

// normalizedEmail is the address in the canonical form the provider itself
// treats as one mailbox: lower-cased, plus-tags and gmail's dots removed. It is
// what a deduplicating join wants on both sides.
func normalizedEmail(c Ctx) any {
	addr, _ := c.Sibling("__from__").(string)
	if addr == "" {
		addr, _ = email(c).(string)
	}
	n, err := emailx.Normalize(addr)
	if err != nil {
		// Not an address emailx will take. Inventing a canonical form for it
		// would be a claim about a mailbox that may not exist, so the input
		// stands as it is.
		return addr
	}
	return n
}
