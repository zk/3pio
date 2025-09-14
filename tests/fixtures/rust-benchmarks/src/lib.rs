#![feature(test)]

extern crate test;

pub fn fibonacci(n: u32) -> u32 {
    match n {
        0 => 0,
        1 => 1,
        _ => fibonacci(n - 1) + fibonacci(n - 2),
    }
}

pub fn factorial(n: u32) -> u64 {
    match n {
        0 | 1 => 1,
        _ => (n as u64) * factorial(n - 1),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use test::Bencher;

    #[test]
    fn test_fibonacci() {
        assert_eq!(fibonacci(0), 0);
        assert_eq!(fibonacci(1), 1);
        assert_eq!(fibonacci(5), 5);
        assert_eq!(fibonacci(10), 55);
    }

    #[test]
    fn test_factorial() {
        assert_eq!(factorial(0), 1);
        assert_eq!(factorial(1), 1);
        assert_eq!(factorial(5), 120);
    }

    #[bench]
    fn bench_fibonacci_10(b: &mut Bencher) {
        b.iter(|| fibonacci(10));
    }

    #[bench]
    fn bench_factorial_10(b: &mut Bencher) {
        b.iter(|| factorial(10));
    }
}