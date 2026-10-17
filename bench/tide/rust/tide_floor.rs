// Tide control, dead-code floor: the same loop without black_box, so rustc may scalarize the Vec
// away. The checksum keeps the loop itself alive.
use std::time::Instant;

fn main() {
    let n: u64 = std::env::args()
        .nth(1)
        .and_then(|s| s.parse().ok())
        .unwrap_or(100_000_000);
    let mut acc: i64 = 0;
    let t0 = Instant::now();
    for i in 0..n {
        let mut v: Vec<i32> = Vec::with_capacity(3);
        v.push((i % 7) as i32);
        v.push((i % 11) as i32);
        v.push((i % 13) as i32);
        acc += (v[0] + v[2]) as i64;
    }
    let elapsed_us = t0.elapsed().as_micros();
    println!("{}", elapsed_us);
    println!("{}", acc);
}
