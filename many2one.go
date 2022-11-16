package codejen

type ManyToOne[Input any] interface {
	Jenny[Input]

	// Generate takes a slice of Input and generates one File, The zero value of a
	// File may be returned to indicate the jenny was a no-op for the provided
	// Inputs.
	Generate(...Input) (*File, error)
}

type m2oAdapt[OriginalInput, AdaptedInput any] struct {
	fn func(AdaptedInput) OriginalInput
	g  ManyToOne[OriginalInput]
}

func (oa *m2oAdapt[OriginalInput, AdaptedInput]) JennyName() string {
	return oa.g.JennyName()
}

func (oa *m2oAdapt[OriginalInput, AdaptedInput]) Generate(ps ...AdaptedInput) (*File, error) {
	qs := make([]OriginalInput, len(ps))
	for i, p := range ps {
		qs[i] = oa.fn(p)
	}
	return oa.g.Generate(qs...)
}

// AdaptManyToOne takes a ManyToOne jenny that accepts a particular type as input
// (OriginalInput), and transforms it into a jenny that accepts a different type
// as input (AdaptedInput), given a function that can transform an OriginalInput
// to an AdaptedInput.
//
// Use this to make jennies reusable in other Input type contexts.
func AdaptManyToOne[OriginalInput, AdaptedInput any](g ManyToOne[OriginalInput], fn func(AdaptedInput) OriginalInput) ManyToOne[AdaptedInput] {
	return &m2oAdapt[OriginalInput, AdaptedInput]{
		fn: fn,
		g:  g,
	}
}
