// Tide control, observable allocation: the same three-int allocate/use/drop loop as the QuarkLang
// benchmark, with black_box on the buffer pointer so rustc cannot remove the Vec allocation.
use std::hint::black_box;
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
        black_box(v.as_ptr());
        acc += (v[0] + v[2]) as i64;
    }
    let elapsed_us = t0.elapsed().as_micros();
    println!("{}", elapsed_us);
    println!("{}", acc);
}
