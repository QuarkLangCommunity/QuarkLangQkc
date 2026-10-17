// Command policy measures the interpreter's logical block policy with the same tide shape the
// standard's §10 describes: allocate, use and release, then read the counters the manager exposes.
package main

import (
	"flag"
	"fmt"

	"github.com/QuarkLangCommunity/QuarkLangQkc/internal/lang"
)

// main runs the reuse and fragmentation unit workloads and prints their counters as TSV.
func main() {
	waves := flag.Int("waves", 10000, "number of tide waves")
	fragWaves := flag.Int("frag-waves", 1000, "number of mixed-survival waves")
	flag.Parse()

	m := lang.NewMemoryManager()
	for w := 0; w < *waves; w++ {
		ids := make([]int, 0, 100)
		for j := 0; j < 100; j++ {
			ids = append(ids, m.Alloc(64, 0))
		}
		for _, id := range ids {
			m.Delete(id)
		}
	}
	reuse := 0.0
	if m.AllocCalls > 0 {
		reuse = float64(m.ReusedCount) / float64(m.AllocCalls)
	}
	fmt.Printf("reuse\t%d\t%d\t%.6f\t%d\t%d\t%.6f\n", *waves, m.AllocCalls, reuse, m.NewBlocks, m.BlockCount(), m.Fragmentation())

	f := lang.NewMemoryManager()
	sizes := []int{8, 16, 32, 64}
	for w := 0; w < *fragWaves; w++ {
		ids := make([]int, 0, 1000)
		for j := 0; j < 1000; j++ {
			ids = append(ids, f.Alloc(sizes[j%len(sizes)], 0))
		}
		for j := 500; j < 1000; j++ {
			f.Delete(ids[j])
		}
	}
	fragReuse := 0.0
	if f.AllocCalls > 0 {
		fragReuse = float64(f.ReusedCount) / float64(f.AllocCalls)
	}
	fmt.Printf("fragmentation\t%d\t%d\t%.6f\t%d\t%d\t%.6f\n", *fragWaves, f.AllocCalls, fragReuse, f.NewBlocks, f.BlockCount(), f.Fragmentation())
}
