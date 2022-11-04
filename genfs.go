package jennywrites

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/sync/errgroup"
)

// GenFS is a pseudo-filesystem that supports batch-writing its contents
// to the real filesystem, or batch-comparing its contents to the real
// filesystem. Its intended use is for idiomatic `go generate`-style code
// generators, where it is expected that the results of codegen are committed to
// version control.
//
// In such cases, the normal behavior of a generator is to write files to disk,
// but in CI, that behavior should change to verify that what is already on disk
// is identical to the results of code generation. This allows CI to ensure that
// the results of code generation are always up to date. GenFS supports
// these related behaviors through its Write() and Verify() methods, respectively.
//
// Note that the statelessness of GenFS means that, if a particular input
// to the code generator goes away, it will not notice generated files left
// behind if their inputs are removed.
//
// Files may not be removed once [GenFS.Add]ed. If a path conflict occurs
// when adding a new file or merging another GenFS, an error is returned.
// TODO introduce a search/match system
type GenFS struct {
	mapFS
	mu sync.Mutex
}

// ShouldExistErr is an error that indicates a file should exist, but does not.
type ShouldExistErr struct {
}

// ContentsDifferErr is an error that indicates the contents of a file on disk are
// different than those in the GenFS.
type ContentsDifferErr struct {
}

// File represents a single file object within a GenFS.
type File struct {
	// The relative path to which the generated file should be written.
	RelativePath string

	// Contents of the generated file.
	Data []byte

	// From is the stack of Generiters responsible for producing this File.
	From []Generiter
}

// ToFS turns a single File into a GenFS containing only
// that file, given an owner string.
//
// An error is only possible if an absolute path is provided.
func (f *File) ToFS(owner string) (*GenFS, error) {
	wd := New()
	err := wd.add(owner, f)
	if err != nil {
		return nil, err
	}
	return wd, nil
}

type file struct {
	b     []byte
	owner string
}

// New creates a new GenFS, ready for use.
func New() *GenFS {
	return &GenFS{
		mapFS: make(mapFS),
	}
}

type writeSlice []struct {
	path     string
	contents []byte
}

// Verify checks the contents of each file against the filesystem. It emits an error
// if any of its contained files differ.
//
// If the provided prefix path is non-empty, it will be prepended to all file
// entries in the map for writing. prefix may be an absolute path.
func (wd *GenFS) Verify(ctx context.Context, prefix string) error {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	g, _ := errgroup.WithContext(ctx)
	g.SetLimit(12)
	var result *multierror.Error

	for _, it := range wd.toSlice() {
		item := it
		g.Go(func() error {
			ipath := filepath.Join(prefix, item.path)
			if _, err := os.Stat(ipath); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					result = multierror.Append(result, fmt.Errorf("%s: generated file should exist, but does not", ipath))
				} else {
					return fmt.Errorf("%s: could not stat generated file: %w", ipath, err)
				}
				return nil
			}

			ob, err := os.ReadFile(ipath) //nolint:gosec
			if err != nil {
				return fmt.Errorf("%s: error reading file: %w", ipath, err)
			}
			dstr := cmp.Diff(string(ob), string(item.contents))
			if dstr != "" {
				result = multierror.Append(result, fmt.Errorf("%s would have changed:\n\n%s", ipath, dstr))
			}
			return nil
		})
	}
	err := g.Wait()
	if err != nil {
		return fmt.Errorf("io error while verifying tree: %w", err)
	}

	return result.ErrorOrNil()
}

// Write writes all of the files to their indicated paths.
//
// If the provided prefix path is non-empty, it will be prepended to all file
// entries in the map for writing. prefix may be an absolute path.
// TODO try to undo already-written files on error (only best effort, it's impossible to guarantee)
func (wd *GenFS) Write(ctx context.Context, prefix string) error {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	g, _ := errgroup.WithContext(ctx)
	g.SetLimit(12)

	for _, item := range wd.toSlice() {
		it := item
		g.Go(func() error {
			path := filepath.Join(prefix, it.path)
			err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
			if err != nil {
				return fmt.Errorf("%s: failed to ensure parent directory exists: %w", path, err)
			}

			if err := os.WriteFile(path, it.contents, 0644); err != nil {
				return fmt.Errorf("%s: error while writing file: %w", path, err)
			}
			return nil
		})
	}

	return g.Wait()
}

func (wd *GenFS) toSlice() writeSlice {
	sl := make(writeSlice, 0, len(wd.mapFS))
	type ws struct {
		path     string
		contents []byte
	}

	for k, v := range wd.mapFS {
		sl = append(sl, ws{
			path:     k,
			contents: v.Data,
		})
	}

	sort.Slice(sl, func(i, j int) bool {
		return sl[i].path < sl[j].path
	})

	return sl
}

// Add adds one or more files to the GenFS. An error is returned if any of
// the provided files would conflict a file already declared added to the
// GenFS.
func (wd *GenFS) Add(owner string, flist ...*File) error {
	wd.mu.Lock()
	err := wd.add(owner, flist...)
	wd.mu.Unlock()
	return err
}

func (wd *GenFS) add(owner string, flist ...*File) error {
	var result *multierror.Error
	for _, f := range flist {
		if rf, has := wd.mapFS[f.RelativePath]; has {
			result = multierror.Append(result, fmt.Errorf("GenFS cannot create %s for %q, already created for %q", f.RelativePath, owner, rf.owner))
		}
		if filepath.IsAbs(f.RelativePath) {
			result = multierror.Append(result, fmt.Errorf("files added to GenFS must have relative paths, got %s from %q", f.RelativePath, owner))
		}
	}
	if result.ErrorOrNil() != nil {
		return result
	}

	for _, f := range flist {
		wd.mapFS[f.RelativePath] = &mapFile{Data: f.Data, Sys: owner}
	}
	return nil
}

// Merge combines all the entries from the provided GenFS into the callee
// GenFS. Duplicate paths result in an error.
func (wd *GenFS) Merge(wd2 *GenFS) error {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	var result *multierror.Error

	for k, inf := range wd2.mapFS {
		result = multierror.Append(result, wd.add(inf.Sys.(string), &File{RelativePath: k, Data: inf.Data}))
	}

	return result.ErrorOrNil()
}
