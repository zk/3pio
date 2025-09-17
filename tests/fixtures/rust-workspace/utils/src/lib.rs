/// Utility functions for the workspace

pub fn capitalize(s: &str) -> String {
    let mut chars = s.chars();
    match chars.next() {
        None => String::new(),
        Some(first) => first.to_uppercase().collect::<String>() + chars.as_str(),
    }
}

pub fn reverse(s: &str) -> String {
    s.chars().rev().collect()
}

pub fn count_words(s: &str) -> usize {
    s.split_whitespace().count()
}

pub mod math {
    pub fn factorial(n: u32) -> u32 {
        match n {
            0 | 1 => 1,
            _ => n * factorial(n - 1),
        }
    }

    pub fn is_prime(n: u32) -> bool {
        if n <= 1 {
            return false;
        }
        for i in 2..=(n as f64).sqrt() as u32 {
            if n % i == 0 {
                return false;
            }
        }
        true
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_capitalize() {
        assert_eq!(capitalize("hello"), "Hello");
        assert_eq!(capitalize("WORLD"), "WORLD");
        assert_eq!(capitalize(""), "");
        assert_eq!(capitalize("a"), "A");
    }

    #[test]
    fn test_reverse() {
        assert_eq!(reverse("hello"), "olleh");
        assert_eq!(reverse("rust"), "tsur");
        assert_eq!(reverse(""), "");
        assert_eq!(reverse("a"), "a");
    }

    #[test]
    fn test_count_words() {
        assert_eq!(count_words("hello world"), 2);
        assert_eq!(count_words("  multiple   spaces  "), 2);
        assert_eq!(count_words(""), 0);
        assert_eq!(count_words("one"), 1);
    }

    mod math_tests {
        use super::math::*;

        #[test]
        fn test_factorial() {
            assert_eq!(factorial(0), 1);
            assert_eq!(factorial(1), 1);
            assert_eq!(factorial(5), 120);
            assert_eq!(factorial(10), 3628800);
        }

        #[test]
        fn test_is_prime() {
            assert!(!is_prime(0));
            assert!(!is_prime(1));
            assert!(is_prime(2));
            assert!(is_prime(3));
            assert!(!is_prime(4));
            assert!(is_prime(5));
            assert!(is_prime(17));
            assert!(!is_prime(100));
        }
    }
}