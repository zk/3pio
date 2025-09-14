use criterion::{black_box, criterion_group, criterion_main, Criterion};
use rust_benchmarks::{fibonacci, factorial};

fn fibonacci_benchmark(c: &mut Criterion) {
    c.bench_function("fib 10", |b| b.iter(|| fibonacci(black_box(10))));
}

fn factorial_benchmark(c: &mut Criterion) {
    c.bench_function("fact 10", |b| b.iter(|| factorial(black_box(10))));
}

criterion_group!(benches, fibonacci_benchmark, factorial_benchmark);
criterion_main!(benches);