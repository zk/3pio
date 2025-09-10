import pytest


class TestMathOperations:
    def test_add_numbers_correctly(self):
        print("Testing addition...")
        assert 1 + 1 == 2
        assert 10 + 5 == 15
        print("Addition tests passed!")

    def test_multiply_numbers_correctly(self):
        print("Testing multiplication...")
        assert 2 * 3 == 6
        assert 5 * 5 == 25
        print("Multiplication tests passed!")

    def test_handle_division(self):
        print("Testing division...")
        assert 10 / 2 == 5
        assert 20 / 4 == 5
        print("Division tests passed!")