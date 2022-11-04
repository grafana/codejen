package jennywrites

import (
	"fmt"
	"reflect"
	"sync"
)

// A Jenny is a jennywrites code generator.
//
// Each Jenny works with exactly one type of input to its code generation, as
// indicated by type parameter. jennywrites follows a naming convention of
// naming these type parameters "Input" as an indicator for humans that a
// particular type parameter is used in this way.
//
// Each Jenny takes either one or many Inputs, and produces zero, one, or many
// output files. It is a design tenet of jennywrites that good separation of
// concerns in code generation can be achieved by
//
// Unfortunately, Go's generic system does not (yet?) allow expression of the
// necessary abstraction over individual kinds of Jennies as part of the Jenny
// interface itself.
type Jenny[Input any] interface {
	// JennyName returns the name of the generator.
	JennyName() string

	//
	// OneToOne[Input] | ManyToOne[Input any] | OneToMany[Input] | ManyToMany[Input any]
}

// NamedJenny makes just the
type NamedJenny interface {
	JennyName() string
}

type JennySet[Input any] struct {
	mut sync.RWMutex
	// TODO add index tracking so that add-order can be preserved, if that becomes important

	onegens  []OneToOne[Input]
	manygens []ManyToOne[Input]
	post     []func(f File) (File, error)
}

func (gs *JennySet[Input]) Generiter() string {
	return fmt.Sprintf("JennySet[%s]", reflect.TypeOf(new(Input)).Elem().Name())
}

func (gs *JennySet[Input]) Generate(objs []Input) (*GenFS, error) {
	gfs := New()
	for _, og := range gs.onegens {
		for _, obj := range objs {
			ggfs, err := og.Generate(obj)
			if err != nil {
				return nil, err
			}
			err = gfs.Merge(ggfs)
			if err != nil {
				return nil, err
			}
		}
	}

	for _, og := range gs.manygens {
		ggfs, err := og.Generate(objs)
		if err != nil {
			return nil, err
		}
		err = gfs.Merge(ggfs)
		if err != nil {
			return nil, err
		}
	}

	return gfs, nil
}

func (gs *JennySet[T]) AddOnes(ones ...OneToOne[T]) {
	gs.mut.Lock()
	gs.onegens = append(gs.onegens, ones...)
	gs.mut.Unlock()
}

func (gs *JennySet[T]) AddManys(manys ...ManyToOne[T]) {
	gs.mut.Lock()
	gs.manygens = append(gs.manygens, manys...)
	gs.mut.Unlock()
}
