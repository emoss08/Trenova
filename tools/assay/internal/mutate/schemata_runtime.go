package mutate

import (
	"sort"
	"strings"
)

// SchemataFileName is the file injected into a mutated package through -overlay.
// It never touches the source tree; the overlay makes the compiler see a file that
// does not exist on disk.
const SchemataFileName = "assay_schemata.go"

const (
	helperOn      = "assayMutOn"
	helperAddSub  = "assayMutAddSub"
	helperSubAdd  = "assayMutSubAdd"
	helperMulQuo  = "assayMutMulQuo"
	helperQuoMul  = "assayMutQuoMul"
	helperRemMul  = "assayMutRemMul"
	helperLssLeq  = "assayMutLssLeq"
	helperLeqLss  = "assayMutLeqLss"
	helperGtrGeq  = "assayMutGtrGeq"
	helperGeqGtr  = "assayMutGeqGtr"
	helperLandLor = "assayMutLandLor"
	helperLorLand = "assayMutLorLand"
	helperIncDec  = "assayMutIncDec"
	helperDecInc  = "assayMutDecInc"
)

const (
	constraintNum = "assayMutNum"
	constraintInt = "assayMutInt"
	constraintOrd = "assayMutOrd"
)

// SchemataEnvVar names the mutant to activate. Zero, and anything unparseable,
// leaves every site behaving exactly as the original — which is what makes the
// same binary usable as its own control.
const SchemataEnvVar = "ASSAY_MUTANT"

// TrackEnvVar switches on init-site tracking. Only the mutant harness sets it:
// tracking costs a sync.Map store per site execution until it is turned off, and
// the env-selected path never turns it off because it never needs the answer.
const TrackEnvVar = "ASSAY_MUTANT_TRACK"

// schemataPrelude is always emitted. os, sync and sync/atomic are the only
// imports: the lower the dependency, the smaller the chance of an import cycle
// when the package under mutation is itself low-level.
//
// The active mutant is an atomic because the harness switches it between m.Run
// calls while goroutines leaked by earlier tests may still be executing sites.
// Init tracking exists for the harness's one blind spot: package init runs
// before TestMain can select anything, so a site executed during init has
// already behaved as the original by the time the harness switches — the
// harness must judge such mutants in a process of their own, where the
// environment selects the mutant before init runs.
const schemataPrelude = `var assayMutActive atomic.Int32
var assayMutTracking atomic.Bool
var assayMutInitSeen sync.Map

func init() {
	assayMutActive.Store(assayMutSelect(os.Getenv("` + SchemataEnvVar + `")))
	assayMutTracking.Store(os.Getenv("` + TrackEnvVar + `") == "1")
}

func assayMutSelect(raw string) int32 {
	var n int32
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int32(c-'0')
	}

	return n
}

// AssaySelect switches the active mutant in-process. Generated for the mutant
// harness; a normal env-selected run never calls it.
func AssaySelect(id int32) { assayMutActive.Store(id) }

// AssayInitSites reports every mutation site executed before it was called and
// stops tracking. The harness calls it once, at the top of TestMain.
func AssayInitSites() []int32 {
	assayMutTracking.Store(false)
	var out []int32
	assayMutInitSeen.Range(func(key, _ any) bool {
		out = append(out, key.(int32))

		return true
	})

	return out
}

func ` + helperOn + `(id int32) bool {
	if assayMutTracking.Load() {
		assayMutInitSeen.Store(id, struct{}{})
	}

	return assayMutActive.Load() == id
}`

var constraintSource = map[string]string{
	constraintNum: `type ` + constraintNum + ` interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~complex64 | ~complex128
}`,
	constraintInt: `type ` + constraintInt + ` interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}`,
	constraintOrd: `type ` + constraintOrd + ` interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~string
}`,
}

type helperDefinition struct {
	constraint string
	source     string
}

func binaryHelper(name, constraint, result, active, original string) helperDefinition {
	return helperDefinition{
		constraint: constraint,
		source: "func " + name + "[T " + constraint + "](id int32, a, b T) " + result + " {\n" +
			"\tif " + helperOn + "(id) {\n" +
			"\t\treturn " + active + "\n" +
			"\t}\n\n" +
			"\treturn " + original + "\n" +
			"}",
	}
}

