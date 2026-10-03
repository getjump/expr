package bench_test

import (
	"testing"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/internal/testify/require"
	"github.com/expr-lang/expr/vm"
)

// ValRecv has methods with a value receiver.
type ValRecv struct {
	N    int
	Name string
}

func (r ValRecv) M0() bool                     { return r.N >= 0 }
func (r ValRecv) M1(a int) bool                { return a >= r.N }
func (r ValRecv) M2(a, b int) bool             { return a+b >= r.N }
func (r ValRecv) M3(a, b, c int) bool          { return a+b+c >= r.N }
func (r ValRecv) E0() (bool, error)            { return r.N >= 0, nil }
func (r ValRecv) E1(a int) (bool, error)       { return a >= r.N, nil }
func (r ValRecv) E2(a, b int) (bool, error)    { return a+b >= r.N, nil }
func (r ValRecv) E3(a, b, c int) (bool, error) { return a+b+c >= r.N, nil }
func (r ValRecv) V(xs ...int) bool             { return len(xs) >= r.N }

// PtrRecv has methods with a pointer receiver.
type PtrRecv struct {
	N    int
	Name string
}

func (r *PtrRecv) M0() bool                     { return r.N >= 0 }
func (r *PtrRecv) M1(a int) bool                { return a >= r.N }
func (r *PtrRecv) M2(a, b int) bool             { return a+b >= r.N }
func (r *PtrRecv) M3(a, b, c int) bool          { return a+b+c >= r.N }
func (r *PtrRecv) E0() (bool, error)            { return r.N >= 0, nil }
func (r *PtrRecv) E1(a int) (bool, error)       { return a >= r.N, nil }
func (r *PtrRecv) E2(a, b int) (bool, error)    { return a+b >= r.N, nil }
func (r *PtrRecv) E3(a, b, c int) (bool, error) { return a+b+c >= r.N, nil }
func (r *PtrRecv) V(xs ...int) bool             { return len(xs) >= r.N }

// Embedded gets M1 and E1 promoted from ValRecv.
type Embedded struct {
	ValRecv
}

type Matcher interface {
	M1(int) bool
	E1(int) (bool, error)
}

type Segment struct {
	ID    int
	Items []int
}

func (s *Segment) Match(x int) bool { return s.ID >= x }

type SegmentVal struct {
	ID    int
	Items []int
}

func (s SegmentVal) Match(x int) bool { return s.ID >= x }

type CallPathEnv struct {
	X, Y, Z int

	Val   ValRecv
	Ptr   *PtrRecv
	Emb   Embedded
	Iface Matcher

	Fn1  func(int) bool
	FnE1 func(int) (bool, error)

	Segments100  []*Segment
	Segments1000 []*Segment
	SegVals1000  []SegmentVal
}

func (CallPathEnv) EnvM1(a int) bool          { return a >= 0 }
func (CallPathEnv) EnvE1(a int) (bool, error) { return a >= 0, nil }

func newCallPathEnv() CallPathEnv {
	env := CallPathEnv{
		X:     1,
		Y:     2,
		Z:     3,
		Val:   ValRecv{Name: "val"},
		Ptr:   &PtrRecv{Name: "ptr"},
		Emb:   Embedded{ValRecv{Name: "emb"}},
		Iface: &PtrRecv{Name: "iface"},
		Fn1:   func(a int) bool { return a >= 0 },
		FnE1:  func(a int) (bool, error) { return a >= 0, nil },
	}
	segments := func(n int) []*Segment {
		s := make([]*Segment, n)
		for i := range s {
			s[i] = &Segment{ID: i, Items: []int{i}}
		}
		return s
	}
	env.Segments100 = segments(100)
	env.Segments1000 = segments(1000)
	env.SegVals1000 = make([]SegmentVal, 1000)
	for i := range env.SegVals1000 {
		env.SegVals1000[i] = SegmentVal{ID: i, Items: []int{i}}
	}
	return env
}

