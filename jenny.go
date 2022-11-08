package codejen

// A Jenny is a single codejen code generator.
//
// Each Jenny works with exactly one type of input to its code generation, as
// indicated by type parameter. codejen follows a naming convention of
// naming these type parameters "Input" as an indicator for humans that a
// particular type parameter is used in this way.
//
// Each Jenny takes either one or many Inputs, and produces one or many
// output files. Jennies may also return nils to indicate zero outputs.
//
// It is a design tenet of codejen that, in code generation, good separation
// of concerns starts with keeping a single file to a single responsibility. Thus,
// where possible, most Jennies should aim for one input to one output.
//
// Unfortunately, Go's generic system does not (yet?) allow expression of the
// necessary abstraction over individual kinds of Jennies as part of the Jenny
// interface itself. As such, the actual, functional interface is split into four:
//
//   - [OneToOne]: one Input in, one [File] out
//   - [OneToMany]: one Input in, many [File]s out
//   - [ManyToOne]: many Inputs in, one [File] out
//   - [ManyToMany]: many Inputs in, many [File]s out
//
// All jennies will follow exactly one of these four interfaces.
type Jenny[Input any] interface {
	// JennyName returns the name of the generator.
	JennyName() string

	// if only the type system let us do something like this, the API surface of
	// this library would shrink to a quarter its current size. so much more crisp
	// OneToOne[Input] | ManyToOne[Input any] | OneToMany[Input] | ManyToMany[Input]
}

// NamedJenny includes just the JennyName method. We have to have this interface
// due to the limits on Go's type system.
type NamedJenny interface {
	JennyName() string
}

// This library was originally written with the type Jinspiration used as the
// type for Input type parameters. As in, `type Jenny[Input Jinspiration]`.
//
// It's preserved here because you, dear reader of source code, deserve to
// giggle today.
//
// type Jinspiration any
