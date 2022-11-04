package jennywrites

type OneToOne[Input any] interface {
	Jenny[Input]

	// Generate takes an Input and generates one [File], or none (nil) if the j
	// was a no-op for the provided Input.
	Generate(Input) (*GenFS, error)
}

type o2oAdapt[AdaptedInput, OriginalInput any] struct {
	fn func(AdaptedInput) OriginalInput
	j  OneToOne[OriginalInput]
}

func (oa *o2oAdapt[AdaptedInput, OriginalInput]) JennyName() string {
	return oa.j.JennyName()
}

func (oa *o2oAdapt[AdaptedInput, OriginalInput]) Generate(t AdaptedInput) (*GenFS, error) {
	return oa.j.Generate(oa.fn(t))
}

// AdaptOneToOne takes a OneToOne jenny that accepts a particular type as input
// (OriginalInput), and transforms it into a jenny that accepts a different type
// as input (AdaptedInput), given a function that can transform an OriginalInput
// to an AdaptedInput.
//
// Use this to make jennies reusable in other Input type contexts.
func AdaptOneToOne[AdaptedInput, OriginalInput any](j OneToOne[OriginalInput], fn func(AdaptedInput) OriginalInput) OneToOne[AdaptedInput] {
	return &o2oAdapt[AdaptedInput, OriginalInput]{
		fn: fn,
		j:  j,
	}
}