func benchCallPath(b *testing.B, code string, env any, opts ...expr.Option) {
	program, err := expr.Compile(code, append([]expr.Option{expr.Env(env), expr.AsBool()}, opts...)...)
	require.NoError(b, err)

	var machine vm.VM
	var out any

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		out, err = machine.Run(program, env)
	}
	b.StopTimer()

	require.NoError(b, err)
	require.Equal(b, true, out)
}

// BenchmarkCallMethod measures calling methods of struct values from the env.
// M0-M2 have signatures known to vm/func_types (OpCallTyped), M3, E0-E3 and V
// do not and go through OpCall.
func BenchmarkCallMethod(b *testing.B) {
	env := newCallPathEnv()
	cases := []struct{ name, code string }{
		{"value/M0", `Val.M0()`},
		{"value/M1", `Val.M1(X)`},
		{"value/M2", `Val.M2(X, Y)`},
		{"value/M3", `Val.M3(X, Y, Z)`},
		{"value/E0", `Val.E0()`},
		{"value/E1", `Val.E1(X)`},
		{"value/E2", `Val.E2(X, Y)`},
		{"value/E3", `Val.E3(X, Y, Z)`},
		{"value/V", `Val.V(X, Y, Z)`},
		{"pointer/M0", `Ptr.M0()`},
		{"pointer/M1", `Ptr.M1(X)`},
		{"pointer/M2", `Ptr.M2(X, Y)`},
		{"pointer/M3", `Ptr.M3(X, Y, Z)`},
		{"pointer/E0", `Ptr.E0()`},
		{"pointer/E1", `Ptr.E1(X)`},
		{"pointer/E2", `Ptr.E2(X, Y)`},
		{"pointer/E3", `Ptr.E3(X, Y, Z)`},
		{"pointer/V", `Ptr.V(X, Y, Z)`},
		{"embedded/M1", `Emb.M1(X)`},
		{"embedded/E1", `Emb.E1(X)`},
		{"interface/M1", `Iface.M1(X)`},
		{"interface/E1", `Iface.E1(X)`},
		{"env/M1", `EnvM1(X)`},
		{"env/E1", `EnvE1(X)`},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			benchCallPath(b, c.code, env)
		})
	}
}

// BenchmarkCallFunc measures calling plain functions for comparison with
// methods: struct fields, map env values and expr.Function.
func BenchmarkCallFunc(b *testing.B) {
	env := newCallPathEnv()
	mapEnv := map[string]any{
		"X":    1,
		"Fn1":  func(a int) bool { return a >= 0 },
		"FnE1": func(a int) (bool, error) { return a >= 0, nil },
	}
	function := expr.Function(
		"Fn1",
		func(params ...any) (any, error) { return params[0].(int) >= 0, nil },
		new(func(int) bool),
	)
	b.Run("field/Fn1", func(b *testing.B) {
		benchCallPath(b, `Fn1(X)`, env)
	})
	b.Run("field/FnE1", func(b *testing.B) {
		benchCallPath(b, `FnE1(X)`, env)
	})
	b.Run("map/Fn1", func(b *testing.B) {
		benchCallPath(b, `Fn1(X)`, mapEnv)
	})
	b.Run("map/FnE1", func(b *testing.B) {
		benchCallPath(b, `FnE1(X)`, mapEnv)
	})
	b.Run("function/Fn1", func(b *testing.B) {
		benchCallPath(b, `Fn1(X)`, map[string]any{"X": 1}, function)
	})
}

// BenchmarkCallMethodInPredicate measures a method call inside a predicate,
// evaluated once per element.
func BenchmarkCallMethodInPredicate(b *testing.B) {
	env := newCallPathEnv()
	b.Run("pointer/100", func(b *testing.B) {
		benchCallPath(b, `all(Segments100, {.Match(0) && len(.Items) > 0})`, env)
	})
	b.Run("pointer/1000", func(b *testing.B) {
		benchCallPath(b, `all(Segments1000, {.Match(0) && len(.Items) > 0})`, env)
	})
	b.Run("value/1000", func(b *testing.B) {
		benchCallPath(b, `all(SegVals1000, {.Match(0) && len(.Items) > 0})`, env)
	})
}
