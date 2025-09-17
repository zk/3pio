pub fn add(a: i32, b: i32) -> i32 {
    a + b
}

pub fn subtract(a: i32, b: i32) -> i32 {
    a - b
}

pub fn multiply(a: i32, b: i32) -> i32 {
    a * b
}

pub fn divide(a: i32, b: i32) -> Result<i32, String> {
    if b == 0 {
        Err("Division by zero".to_string())
    } else {
        Ok(a / b)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_add() {
        assert_eq!(add(2, 3), 5);
        assert_eq!(add(-1, 1), 0);
    }

    #[test]
    fn test_subtract() {
        assert_eq!(subtract(5, 3), 2);
        assert_eq!(subtract(0, 5), -5);
    }

    #[test]
    fn test_multiply() {
        assert_eq!(multiply(3, 4), 12);
        assert_eq!(multiply(-2, 3), -6);
    }

    #[test]
    fn test_divide() {
        assert_eq!(divide(10, 2), Ok(5));
        assert_eq!(divide(9, 3), Ok(3));
    }

    #[test]
    fn test_divide_by_zero() {
        assert_eq!(divide(5, 0), Err("Division by zero".to_string()));
    }

    #[test]
    #[should_panic(expected = "assertion failed")]
    fn test_that_fails() {
        assert_eq!(add(2, 2), 5, "assertion failed");
    }

    #[test]
    #[ignore]
    fn test_ignored() {
        assert_eq!(add(100, 200), 300);
    }
}

mod integration_tests {
    use super::*;

    #[test]
    fn test_combined_operations() {
        let result = add(multiply(2, 3), subtract(10, 5));
        assert_eq!(result, 11); // (2*3) + (10-5) = 6 + 5 = 11
    }
}