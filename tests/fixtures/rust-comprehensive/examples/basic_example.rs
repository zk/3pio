use rust_comprehensive::calculator::*;
use rust_comprehensive::strings::*;

fn main() {
    println!("Calculator Examples:");
    println!("10 + 5 = {}", add(10, 5));
    println!("10 - 5 = {}", subtract(10, 5));
    println!("10 * 5 = {}", multiply(10, 5));
    println!("10 / 5 = {:?}", divide(10, 5));

    println!("\nString Examples:");
    println!("Reverse 'hello': {}", reverse("hello"));
    println!("Is 'racecar' a palindrome? {}", is_palindrome("racecar"));
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_example_usage() {
        assert_eq!(add(1, 1), 2);
        assert_eq!(reverse("test"), "tset");
    }
}