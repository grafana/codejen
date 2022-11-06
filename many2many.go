package jennywrites

// ManyToMany is a Jenny that accepts many inputs, and produces 0 to N files as output.
type ManyToMany[Input any] interface {
	Jenny[Input]

	// Generate takes a slice of Input and generates many [File]s, or none (nil) if the j
	// was a no-op for the provided Input.
	//
	// A nil, nil return is used to indicate the generator had nothing to do for the
	// provided Input.
	Generate([]Input) (Files, error)
}

type m2mAdapt[OriginalInput, AdaptedInput any] struct {
	fn func(AdaptedInput) OriginalInput
	j  ManyToMany[OriginalInput]
}

func (oa *m2mAdapt[OriginalInput, AdaptedInput]) JennyName() string {
	return oa.j.JennyName()
}

func (oa *m2mAdapt[OriginalInput, AdaptedInput]) Generate(ps []AdaptedInput) (Files, error) {
	qs := make([]OriginalInput, len(ps))
	for i, p := range ps {
		qs[i] = oa.fn(p)
	}
	return oa.j.Generate(qs)
}

// AdaptManyToMany takes a ManyToMany jenny that accepts a particular type as input
// (OriginalInput), and transforms it into a jenny that accepts a different type
// as input (AdaptedInput), given a function that can transform an OriginalInput
// to an AdaptedInput.
//
// Use this to make jennies reusable in other Input type contexts.
func AdaptManyToMany[OriginalInput, AdaptedInput any](j ManyToMany[OriginalInput], fn func(AdaptedInput) OriginalInput) ManyToMany[AdaptedInput] {
	return &m2mAdapt[OriginalInput, AdaptedInput]{
		fn: fn,
		j:  j,
	}
}
