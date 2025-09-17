/// Core functionality for the workspace
pub struct CoreEngine {
    pub value: i32,
}

impl CoreEngine {
    pub fn new(value: i32) -> Self {
        CoreEngine { value }
    }

    pub fn process(&self, input: i32) -> i32 {
        self.value + input
    }

    pub fn validate(&self, input: i32) -> bool {
        input > 0 && input < 100
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_engine_creation() {
        let engine = CoreEngine::new(42);
        assert_eq!(engine.value, 42);
    }

    #[test]
    fn test_engine_process() {
        let engine = CoreEngine::new(10);
        assert_eq!(engine.process(5), 15);
        assert_eq!(engine.process(-5), 5);
    }

    #[test]
    fn test_validation() {
        let engine = CoreEngine::new(0);
        assert!(engine.validate(50));
        assert!(!engine.validate(0));
        assert!(!engine.validate(100));
        assert!(!engine.validate(-1));
    }

    #[test]
    #[ignore]
    fn test_expensive_operation() {
        // This test is ignored because it's expensive
        let engine = CoreEngine::new(1000);
        assert_eq!(engine.process(1000), 2000);
    }
}

#[cfg(test)]
mod integration_tests {
    use super::*;

    #[test]
    fn test_full_workflow() {
        let engine = CoreEngine::new(25);
        let result = if engine.validate(30) {
            engine.process(30)
        } else {
            0
        };
        assert_eq!(result, 55);
    }
}