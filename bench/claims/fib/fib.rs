use std::time::Instant;

fn fib(n: i64) -> i64 {
    if n < 2 {
        n
    } else {
        fib(n - 1) + fib(n - 2)
    }
}

fn main() {
    let n: i64 = std::env::args()
        .nth(1)
        .and_then(|s| s.parse().ok())
        .unwrap_or(35);
    let t0 = Instant::now();
    let r = fib(n);
    let us = t0.elapsed().as_micros();
    println!("{}", us);
    println!("{}", r);
}
