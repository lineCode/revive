package lint

import (
	"bufio"
	"bytes"
	"go/token"
)

// ReadFile defines an abstraction for reading files.
type ReadFile func(path string) (result []byte, err error)

type disabledIntervalsMap = map[string][]DisabledInterval

// Linter is used for linting set of files.
type Linter struct {
	reader ReadFile
}

// New creates a new Linter
func New(reader ReadFile) Linter {
	return Linter{reader: reader}
}

var (
	genHdr = []byte("// Code generated ")
	genFtr = []byte(" DO NOT EDIT.")
)

// isGenerated reports whether the source file is generated code
// according the rules from https://golang.org/s/generatedcode.
// This is inherited from the original go lint.
func isGenerated(src []byte) bool {
	sc := bufio.NewScanner(bytes.NewReader(src))
	for sc.Scan() {
		b := sc.Bytes()
		if bytes.HasPrefix(b, genHdr) && bytes.HasSuffix(b, genFtr) && len(b) >= len(genHdr)+len(genFtr) {
			return true
		}
	}
	return false
}

// Lint lints a set of files with the specified rule.
func (l *Linter) Lint(filenames []string, ruleSet []Rule, config Config) (<-chan Failure, error) {
	failures := make(chan Failure)
	pkg := &Package{
		fset:  token.NewFileSet(),
		files: map[string]*File{},
	}
	for _, filename := range filenames {
		content, err := l.reader(filename)
		if err != nil {
			return nil, err
		}
		if isGenerated(content) && !config.IgnoreGeneratedHeader {
			continue
		}

		file, err := NewFile(filename, content, pkg)
		if err != nil {
			return nil, err
		}

		pkg.files[filename] = file
	}

	go (func() {
		pkg.lint(ruleSet, config, failures)
	})()

	return failures, nil
}
