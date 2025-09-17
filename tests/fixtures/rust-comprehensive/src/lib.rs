/// A simple calculator module with comprehensive tests
pub mod calculator {
    /// Adds two numbers together
    ///
    /// # Examples
    ///
    /// ```
    /// use rust_comprehensive::calculator::add;
    /// assert_eq!(add(2, 3), 5);
    /// ```
    pub fn add(a: i32, b: i32) -> i32 {
        a + b
    }

    /// Subtracts the second number from the first
    ///
    /// # Examples
    ///
    /// ```
    /// use rust_comprehensive::calculator::subtract;
    /// assert_eq!(subtract(5, 3), 2);
    /// ```
    ///
    /// ```
    /// use rust_comprehensive::calculator::subtract;
    /// assert_eq!(subtract(0, 5), -5);
    /// ```
    pub fn subtract(a: i32, b: i32) -> i32 {
        a - b
    }

    /// Multiplies two numbers
    ///
    /// # Examples
    ///
    /// ```
    /// use rust_comprehensive::calculator::multiply;
    /// assert_eq!(multiply(3, 4), 12);
    /// ```
    pub fn multiply(a: i32, b: i32) -> i32 {
        a * b
    }

    /// Divides the first number by the second
    ///
    /// # Examples
    ///
    /// ```
    /// use rust_comprehensive::calculator::divide;
    /// assert_eq!(divide(10, 2), Some(5));
    /// assert_eq!(divide(10, 0), None);
    /// ```
    pub fn divide(a: i32, b: i32) -> Option<i32> {
        if b == 0 {
            None
        } else {
            Some(a / b)
        }
    }
}

/// String utilities module
pub mod strings {
    /// Reverses a string
    ///
    /// # Examples
    ///
    /// ```
    /// use rust_comprehensive::strings::reverse;
    /// assert_eq!(reverse("hello"), "olleh");
    /// ```
    pub fn reverse(s: &str) -> String {
        s.chars().rev().collect()
    }

    /// Checks if a string is a palindrome
    ///
    /// # Examples
    ///
    /// ```
    /// use rust_comprehensive::strings::is_palindrome;
    /// assert!(is_palindrome("racecar"));
    /// assert!(!is_palindrome("hello"));
    /// ```
    pub fn is_palindrome(s: &str) -> bool {
        let clean: String = s.chars().filter(|c| c.is_alphanumeric()).collect();
        let reversed = reverse(&clean);
        clean.eq_ignore_ascii_case(&reversed)
    }
}

#[cfg(test)]
mod unit_tests {
    use super::calculator::*;
    use super::strings::*;

    mod calculator_tests {
        use super::*;

        #[test]
        fn test_addition() {
            assert_eq!(add(2, 2), 4);
            assert_eq!(add(-1, 1), 0);
            assert_eq!(add(0, 0), 0);
        }

        #[test]
        fn test_subtraction() {
            assert_eq!(subtract(5, 3), 2);
            assert_eq!(subtract(0, 5), -5);
            assert_eq!(subtract(-5, -3), -2);
        }

        #[test]
        fn test_multiplication() {
            assert_eq!(multiply(3, 4), 12);
            assert_eq!(multiply(-2, 3), -6);
            assert_eq!(multiply(0, 100), 0);
        }

        #[test]
        fn test_division() {
            assert_eq!(divide(10, 2), Some(5));
            assert_eq!(divide(10, 0), None);
            assert_eq!(divide(0, 5), Some(0));
        }

        #[test]
        #[should_panic]
        fn test_panic_example() {
            panic!("This test should panic");
        }

        #[test]
        #[ignore]
        fn test_ignored() {
            // This test is ignored by default
            assert_eq!(1 + 1, 2);
        }
    }

    mod string_tests {
        use super::*;

        #[test]
        fn test_reverse() {
            assert_eq!(reverse("hello"), "olleh");
            assert_eq!(reverse(""), "");
            assert_eq!(reverse("a"), "a");
        }

        #[test]
        fn test_palindrome() {
            assert!(is_palindrome("racecar"));
            assert!(is_palindrome("A man a plan a canal Panama"));
            assert!(!is_palindrome("hello"));
        }
    }
}