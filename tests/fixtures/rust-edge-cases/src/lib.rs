/// Edge case testing module
pub fn will_panic() {
    panic!("This function always panics!");
}

pub fn divide_by_zero() -> i32 {
    let x = 10;
    let y = 0;
    if y == 0 {
        panic!("Division by zero!");
    }
    x / y
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::thread;
    use std::time::Duration;

    #[test]
    fn test_normal_pass() {
        assert_eq!(2 + 2, 4);
    }

    #[test]
    fn test_assertion_failure() {
        assert_eq!(2 + 2, 5, "Math is broken!");
    }

    #[test]
    #[should_panic(expected = "This function always panics!")]
    fn test_expected_panic() {
        will_panic();
    }

    #[test]
    fn test_unexpected_panic() {
        panic!("Unexpected panic occurred!");
    }

    #[test]
    fn test_divide_by_zero_panic() {
        let _ = divide_by_zero();
    }

    #[test]
    fn test_with_unicode_in_name() {
        // Unicode characters in function names are limited
        assert!(true);
    }

    #[test]
    fn test_with_numbers_123() {
        assert!(true);
    }

    #[test]
    fn test_long_running() {
        // Simulate a test that takes time
        thread::sleep(Duration::from_millis(100));
        assert!(true);
    }

    #[test]
    fn test_assert_with_custom_message() {
        assert!(false, "Custom assertion message: expected true but got false");
    }

    #[test]
    fn test_multiple_assertions_first_fails() {
        assert_eq!(1, 2);
        assert_eq!(2, 2); // This won't be reached
        assert_eq!(3, 3); // This won't be reached
    }

    #[test]
    #[ignore]
    fn test_ignored_that_would_fail() {
        panic!("This test is ignored");
    }

    #[test]
    fn test_overflow_panic() {
        let x: u8 = 255;
        let _result = x.wrapping_add(1); // In debug mode, regular addition would panic
        // Force a panic to test panic handling
        let y: u8 = 255;
        let _overflow = y.checked_add(1).expect("Overflow occurred!");
    }

    #[test]
    fn test_index_out_of_bounds() {
        let v = vec![1, 2, 3];
        let _ = v[10]; // Index out of bounds
    }

    #[test]
    fn test_unwrap_none() {
        let x: Option<i32> = None;
        x.unwrap(); // This will panic
    }

    #[test]
    fn test_expect_with_message() {
        let x: Result<i32, &str> = Err("Something went wrong");
        x.expect("Failed to get value"); // This will panic with custom message
    }
}

#[cfg(test)]
mod nested_module_tests {
    #[test]
    fn nested_test_pass() {
        assert!(true);
    }

    #[test]
    fn nested_test_fail() {
        assert!(false, "Nested test failure");
    }

    mod deeply_nested {
        #[test]
        fn very_nested_test() {
            assert_eq!(1, 1);
        }

        #[test]
        fn very_nested_panic() {
            panic!("Deep panic!");
        }
    }
}