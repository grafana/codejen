package jennywrites

import (
	"fmt"
	"reflect"
	"sync"
)

// as His Holiness Hightower said:
//
// i think this slide puts it well, even if unintentionally - "we got all this sorted. But...fuck, now it's observability time."

// Generiter is an interface shared by all generite generators, requiring
// them to report their name.
type Generiter interface {
	// GeneriterName returns the name of the generator.
	GeneriterName() string
}

type FromOne[T any] interface {
	Generiter

	// Generate takes a T and generates zero to n files, returning them
	// within a GenFS. A nil, nil return indicates the generator had nothing to do
	// for the provided T.
	Generate(T) (*GenFS, error)
}

type FromMany[T any] interface {
	Generiter

	// Generate takes a slice of T and generates zero to n files, returning them
	// within a GenFS. A nil, nil return indicates the generator had nothing to do
	// for the provided T.
	Generate([]T) (*GenFS, error)
}

type GeneratorSet[T any] struct {
	mut sync.RWMutex
	// TODO add index tracking so that add-order can be preserved, if that becomes important

	onegens  []FromOne[T]
	manygens []FromMany[T]
	post     []func(f File) (File, error)
}

func (gs *GeneratorSet[T]) Generiter() string {
	return fmt.Sprintf("GeneratorSet[%s]", reflect.TypeOf(new(T)).Elem().Name())
}

func (gs *GeneratorSet[T]) Generate(objs []T) (*GenFS, error) {
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

func (gs *GeneratorSet[T]) AddOnes(ones ...FromOne[T]) {
	gs.mut.Lock()
	gs.onegens = append(gs.onegens, ones...)
	gs.mut.Unlock()
}

func (gs *GeneratorSet[T]) AddManys(manys ...FromMany[T]) {
	gs.mut.Lock()
	gs.manygens = append(gs.manygens, manys...)
	gs.mut.Unlock()
}

type oneadapt[P, Q any] struct {
	fn func(P) Q
	g  FromOne[Q]
}

func (oa *oneadapt[P, Q]) GeneriterName() string {
	return oa.g.GeneriterName()
}

func (oa *oneadapt[P, Q]) Generate(t P) (*GenFS, error) {
	return oa.g.Generate(oa.fn(t))
}

func AdaptOne[P, Q any](g FromOne[Q], fn func(P) Q) FromOne[P] {
	return &oneadapt[P, Q]{
		fn: fn,
		g:  g,
	}
}

type manyadapt[P, Q any] struct {
	fn func(P) Q
	g  FromMany[Q]
}

func (oa *manyadapt[P, Q]) GeneriterName() string {
	return oa.g.GeneriterName()
}

func (oa *manyadapt[P, Q]) Generate(ps []P) (*GenFS, error) {
	qs := make([]Q, len(ps))
	for i, p := range ps {
		qs[i] = oa.fn(p)
	}
	return oa.g.Generate(qs)
}

func AdaptMany[P, Q any](g FromMany[Q], fn func(P) Q) FromMany[P] {
	return &manyadapt[P, Q]{
		fn: fn,
		g:  g,
	}
}