func stepHelper(name, active, original string) helperDefinition {
	return helperDefinition{
		constraint: constraintNum,
		source: "func " + name + "[T " + constraintNum + "](id int32, p *T) {\n" +
			"\tif " + helperOn + "(id) {\n" +
			"\t\t" + active + "\n\n" +
			"\t\treturn\n" +
			"\t}\n\n" +
			"\t" + original + "\n" +
			"}",
	}
}

var helperSource = map[string]helperDefinition{
	helperAddSub: binaryHelper(helperAddSub, constraintNum, "T", "a - b", "a + b"),
	helperSubAdd: binaryHelper(helperSubAdd, constraintNum, "T", "a + b", "a - b"),
	helperMulQuo: binaryHelper(helperMulQuo, constraintNum, "T", "a / b", "a * b"),
	helperQuoMul: binaryHelper(helperQuoMul, constraintNum, "T", "a * b", "a / b"),
	helperRemMul: binaryHelper(helperRemMul, constraintInt, "T", "a * b", "a % b"),
	helperLssLeq: binaryHelper(helperLssLeq, constraintOrd, "bool", "a <= b", "a < b"),
	helperLeqLss: binaryHelper(helperLeqLss, constraintOrd, "bool", "a < b", "a <= b"),
	helperGtrGeq: binaryHelper(helperGtrGeq, constraintOrd, "bool", "a >= b", "a > b"),
	helperGeqGtr: binaryHelper(helperGeqGtr, constraintOrd, "bool", "a > b", "a >= b"),

	// `&&` short-circuits on a false left operand and `||` on a true one, so the
	// mutated form has to be spelled out rather than swapping a token.
	helperLandLor: {source: "func " + helperLandLor + `(id int32, a bool, b func() bool) bool {
	if ` + helperOn + `(id) {
		if a {
			return true
		}

		return b()
	}

	if !a {
		return false
	}

	return b()
}`},
	helperLorLand: {source: "func " + helperLorLand + `(id int32, a bool, b func() bool) bool {
	if ` + helperOn + `(id) {
		if !a {
			return false
		}

		return b()
	}

	if a {
		return true
	}

	return b()
}`},

	helperIncDec: stepHelper(helperIncDec, "*p--", "*p++"),
	helperDecInc: stepHelper(helperDecInc, "*p++", "*p--"),
}

const schemataHeader = `// Code generated by assay for mutation testing. DO NOT EDIT.
//
// assay injects this file into the package through -overlay while it runs; it is
// never written to the source tree. Every mutation site in the package compiles
// into a switch on ` + SchemataEnvVar + `, so one binary can be run once per
// mutant instead of being rebuilt for each. An unset or zero ` + SchemataEnvVar + `
// leaves every site behaving exactly as the unmutated original.
`

// SchemataSource renders the injected file for a package, emitting only the
// helpers the package's sites actually reference.
func SchemataSource(packageName string, helpers []string) []byte {
	wanted := make(map[string]struct{}, len(helpers))
	for _, name := range helpers {
		if _, ok := helperSource[name]; ok {
			wanted[name] = struct{}{}
		}
	}

	names := make([]string, 0, len(wanted))
	constraints := make(map[string]struct{})
	for name := range wanted {
		names = append(names, name)
		if c := helperSource[name].constraint; c != "" {
			constraints[c] = struct{}{}
		}
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString(schemataHeader)
	b.WriteString("\npackage ")
	b.WriteString(packageName)
	b.WriteString("\n\nimport (\n\t\"os\"\n\t\"sync\"\n\t\"sync/atomic\"\n)\n\n")
	b.WriteString(schemataPrelude)
	b.WriteString("\n")

	for _, name := range sortedSetKeys(constraints) {
		b.WriteString("\n")
		b.WriteString(constraintSource[name])
		b.WriteString("\n")
	}

	for _, name := range names {
		b.WriteString("\n")
		b.WriteString(helperSource[name].source)
		b.WriteString("\n")
	}

	return []byte(b.String())
}
