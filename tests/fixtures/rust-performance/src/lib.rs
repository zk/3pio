/// Performance testing module with many tests

pub fn add(a: i32, b: i32) -> i32 {
    a + b
}

pub fn multiply(a: i32, b: i32) -> i32 {
    a * b
}

pub fn factorial(n: u32) -> u64 {
    match n {
        0 | 1 => 1,
        _ => (n as u64) * factorial(n - 1),
    }
}

pub fn fibonacci(n: u32) -> u64 {
    match n {
        0 => 0,
        1 => 1,
        _ => fibonacci(n - 1) + fibonacci(n - 2),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    // Generate a large number of tests to test 3pio performance
    #[test] fn test_add_001() { assert_eq!(add(1, 2), 3); }
    #[test] fn test_add_002() { assert_eq!(add(2, 3), 5); }
    #[test] fn test_add_003() { assert_eq!(add(3, 4), 7); }
    #[test] fn test_add_004() { assert_eq!(add(4, 5), 9); }
    #[test] fn test_add_005() { assert_eq!(add(5, 6), 11); }
    #[test] fn test_add_006() { assert_eq!(add(6, 7), 13); }
    #[test] fn test_add_007() { assert_eq!(add(7, 8), 15); }
    #[test] fn test_add_008() { assert_eq!(add(8, 9), 17); }
    #[test] fn test_add_009() { assert_eq!(add(9, 10), 19); }
    #[test] fn test_add_010() { assert_eq!(add(10, 11), 21); }

    #[test] fn test_multiply_001() { assert_eq!(multiply(1, 2), 2); }
    #[test] fn test_multiply_002() { assert_eq!(multiply(2, 3), 6); }
    #[test] fn test_multiply_003() { assert_eq!(multiply(3, 4), 12); }
    #[test] fn test_multiply_004() { assert_eq!(multiply(4, 5), 20); }
    #[test] fn test_multiply_005() { assert_eq!(multiply(5, 6), 30); }
    #[test] fn test_multiply_006() { assert_eq!(multiply(6, 7), 42); }
    #[test] fn test_multiply_007() { assert_eq!(multiply(7, 8), 56); }
    #[test] fn test_multiply_008() { assert_eq!(multiply(8, 9), 72); }
    #[test] fn test_multiply_009() { assert_eq!(multiply(9, 10), 90); }
    #[test] fn test_multiply_010() { assert_eq!(multiply(10, 11), 110); }

    #[test] fn test_factorial_001() { assert_eq!(factorial(1), 1); }
    #[test] fn test_factorial_002() { assert_eq!(factorial(2), 2); }
    #[test] fn test_factorial_003() { assert_eq!(factorial(3), 6); }
    #[test] fn test_factorial_004() { assert_eq!(factorial(4), 24); }
    #[test] fn test_factorial_005() { assert_eq!(factorial(5), 120); }
    #[test] fn test_factorial_006() { assert_eq!(factorial(6), 720); }
    #[test] fn test_factorial_007() { assert_eq!(factorial(7), 5040); }
    #[test] fn test_factorial_008() { assert_eq!(factorial(8), 40320); }
    #[test] fn test_factorial_009() { assert_eq!(factorial(9), 362880); }
    #[test] fn test_factorial_010() { assert_eq!(factorial(10), 3628800); }

    #[test] fn test_fibonacci_001() { assert_eq!(fibonacci(1), 1); }
    #[test] fn test_fibonacci_002() { assert_eq!(fibonacci(2), 1); }
    #[test] fn test_fibonacci_003() { assert_eq!(fibonacci(3), 2); }
    #[test] fn test_fibonacci_004() { assert_eq!(fibonacci(4), 3); }
    #[test] fn test_fibonacci_005() { assert_eq!(fibonacci(5), 5); }
    #[test] fn test_fibonacci_006() { assert_eq!(fibonacci(6), 8); }
    #[test] fn test_fibonacci_007() { assert_eq!(fibonacci(7), 13); }
    #[test] fn test_fibonacci_008() { assert_eq!(fibonacci(8), 21); }
    #[test] fn test_fibonacci_009() { assert_eq!(fibonacci(9), 34); }
    #[test] fn test_fibonacci_010() { assert_eq!(fibonacci(10), 55); }
}

#[cfg(test)]
mod module_a {
    use super::*;

    #[test] fn test_a_001() { assert_eq!(add(100, 1), 101); }
    #[test] fn test_a_002() { assert_eq!(add(100, 2), 102); }
    #[test] fn test_a_003() { assert_eq!(add(100, 3), 103); }
    #[test] fn test_a_004() { assert_eq!(add(100, 4), 104); }
    #[test] fn test_a_005() { assert_eq!(add(100, 5), 105); }
    #[test] fn test_a_006() { assert_eq!(add(100, 6), 106); }
    #[test] fn test_a_007() { assert_eq!(add(100, 7), 107); }
    #[test] fn test_a_008() { assert_eq!(add(100, 8), 108); }
    #[test] fn test_a_009() { assert_eq!(add(100, 9), 109); }
    #[test] fn test_a_010() { assert_eq!(add(100, 10), 110); }
}

#[cfg(test)]
mod module_b {
    use super::*;

    #[test] fn test_b_001() { assert_eq!(multiply(10, 1), 10); }
    #[test] fn test_b_002() { assert_eq!(multiply(10, 2), 20); }
    #[test] fn test_b_003() { assert_eq!(multiply(10, 3), 30); }
    #[test] fn test_b_004() { assert_eq!(multiply(10, 4), 40); }
    #[test] fn test_b_005() { assert_eq!(multiply(10, 5), 50); }
    #[test] fn test_b_006() { assert_eq!(multiply(10, 6), 60); }
    #[test] fn test_b_007() { assert_eq!(multiply(10, 7), 70); }
    #[test] fn test_b_008() { assert_eq!(multiply(10, 8), 80); }
    #[test] fn test_b_009() { assert_eq!(multiply(10, 9), 90); }
    #[test] fn test_b_010() { assert_eq!(multiply(10, 10), 100); }
}

#[cfg(test)]
mod module_c {
    use super::*;

    #[test] fn test_c_001() { assert!(fibonacci(5) > 0); }
    #[test] fn test_c_002() { assert!(fibonacci(6) > fibonacci(5)); }
    #[test] fn test_c_003() { assert!(fibonacci(7) > fibonacci(6)); }
    #[test] fn test_c_004() { assert!(fibonacci(8) > fibonacci(7)); }
    #[test] fn test_c_005() { assert!(fibonacci(9) > fibonacci(8)); }
    #[test] fn test_c_006() { assert!(fibonacci(10) > fibonacci(9)); }
    #[test] fn test_c_007() { assert!(fibonacci(11) > fibonacci(10)); }
    #[test] fn test_c_008() { assert!(fibonacci(12) > fibonacci(11)); }
    #[test] fn test_c_009() { assert!(fibonacci(13) > fibonacci(12)); }
    #[test] fn test_c_010() { assert!(fibonacci(14) > fibonacci(13)); }
}