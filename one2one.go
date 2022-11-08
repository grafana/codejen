package codejen

type OneToOne[Input any] interface {
	Jenny[Input]

	// Generate takes an Input and generates one [File]. The zero value of a File
	// may be returned to indicate the jenny was a no-op for the provided Input.
	Generate(Input) (*File, error)
}

type o2oAdapt[OriginalInput, AdaptedInput any] struct {
	fn func(AdaptedInput) OriginalInput
	j  OneToOne[OriginalInput]
}

func (oa *o2oAdapt[OriginalInput, AdaptedInput]) JennyName() string {
	return oa.j.JennyName()
}

func (oa *o2oAdapt[OriginalInput, AdaptedInput]) Generate(t AdaptedInput) (*File, error) {
	return oa.j.Generate(oa.fn(t))
}

// AdaptOneToOne takes a OneToOne jenny that accepts a particular type as input
// (OriginalInput), and transforms it into a jenny that accepts a different type
// as input (AdaptedInput), given a function that can transform an OriginalInput
// to an AdaptedInput.
//
// Use this to make jennies reusable in other Input type contexts.
func AdaptOneToOne[OriginalInput, AdaptedInput any](j OneToOne[OriginalInput], fn func(AdaptedInput) OriginalInput) OneToOne[AdaptedInput] {
	return &o2oAdapt[OriginalInput, AdaptedInput]{
		fn: fn,
		j:  j,
	}
}

// MapOneToOne takes a OneToOne jenny and wraps it in a stack of FileMappers to create a
// new OneToOne jenny. When Generate is called, the output of the OneToOne jenny will be
// transformed
// func MapOneToOne[Input any](j OneToOne[Input], fn ...FileMapper) OneToOne[Input] {
//
// }
