use rust_comprehensive::calculator::{add, subtract};

fn main() {
    println!("2 + 3 = {}", add(2, 3));
    println!("5 - 3 = {}", subtract(5, 3));
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_binary_functionality() {
        // Test that the binary can use the library functions
        assert_eq!(add(10, 20), 30);
        assert_eq!(subtract(50, 25), 25);
    }
}