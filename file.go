package jennywrites

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
)

// File is a single file, intended to be written or compared against
// existing files on disk through an [FS].
//
// jennywrites treats a File with an empty RelativePath as not existing,
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

func (f File) toMapFile() *mapFile {
	return &mapFile{
		Data: f.Data,
		Sys:  f.From,
	}
}

// Exists indicates whether the File should be considered to exist.
func (f File) Exists() bool {
	return f.RelativePath != ""
}

// Files is a set of File objects.
//
// A Files is [Files.Invalid] if it contains a File that does not [File.Exists],
// or if it contains more than one File having the same [File.RelativePath].
//
// These invariants are internally enforced by FS.
type Files []File

func (fsl Files) Validate() error {
	var result *multierror.Error
	paths := make(map[string][][]NamedJenny)
	for _, f := range fsl {
		if !f.Exists() {
			result = multierror.Append(result, fmt.Errorf(`nonexistent File (RelativePath == "") not allowed within Files slice`))
		} else if exist, has := paths[f.RelativePath]; has {
			paths[f.RelativePath] = append(exist, f.From)
		} else {
			paths[f.RelativePath] = [][]NamedJenny{f.From}
		}
	}
	for path, froms := range paths {
		if len(froms) > 1 {
			fstr := make([]string, 0, len(froms))
			for _, from := range froms {
				fstr = append(fstr, "'"+jennystack(from).String()+"'")
			}
			result = multierror.Append(result, fmt.Errorf("multiple files at path %s from jennies: %s", path, strings.Join(fstr, ", ")))
		}
	}
	return result.ErrorOrNil()
}