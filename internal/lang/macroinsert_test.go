package lang

import (
	"testing"
)

// #insert(#ast(参数)) 与 #execute 恢复：显式直插/标识符拼接
func TestMacroInsertDirectives(t *testing.T) {
	out, err := runSrc(t, `#macro pair (a, b) {
    #when (run) {
        #return #insert(#ast(a)) + #insert(#ast(b))
    }
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

// #insert(#ast(...)) 按声明顺序直插全部参数（转发用法）
func TestMacroInsertAll(t *testing.T) {
	out, err := runSrc(t, `fn add2(int a, int b) int {
    return a + b;
}
#macro calladd (a, b) {
    #when (run) {
        #return add2(#insert(#ast(...)))
    }
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

// #execute(name) 直插标识符 token：拼接出可调用名
func TestMacroExecuteSplice(t *testing.T) {
	out, err := runSrc(t, `fn give() int {
    return 7;
}
#macro callit () {
    #when (run) {
        #return #execute(give)()
    }
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
