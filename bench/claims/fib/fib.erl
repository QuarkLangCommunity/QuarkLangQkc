-module(fib).
-export([main/0]).

fib(0) -> 0;
fib(1) -> 1;
fib(N) -> fib(N - 1) + fib(N - 2).

main() ->
    {Us, R} = timer:tc(fun() -> fib(30) end),
    io:format("~p~n~p~n", [Us, R]).
