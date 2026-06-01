package codejen

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// NewFile makes it slightly more ergonomic to create a new File than
// with a raw struct declaration.
func NewFile(path string, data []byte, from ...NamedJenny) *File {
	return &File{
		RelativePath: path,
		Data:         data,
		From:         from,
	}
}

// File is a single file, intended to be written or compared against
// existing files on disk through an [FS].
//
// codejen treats a File with an empty RelativePath as not existing,
// regardless of whether Data is empty. Thus, the zero value of File is
// considered not to exist.
type File struct {
	// The relative path to which the file should be written. An empty
	// RelativePath indicates a File that does not [File.Exists].
	RelativePath string

	// Data is the contents of the file.
	Data []byte

	// From is the stack of jennies responsible for producing this File.
	// Wrapper jennies should precede the jennies they wrap.
	From []NamedJenny
}

// Exists indicates whether the File should be considered to exist.
func (f File) Exists() bool {
	return f.RelativePath != ""
}

// Validate checks that the File is valid - has a relative path, and at least
// one jenny in its From.
func (f File) Validate() error {
	if !f.Exists() {
		return nil
	}

	if filepath.IsAbs(f.RelativePath) {
		return fmt.Errorf("%s: File paths must be relative", f.RelativePath)
	}
	if len(f.From) == 0 {
		return fmt.Errorf("%s: File must have at least one From jenny", f.RelativePath)
	}
	return nil
}

// ToFS turns a single File into a FS containing only
// that file.
//
// An error is only possible if the File does not Validate.
func (f File) ToFS() (*FS, error) {
	wd := NewFS()
	err := wd.add(f)
	if err != nil {
		return nil, err
	}
	return wd, nil
}

// FromString converts the stack of jennies in File.From to a string by
// joining them with a colon.
func (f File) FromString() string {
	strs := make([]string, len(f.From))
	for i, j := range f.From {
		strs[i] = j.JennyName()
	}
	return strings.Join(strs, ":")
}

// Files is a set of File objects.
//
// A Files is [Files.Invalid] if it contains a File that does not [File.Exists],
// or if it contains more than one File having the same [File.RelativePath] (note: this is case-insensitive).
//
// These invariants are internally enforced by FS.
type Files []File

func (fsl Files) Validate() error {
	var errs []error

	paths := make(map[string][][]NamedJenny)
	for _, f := range fsl {
		normalizedPath := strings.ToLower(f.RelativePath)

		if err := f.Validate(); err != nil {
			errs = append(errs, err)
		} else if !f.Exists() {
			errs = append(errs, fmt.Errorf(`nonexistent File (RelativePath == "") not allowed within Files slice`))
		} else if exist, has := paths[normalizedPath]; has {
			paths[normalizedPath] = append(exist, f.From)
		} else {
			paths[normalizedPath] = [][]NamedJenny{f.From}
		}
	}
	for path, froms := range paths {
		if len(froms) > 1 {
			fstr := make([]string, 0, len(froms))
			for _, from := range froms {
				fstr = append(fstr, "'"+jennystack(from).String()+"'")
			}
			errs = append(errs, fmt.Errorf("multiple files at path %s from jennies: %s", path, strings.Join(fstr, ", ")))
		}
	}

	return errors.Join(errs...)
}
