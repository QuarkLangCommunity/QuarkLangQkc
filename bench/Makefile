all: qk c rust go erl
qk:
	cd ../compiler && go build -o qkc .
	./qkc ../examples/fib.qk > /tmp/fib35.ll && clang /tmp/fib35.ll -o /tmp/qk_fib35 -O3 -Wno-override-module
	time /tmp/qk_fib35
c:
	clang c/fib.c -o /tmp/c_fib35 -O3 && time /tmp/c_fib35
rust:
	rustc -O fib.rs -o /tmp/rs_fib35 && time /tmp/rs_fib35
go:
	go build -o /tmp/go_fib35 fib.go && time /tmp/go_fib35
erl:
	erlc -o /tmp fib.erl && time erl -noshell -pa /tmp -run fib main -s init stop
