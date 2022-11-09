package codejen

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/hashicorp/go-multierror"
)

type jnode struct {
	next *jnode
	j    NamedJenny
}

// JennyListWithNamer creates a new JennyList that decorates errors using the
// provided namer func, which can derive a meaningful identifier string from the
// Input type for the JennyList.
func JennyListWithNamer[Input any](namer func(t Input) string) *JennyList[Input] {
	return &JennyList[Input]{
		inputnamer: namer,
	}
}

// JennyList is an ordered collection of jennies. JennyList itself implements
// [ManyToMany], and when called, will construct an [FS] by calling each of its
// contained jennies in order.
//
// The primary purpose of JennyList is to make it easy to create complex,
// case-specific code generators by composing sets of small, reusable jennies
// that each have clear, narrow responsibilities.
//
// The File outputs of all member jennies in a JennyList exist in the same
// relative path namespace. JennyList does not modify emitted paths. Path
// uniqueness (per [Files.Validate]) is internally enforced across the aggregate
// set of Files.
//
// JennyList's Input type parameter is used to enforce that every Jenny in the
// JennyList takes the same type parameter.
type JennyList[Input any] struct {
	mut sync.RWMutex

	// entrypoint to the singly linked list of jennies
	first *jnode

	// postprocessors, to be run on every file returned from each contained jenny
	post []FileMapper

	// inputnamer, if non-nil, gives a name to an input.
	inputnamer func(t Input) string
}

func (js *JennyList[Input]) last() *jnode {
	j := js.first
	for j != nil && j.next != nil {
		j = j.next
	}
	return j
}

func (js *JennyList[Input]) JennyName() string {
	return fmt.Sprintf("JennyList[%s]", reflect.TypeOf(new(Input)).Elem().Name())
}

func (js *JennyList[Input]) wrapinerr(in Input, err error) error {
	if err == nil {
		return nil
	}
	if js.inputnamer == nil {
		return err
	}
	return fmt.Errorf("%w for input %q", err, js.inputnamer(in))
}

func (js *JennyList[Input]) GenerateFS(objs []Input) (*FS, error) {
	js.mut.RLock()
	defer js.mut.RUnlock()

	if js.first == nil {
		return nil, nil
	}

	jfs := NewFS()

	manyout := func(j Jenny[Input], fl Files, err error) error {
		if err != nil {
			return fmt.Errorf("%s: %w", j.JennyName(), err)
		}

		if err = fl.Validate(); err != nil {
			// This is unreachable in the case where there was a single File output, so plural is fine
			return fmt.Errorf("%s returned invalid Files: %w", j.JennyName(), err)
		}

		// postprocessing
		for i, f := range fl {
			for _, post := range js.post {
				of, err := post(f)
				if err != nil {
					return fmt.Errorf("postprocessing of %s from %s failed: %w", f.RelativePath, jennystack(f.From), err)
				}
				f = of
			}
			fl[i] = f
		}
		return jfs.addValidated(fl...)
	}
	oneout := func(j Jenny[Input], f *File, err error) error {
		// err will be handled in manyout
		var fl Files
		if f != nil && f.Exists() {
			fl = Files{*f}
		}
		if err == nil && len(fl) == 0 {
			return nil
		}
		return manyout(j, fl, err)
	}

	result := new(multierror.Error)
	jn := js.first
	for jn != nil {
		var handlerr error
		switch jenny := jn.j.(type) {
		case OneToOne[Input]:
			for _, obj := range objs {
				f, err := jenny.Generate(obj)
				if procerr := js.wrapinerr(obj, oneout(jenny, f, err)); procerr != nil {
					result = multierror.Append(result, procerr)
				}
			}
		case OneToMany[Input]:
			for _, obj := range objs {
				fl, err := jenny.Generate(obj)
				if procerr := js.wrapinerr(obj, manyout(jenny, fl, err)); procerr != nil {
					result = multierror.Append(result, procerr)
				}
			}
		case ManyToOne[Input]:
			f, err := jenny.Generate(objs)
			handlerr = oneout(jenny, f, err)
		case ManyToMany[Input]:
			fl, err := jenny.Generate(objs)
			handlerr = manyout(jenny, fl, err)
		default:
			panic("unreachable")
		}

		if handlerr != nil {
			result = multierror.Append(result, handlerr)
		}
		jn = jn.next
	}

	if result.ErrorOrNil() != nil {
		return nil, multierror.Flatten(result)
	}

	return jfs, nil
}

func (js *JennyList[Input]) Generate(objs []Input) (Files, error) {
	jfs, err := js.GenerateFS(objs)
	if err != nil {
		return nil, err
	}
	return jfs.AsFiles(), nil
}

func (js *JennyList[Input]) append(n ...*jnode) {
	js.mut.Lock()
	last := js.last()
	if last == nil {
		js.first = n[0]
		n = n[1:]
		last = js.first
	}
	for _, jn := range n {
		last.next = jn
		last = last.next
	}
	js.mut.Unlock()
}

func tojnode[J NamedJenny](jennies ...J) []*jnode {
	nlist := make([]*jnode, len(jennies))
	for i, j := range jennies {
		nlist[i] = &jnode{
			j: j,
		}
	}
	return nlist
}

// Append adds Jennies to the end of the JennyList. In Generate, Jennies are
// called in the order they were appended.
//
// All provided jennies must also implement one of [OneToOne], [OneToMany],
// [ManyToOne], [ManyToMany], or this method will panic. For proper type safety,
// use the Append* methods.
func (js *JennyList[Input]) Append(jennies ...Jenny[Input]) {
	nlist := make([]*jnode, len(jennies))
	for i, j := range jennies {
		switch j.(type) {
		case OneToOne[Input], OneToMany[Input], ManyToOne[Input], ManyToMany[Input]:
			nlist[i] = &jnode{
				j: j,
			}
		default:
			panic(fmt.Sprintf("%T is not a valid Jenny, must implement (OneToOne | OneToMany | ManyToOne | ManyToMany)", j))
		}
	}
	js.append(nlist...)
}

// AppendOneToOne is like [JennyList.Append], but typesafe for OneToOne jennies.
func (js *JennyList[Input]) AppendOneToOne(jennies ...OneToOne[Input]) {
	js.append(tojnode(jennies...)...)
}

// AppendManyToOne is like [JennyList.Append], but typesafe for ManyToOne jennies.
func (js *JennyList[Input]) AppendManyToOne(jennies ...ManyToOne[Input]) {
	js.append(tojnode(jennies...)...)
}

// AppendOneToMany is like [JennyList.Append], but typesafe for OneToMany jennies.
func (js *JennyList[Input]) AppendOneToMany(jennies ...OneToMany[Input]) {
	js.append(tojnode(jennies...)...)
}

// AppendManyToMany is like [JennyList.Append], but typesafe for ManyToMany jennies.
func (js *JennyList[Input]) AppendManyToMany(jennies ...ManyToMany[Input]) {
	js.append(tojnode(jennies...)...)
}

// AddPostprocessors appends a slice of FileMapper to its internal list of
// postprocessors.
//
// Postprocessors are run (FIFO) on every File produced by the JennyList.
func (js *JennyList[Input]) AddPostprocessors(fn ...FileMapper) {
	js.mut.Lock()
	js.post = append(js.post, fn...)
	js.mut.Unlock()
}
