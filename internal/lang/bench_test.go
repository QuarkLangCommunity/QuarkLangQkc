package lang

import (
	"bytes"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// ============ performance benchmarks (compute-heavy scenarios) ============

// Arithmetic-heavy loop: 1e6 iterations
const loopSrc = `fn main(IOStream io) {
    int n = 0;
    int i = 0;
    while (i < 1000000) {
        n = n + i * 2 - 1;
        i = i + 1;
    }
    io.println(n);
}
`

// Recursive computation: fib(24)
const fibSrc = `fn fib(int n) int {
    if (n < 2) {
        return n;
    }
    return fib(n - 1) + fib(n - 2);
}

fn main(IOStream io) {
    io.println(fib(24));
}
`

// Dense function calls + List operations
const callSrc = `fn sq(int n) int {
    return n * n;
}

fn main(IOStream io) {
    int total = 0;
    int i = 0;
    while (i < 100000) {
        total = total + sq(i);
        i = i + 1;
    }
    io.println(total);
}
`

func benchRun(b *testing.B, src string) {
	prog, err := Compile(src)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out bytes.Buffer
		if err := Run(prog, "bench.qk", nil, strings.NewReader(""), &out); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		src := fibSrc + "\n// " + strconv.Itoa(i)
		if _, err := Compile(src); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEvalLoop1M(b *testing.B)    { benchRun(b, loopSrc) }
func BenchmarkFib24(b *testing.B)         { benchRun(b, fibSrc) }
func BenchmarkFuncCalls100K(b *testing.B) { benchRun(b, callSrc) }

// ============ temporary-data churn scenario (lots of data created and dropped quickly) ============
// QuarkLang memory manager: linear block reuse + delete returns blocks to the queue at once, no GC pause, no fragmentation.

func BenchmarkMemManagerChurn(b *testing.B) {
	m := NewMemoryManager()
	ids := make([]int, 0, 64)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ids = ids[:0]
		for j := 0; j < 100; j++ {
			ids = append(ids, m.Alloc(64, 0))
		}
		for _, id := range ids {
			m.Delete(id)
		}
	}
	b.StopTimer()
	// Reuse rate: the vast majority of allocations should hit reuse (a new block is requested only once)
	b.ReportMetric(float64(m.ReusedCount)/float64(m.AllocCalls), "reuse-rate")
	b.ReportMetric(float64(m.NewBlocks), "new-blocks")
}

// Control: native Go slice allocation + GC (same churn scale)
// Go control: large-object churn (forces heap allocation, creating real GC pressure)
var goSink [][]interface{}

func BenchmarkGoSliceChurn(b *testing.B) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	gcBefore := stats.NumGC
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100; j++ {
			s := make([]interface{}, 64)
			for k := range s {
				s[k] = k
			}
			goSink = append(goSink, s)
			if len(goSink) > 1000 {
				goSink = goSink[:0]
			} // churn: drop in batches
		}
	}
	b.StopTimer()
	runtime.ReadMemStats(&stats)
	b.ReportMetric(float64(stats.NumGC-gcBefore)/float64(b.N), "gc-count")
}

// Fragmentation rate: internal block fragmentation after mixed-size churn (should approach 0: the lowest-occupancy block is reused first)
func BenchmarkFragmentationAfterChurn(b *testing.B) {
	m := NewMemoryManager()
	sizes := []int{8, 16, 32, 64}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var ids []int
		for j := 0; j < 1000; j++ {
			ids = append(ids, m.Alloc(sizes[j%len(sizes)], 0))
		}
		// Mixed survival: half are deleted into the free queue, half are kept (measures real internal fragmentation)
		for j := 500; j < 1000; j++ {
			m.Delete(ids[j])
		}
	}
	b.StopTimer()
	b.ReportMetric(m.Fragmentation(), "fragmentation")
}

// Go control (large sample triggers GC)
func BenchmarkGoSliceChurnGC(b *testing.B) {
	var sink [][]int
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100000; j++ {
			sink = append(sink, make([]int, 64))
			if len(sink) > 10000 {
				sink = sink[:0]
			}
		}
	}
}

// String text-processing benchmark (1M iterations)
func BenchmarkStringProcessing1M(b *testing.B) {
	srcs := []string{
		`fn main(IOStream io) {
    String s = "  Hello, QuarkLang World  ";
    int i = 0;
    int t = 0;
    while (i < 1000000) {
        t = t + s.trim().size();
        i = i + 1;
    }
    io.println(t);
}`,
		`fn main(IOStream io) {
    String s = "abcdefghijklmnopqrstuvwxyz0123456789";
    int i = 0;
    int t = 0;
    while (i < 1000000) {
        t = t + s.substring(4, 20).size();
        i = i + 1;
    }
    io.println(t);
}`,
		`fn main(IOStream io) {
    String s = "a,b,c,d,e,f,g,h,i,j";
    int i = 0;
    int t = 0;
    while (i < 300000) {
        List<String> parts = s.split(",");
        t = t + parts.size();
        i = i + 1;
    }
    io.println(t);
}`,
	}
	names := []string{"trim+size", "substring", "split"}
	for i, src := range srcs {
		b.Run(names[i], func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				prog, err := Compile(src)
				if err != nil {
					b.Fatal(err)
				}
				var sb strings.Builder
				if err := Run(prog, "t.qk", nil, strings.NewReader(""), &sb); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Chain-compilation benchmark: a left-deep expression chain is the shape that made VM compilation
// quadratic before the per-node type cache in vm.go, because typeOf re-walked each node's whole
// subtree. The comparison that must stay linear is 20k against 60k: the times should roughly double.
func BenchmarkVMCompileChain(b *testing.B) {
	for _, n := range []int{20000, 60000} {
		b.Run(itoaTest(n)+"_additions", func(b *testing.B) {
			src := chainSource(n)
			for i := 0; i < b.N; i++ {
				prog, err := Compile(src)
				if err != nil {
					b.Fatal(err)
				}
				var out strings.Builder
				if err := Run(prog, "chain.qk", nil, strings.NewReader(""), &out); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
