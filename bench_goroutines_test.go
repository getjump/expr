package expr_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/internal/testify/require"
	"github.com/expr-lang/expr/vm"
)

func benchmarkGoroutines(b *testing.B, expression string, env map[string]any, goroutines int) {
	program, err := expr.Compile(expression, expr.Env(env))
	require.NoError(b, err)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := vm.Run(program, env)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkGoroutines_Simple(b *testing.B) {
	env := map[string]any{
		"X": 42,
		"Y": 84,
	}
	expression := `X + Y * 2`

	for _, n := range []int{1, 2, 4, 8, 16, 32} {
		b.Run(fmt.Sprintf("goroutines-%d", n), func(b *testing.B) {
			b.SetParallelism(n)
			benchmarkGoroutines(b, expression, env, n)
		})
	}
}

func BenchmarkGoroutines_Complex(b *testing.B) {
	env := map[string]any{
		"Origin":  "MOW",
		"Country": "RU",
		"Adults":  1,
		"Value":   100,
		"Items": []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}
	expression := `(Origin == "MOW" || Country == "RU") && (Value >= 100 || Adults == 1) && len(Items) > 0 && sum(filter(Items, # > 3)) > 10`

	for _, n := range []int{1, 2, 4, 8, 16, 32} {
		b.Run(fmt.Sprintf("goroutines-%d", n), func(b *testing.B) {
			b.SetParallelism(n)
			benchmarkGoroutines(b, expression, env, n)
		})
	}
}

// BenchmarkGoroutines_ManualFanOut runs a fixed total of work items
// distributed across N goroutines to measure actual throughput scaling.
func BenchmarkGoroutines_ManualFanOut_Simple(b *testing.B) {
	env := map[string]any{
		"X": 42,
		"Y": 84,
	}
	expression := `X + Y * 2`

	program, err := expr.Compile(expression, expr.Env(env))
	require.NoError(b, err)

	for _, numG := range []int{1, 2, 4, 8, 16, 32} {
		b.Run(fmt.Sprintf("goroutines-%d", numG), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				wg.Add(numG)
				for g := 0; g < numG; g++ {
					go func() {
						defer wg.Done()
						_, _ = vm.Run(program, env)
					}()
				}
				wg.Wait()
			}
		})
	}
}

func BenchmarkGoroutines_ManualFanOut_Complex(b *testing.B) {
	env := map[string]any{
		"Origin":  "MOW",
		"Country": "RU",
		"Adults":  1,
		"Value":   100,
		"Items":   []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}
	expression := `(Origin == "MOW" || Country == "RU") && (Value >= 100 || Adults == 1) && len(Items) > 0 && sum(filter(Items, # > 3)) > 10`

	program, err := expr.Compile(expression, expr.Env(env))
	require.NoError(b, err)

	for _, numG := range []int{1, 2, 4, 8, 16, 32} {
		b.Run(fmt.Sprintf("goroutines-%d", numG), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				wg.Add(numG)
				for g := 0; g < numG; g++ {
					go func() {
						defer wg.Done()
						_, _ = vm.Run(program, env)
					}()
				}
				wg.Wait()
			}
		})
	}
}
