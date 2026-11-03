package agenttoolcatalog_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// toolPackages are the trees whose tools name their parameters and fields,
// each with the package that holds the words its tools share.
var toolPackages = map[string]string{
	"../querytools": "../querytools/querytool",
	"../agenttools": "../agenttools/toolparams",
}

var (
	roleConstant = regexp.MustCompile(`^([Pp]aram|[Ff]ield|[Aa]rg|[Ii]nput)[A-Z]`)
	camelWord    = regexp.MustCompile(`^[a-z][A-Za-z0-9]*$`)
	wordPart     = regexp.MustCompile(`[a-z0-9]+|[A-Z][a-z0-9]*`)
)

// initialisms are the words a constant's name writes in capitals, as the
// repository writes them elsewhere: shipmentId is ParamShipmentID, proNumber is
// ParamProNumber.
var initialisms = map[string]string{
	"id": "ID", "ids": "IDs", "url": "URL", "urls": "URLs", "edi": "EDI", "dot": "DOT",
	"hos": "HOS", "bol": "BOL", "scac": "SCAC", "ifta": "IFTA", "pod": "POD", "isa": "ISA",
	"eta": "ETA", "cdl": "CDL", "mvr": "MVR", "twic": "TWIC", "pto": "PTO", "api": "API",
	"json": "JSON", "html": "HTML", "sql": "SQL", "uuid": "UUID", "ip": "IP", "mc": "MC",
	"ein": "EIN", "ach": "ACH", "gl": "GL", "sla": "SLA", "csv": "CSV", "pdf": "PDF",
	"x12": "X12", "as2": "AS2", "sftp": "SFTP", "vin": "VIN", "sms": "SMS", "eld": "ELD",
	"usdot": "USDOT", "un": "UN",
}

func nameFor(word string) string {
	var b strings.Builder
	for _, part := range wordPart.FindAllString(word, -1) {
		if upper, ok := initialisms[strings.ToLower(part)]; ok {
			b.WriteString(upper)
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]) + part[1:])
	}

	return b.String()
}

// A tool's parameter or a field it reads is one constant per word in its
// package, named for the word: param<Word> for what a tool takes, field<Word>
// for what it only reads or writes, read from the tree's shared package when
// that package already names the word. transferId was once both paramEDITransferID
// and argTransferID in one package, and a reader could not tell they were the
// same argument.
func TestEveryParameterAndFieldConstantIsNamedForItsWord(t *testing.T) {
	t.Parallel()

	type declared struct{ name, at string }
	var problems []string
	for root, sharedDir := range toolPackages {
		byPackage := map[string]map[string][]declared{}
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") ||
				strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "_gen.go") {
				return err
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			words := byPackage[filepath.Dir(path)]
			if words == nil {
				words = map[string][]declared{}
				byPackage[filepath.Dir(path)] = words
			}
			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.CONST {
					continue
				}
				for _, spec := range gen.Specs {
					value, ok := spec.(*ast.ValueSpec)
					if !ok || len(value.Names) != 1 || len(value.Values) != 1 {
						continue
					}
					literal, ok := value.Values[0].(*ast.BasicLit)
					name := value.Names[0].Name
					if !ok || literal.Kind != token.STRING || !roleConstant.MatchString(name) {
						continue
					}
					word, err := strconv.Unquote(literal.Value)
					if err != nil || !camelWord.MatchString(word) {
						continue
					}
					words[word] = append(words[word], declared{name: name, at: path})
				}
			}

			return nil
		})
		require.NoError(t, err)

		shared := byPackage[filepath.Clean(sharedDir)]
		for directory, words := range byPackage {
			for word, names := range words {
				if owner, ok := shared[word]; ok && directory != filepath.Clean(sharedDir) &&
					ast.IsExported(owner[0].name) {
					problems = append(
						problems,
						fmt.Sprintf(
							"%s: %s holds %q, which %s already names",
							names[0].at,
							names[0].name,
							word,
							filepath.Base(sharedDir)+"."+owner[0].name,
						),
					)
					continue
				}
				if len(names) > 1 {
					all := make([]string, 0, len(names))
					for _, n := range names {
						all = append(all, n.name)
					}
					sort.Strings(all)
					problems = append(problems, fmt.Sprintf("%s: %q is declared %d times (%s)",
						directory, word, len(names), strings.Join(all, ", ")))
					continue
				}
				name := names[0].name
				role := strings.ToLower(roleConstant.FindStringSubmatch(name)[1])
				if role != "param" && role != "field" {
					problems = append(problems, fmt.Sprintf(
						"%s: %s is a %s constant; a tool's words are param or field",
						names[0].at,
						name,
						role,
					))
					continue
				}
				want := role + nameFor(word)
				if ast.IsExported(name) {
					want = strings.ToUpper(want[:1]) + want[1:]
				}
				if name != want {
					problems = append(problems, fmt.Sprintf(
						"%s: %s holds %q and should be named %s", names[0].at, name, word, want))
				}
			}
		}
	}

	sort.Strings(problems)
	require.Emptyf(t, problems, "%d constants break the naming rule:\n%s",
		len(problems), strings.Join(problems, "\n"))
}
