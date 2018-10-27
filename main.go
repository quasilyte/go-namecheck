package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"io/ioutil"
	"log"
	"regexp"
	"strings"

	"golang.org/x/tools/go/packages"
)

func main() {
	log.SetFlags(0)

	rulesFilename := flag.String("rules", "",
		`JSON file with naming convention rules`)
	verbose := flag.Bool("v", false,
		`turn on additional info message printing`)
	debug := flag.Bool("debug", false,
		`turn on detailed program execution info printing`)

	flag.Parse()

	targets := flag.Args()

	if *rulesFilename == "" {
		log.Fatalf("the -rules argument can't be empty")
	}
	if len(targets) == 0 {
		log.Fatalf("not enought positional args (empty targets list)")
	}

	ctxt := &context{
		checkers: parseRules(*rulesFilename),
		fset:     token.NewFileSet(),
		verbose:  *verbose,
		debug:    *debug,
	}

	cfg := &packages.Config{
		Mode:  packages.LoadSyntax,
		Tests: true,
		Fset:  ctxt.fset,
	}
	pkgs, err := packages.Load(cfg, targets...)
	if err != nil {
		log.Fatalf("load targets: %v", err)
	}

	// First pkgs traversal selects external tests and
	// packages built for testing.
	// If there is no tests for the package,
	// we're going to check them during the second traversal
	// which visits normal package if only it was
	// not checked during the first traversal.
	withTests := map[string]bool{}
	for _, pkg := range pkgs {
		if !strings.Contains(pkg.ID, ".test]") {
			continue
		}
		ctxt.checkPackage(pkg)
		withTests[pkg.PkgPath] = true
	}
	for _, pkg := range pkgs {
		if strings.HasSuffix(pkg.PkgPath, ".test") {
			continue
		}
		if pkg.ID != pkg.PkgPath {
			continue
		}
		if !withTests[pkg.PkgPath] {
			ctxt.checkPackage(pkg)
		}
	}
}

type context struct {
	checkers []*nameChecker
	fset     *token.FileSet

	verbose bool
	debug   bool
}

func (ctxt *context) checkPackage(pkg *packages.Package) {
	ctxt.infoPrintf("check %s", pkg.ID)

	emptyMatchers := &nameMatcherList{}

	matchersCache := map[string]*nameMatcherList{}
	w := walker{pkg: pkg}
	for _, f := range pkg.Syntax {
		w.visit = func(id *ast.Ident) {
			typ := removePointers(pkg.TypesInfo.TypeOf(id))
			typeString := types.TypeString(typ, types.RelativeTo(pkg.Types))
			matchers, ok := matchersCache[typeString]
			switch {
			case ok && matchers == emptyMatchers:
				ctxt.debugPrintf("%s: cache hit (non-interesting)", typeString)
			case ok:
				ctxt.debugPrintf("%s: cache hit", typeString)
			default:
				ctxt.debugPrintf("%s: checkers full scan", typeString)
				for _, c := range ctxt.checkers {
					if c.typeRE.MatchString(typeString) {
						matchersCache[typeString] = c.matchers
						matchers = c.matchers
						break
					}
				}
				if matchers == nil {
					ctxt.debugPrintf("%s: mark as non-interesting", typeString)
					matchersCache[typeString] = emptyMatchers
					return
				}
			}

			for _, m := range matchers.list {
				if !m.Match(id.Name) {
					continue
				}
				fmt.Printf("%s: %s %s: %s\n",
					ctxt.fset.Position(id.Pos()),
					id.Name,
					typeString,
					m.Warning())
				break
			}
		}
		w.walkNames(f)
	}
}

func (ctxt *context) debugPrintf(format string, args ...interface{}) {
	if ctxt.debug {
		log.Printf("\tdebug: "+format, args...)
	}
}

func (ctxt *context) infoPrintf(format string, args ...interface{}) {
	if ctxt.verbose {
		log.Printf("\tinfo: "+format, args...)
	}
}

func parseRules(filename string) []*nameChecker {
	var checkers []*nameChecker

	var config map[string]map[string]string
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("read -rules JSON file: %v", err)
	}
	if err := json.Unmarshal(data, &config); err != nil {
		log.Fatalf("parse -rules JSON file: %v", err)
	}

	for pattern, nameMatcherProps := range config {
		typeRE, err := regexp.Compile(pattern)
		if err != nil {
			log.Fatalf("decode rules: type regexp %q: %v", pattern, err)
		}

		var litMatchers []*literalNameMatcher
		var reMatchers []*regexpNameMatcher
		for k, v := range nameMatcherProps {
			if regexp.QuoteMeta(k) == k {
				litMatchers = append(litMatchers, &literalNameMatcher{
					from:    k,
					warning: fmt.Sprintf("rename to %s", v),
				})
				continue
			}
			re, err := regexp.Compile(k)
			if err != nil {
				log.Fatalf("decode rules: %q: %q: %v", pattern, k, err)
			}
			reMatchers = append(reMatchers, &regexpNameMatcher{
				re:      re,
				warning: v,
			})
		}

		// For performance reasons, we want literal matchers go first,
		// regexp matchers go after them.
		var list []nameMatcher
		for _, m := range litMatchers {
			list = append(list, m)
		}
		for _, m := range reMatchers {
			list = append(list, m)
		}

		checkers = append(checkers, &nameChecker{
			typeRE:   typeRE,
			matchers: &nameMatcherList{list: list},
		})
	}

	return checkers
}

type nameChecker struct {
	typeRE   *regexp.Regexp
	matchers *nameMatcherList
}

type nameMatcherList struct {
	list []nameMatcher
}

type nameMatcher interface {
	Match(name string) bool
	Warning() string
}

type literalNameMatcher struct {
	from    string
	warning string
}

func (m *literalNameMatcher) Match(name string) bool {
	return m.from == name
}

func (m *literalNameMatcher) Warning() string { return m.warning }

type regexpNameMatcher struct {
	re      *regexp.Regexp
	warning string
}

func (m *regexpNameMatcher) Match(name string) bool {
	return m.re.MatchString(name)
}

func (m *regexpNameMatcher) Warning() string { return m.warning }

func removePointers(typ types.Type) types.Type {
	if ptr, ok := typ.(*types.Pointer); ok {
		return removePointers(ptr.Elem())
	}
	return typ
}
