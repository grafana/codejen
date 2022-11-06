package jennywrites

type OneToMany[Input any] interface {
	Jenny[Input]

	// Generate takes an Input and generates many [File]s, or none (nil) if the j
	// was a no-op for the provided Input.
	Generate(Input) (Files, error)
}

type o2mAdapt[OriginalInput, AdaptedInput any] struct {
	fn func(AdaptedInput) OriginalInput
	j  OneToMany[OriginalInput]
}

func (oa *o2mAdapt[OriginalInput, AdaptedInput]) JennyName() string {
	return oa.j.JennyName()
}

func (oa *o2mAdapt[OriginalInput, AdaptedInput]) Generate(t AdaptedInput) (Files, error) {
	return oa.j.Generate(oa.fn(t))
}

// AdaptOneToMany takes a OneToMany jenny that accepts a particular type as input
// (OriginalInput), and transforms it into a jenny that accepts a different type
// as input (AdaptedInput), given a function that can transform an OriginalInput
// to an AdaptedInput.
//
// Use this to make jennies reusable in other Input type contexts.
func AdaptOneToMany[OriginalInput, AdaptedInput any](j OneToMany[OriginalInput], fn func(AdaptedInput) OriginalInput) OneToMany[AdaptedInput] {
	return &o2mAdapt[OriginalInput, AdaptedInput]{
		fn: fn,
		j:  j,
	}
}
