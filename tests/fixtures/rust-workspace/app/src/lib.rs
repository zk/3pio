/// Main application that uses core and utils
use core::CoreEngine;
use utils::{capitalize, reverse, count_words};
use utils::math::{factorial, is_prime};

pub struct Application {
    engine: CoreEngine,
    name: String,
}

impl Application {
    pub fn new(name: &str, initial_value: i32) -> Self {
        Application {
            engine: CoreEngine::new(initial_value),
            name: capitalize(name),
        }
    }

    pub fn process_with_validation(&self, input: i32) -> Result<i32, String> {
        if self.engine.validate(input) {
            Ok(self.engine.process(input))
        } else {
            Err(format!("Invalid input: {}", input))
        }
    }

    pub fn get_info(&self) -> String {
        format!("App: {}, Value: {}", self.name, self.engine.value)
    }

    pub fn analyze_text(&self, text: &str) -> (String, usize) {
        (reverse(text), count_words(text))
    }

    pub fn compute_factorial(&self, n: u32) -> u32 {
        factorial(n)
    }

    pub fn check_prime(&self, n: u32) -> bool {
        is_prime(n)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_app_creation() {
        let app = Application::new("test app", 100);
        assert_eq!(app.name, "Test app");
        assert_eq!(app.engine.value, 100);
    }

    #[test]
    fn test_process_with_validation_success() {
        let app = Application::new("validator", 50);
        let result = app.process_with_validation(25);
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), 75);
    }

    #[test]
    fn test_process_with_validation_failure() {
        let app = Application::new("validator", 50);
        let result = app.process_with_validation(150);
        assert!(result.is_err());
        assert_eq!(result.unwrap_err(), "Invalid input: 150");
    }

    #[test]
    fn test_get_info() {
        let app = Application::new("info test", 42);
        assert_eq!(app.get_info(), "App: Info test, Value: 42");
    }

    #[test]
    fn test_analyze_text() {
        let app = Application::new("analyzer", 0);
        let (reversed, count) = app.analyze_text("hello world");
        assert_eq!(reversed, "dlrow olleh");
        assert_eq!(count, 2);
    }

    #[test]
    fn test_math_operations() {
        let app = Application::new("math", 0);
        assert_eq!(app.compute_factorial(5), 120);
        assert!(app.check_prime(17));
        assert!(!app.check_prime(18));
    }

    #[test]
    #[should_panic(expected = "attempt to multiply with overflow")]
    fn test_factorial_overflow() {
        let app = Application::new("overflow", 0);
        // This will overflow for u32
        let _ = app.compute_factorial(20);
    }
}

#[cfg(test)]
mod integration_tests {
    use super::*;

    #[test]
    fn test_full_application_workflow() {
        let app = Application::new("workflow test", 10);

        // Test validation and processing
        let result = app.process_with_validation(50).unwrap();
        assert_eq!(result, 60);

        // Test text analysis
        let (reversed, count) = app.analyze_text("testing the application");
        assert_eq!(count, 3);
        assert!(reversed.starts_with("noitacilppa"));

        // Test math operations
        assert!(app.check_prime(7));
        assert_eq!(app.compute_factorial(4), 24);
    }
}