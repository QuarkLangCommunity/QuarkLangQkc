package lang

import (
	"bytes"
	"sort"
	"strings"
	"testing"
	"time"
)

// 尾延迟 P99/P999：QuarkLang 解释器（零 GC，block 复用）vs Go（GC 停顿）。
func TestP99Latency(t *testing.T) {
	qsrc := "fn main(IOStream io) {\n" +
		"    int i = 0;\n" +
		"    while (i < 50000) {\n" +
		"        List<int> l = [1, 2, 3];\n" +
		"        delete l;\n" +
		"        i = i + 1;\n" +
		"    }\n" +
		"}\n"
	prog, err := Compile(qsrc)
	if err != nil {
		t.Fatal(err)
	}
	var qd []time.Duration
	for k := 0; k < 500; k++ {
		var out bytes.Buffer
		t0 := time.Now()
		Run(prog, "t.qk", nil, strings.NewReader(""), &out)
		qd = append(qd, time.Since(t0))
	}
	q50, q99, q999 := pctsLat(qd)

	var gd []time.Duration
	for k := 0; k < 500; k++ {
		t0 := time.Now()
		for i := 0; i < 50000; i++ {
			s := make([]int, 3)
			_ = s[0]
		}
		gd = append(gd, time.Since(t0))
	}
	g50, g99, g999 := pctsLat(gd)

	t.Logf("QuarkLang解释器: P50=%v P99=%v P999=%v", q50, q99, q999)
	t.Logf("Go:              P50=%v P99=%v P999=%v", g50, g99, g999)
}

func pctsLat(d []time.Duration) (p50, p99, p999 time.Duration) {
	s := make([]time.Duration, len(d))
	copy(s, d)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	i50 := int(float64(len(s)) * 0.50)
	i99 := int(float64(len(s)) * 0.99)
	i999 := int(float64(len(s)) * 0.999)
	if i99 >= len(s) {
		i99 = len(s) - 1
	}
	if i999 >= len(s) {
		i999 = len(s) - 1
	}
	return s[i50], s[i99], s[i999]
}
