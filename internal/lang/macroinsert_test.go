package lang

import (
	"testing"
)

// #insert(#ast(parameter)) and #execute restored: explicit direct insertion / identifier splicing
func TestMacroInsertDirectives(t *testing.T) {
	out, err := runSrc(t, `#macro pair (a, b) {
    #ifdef EXPLAIN
        #return #insert(#ast(a)) + #insert(#ast(b))
    #endif
}

fn main(IOStream io) {
    io.println(pair(20, 22));
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "42\n" {
		t.Fatalf("got %q", out)
	}
}

// #insert(#ast(...)) directly inserts every parameter in declaration order (forwarding use)
func TestMacroInsertAll(t *testing.T) {
	out, err := runSrc(t, `fn add2(int a, int b) int {
    return a + b;
}
#macro calladd (a, b) {
    #ifdef EXPLAIN
        #return add2(#insert(#ast(...)))
    #endif
}

fn main(IOStream io) {
    io.println(calladd(20, 22));
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "42\n" {
		t.Fatalf("got %q", out)
	}
}

// #execute(name) directly inserts an identifier token: splices together a callable name
func TestMacroExecuteSplice(t *testing.T) {
	out, err := runSrc(t, `fn give() int {
    return 7;
}
#macro callit () {
    #ifdef EXPLAIN
        #return #execute(give)()
    #endif
}

fn main(IOStream io) {
    io.println(callit());
}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "7\n" {
		t.Fatalf("got %q", out)
	}
}
