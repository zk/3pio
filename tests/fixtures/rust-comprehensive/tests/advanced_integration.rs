use rust_comprehensive::calculator::*;

#[test]
fn test_large_numbers() {
    assert_eq!(add(1000000, 2000000), 3000000);
    assert_eq!(multiply(1000, 1000), 1000000);
}

#[test]
fn test_negative_numbers() {
    assert_eq!(add(-10, -20), -30);
    assert_eq!(subtract(-10, -20), 10);
    assert_eq!(multiply(-5, -5), 25);
    assert_eq!(divide(-10, -2), Some(5));
}

#[test]
#[ignore]
fn test_performance() {
    // This would be a performance test
    // Ignored by default
    for i in 0..1000000 {
        let _ = add(i, i);
    }
}