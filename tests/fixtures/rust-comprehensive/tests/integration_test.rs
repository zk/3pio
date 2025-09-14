use rust_comprehensive::calculator::*;
use rust_comprehensive::strings::*;

#[test]
fn test_calculator_integration() {
    // Test combined operations
    let result = add(multiply(3, 4), subtract(10, 5));
    assert_eq!(result, 17); // (3*4) + (10-5) = 12 + 5 = 17
}

#[test]
fn test_string_integration() {
    let input = "hello world";
    let reversed = reverse(input);
    assert_eq!(reversed, "dlrow olleh");

    // Test that reversing twice gives original
    assert_eq!(reverse(&reversed), input);
}

#[test]
fn test_error_handling() {
    assert_eq!(divide(10, 0), None);
    assert_eq!(divide(10, 2), Some(5));
}

mod submodule_tests {
    use super::*;

    #[test]
    fn test_nested_module() {
        assert_eq!(add(100, 200), 300);
    }

    #[test]
    fn test_palindrome_cases() {
        assert!(is_palindrome("A man a plan a canal Panama"));
        assert!(is_palindrome("Was it a car or a cat I saw"));
        assert!(!is_palindrome("This is not a palindrome"));
    }
}