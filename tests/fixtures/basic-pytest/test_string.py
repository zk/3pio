import pytest


class TestStringOperations:
    def test_concatenate_strings(self):
        print("Testing string concatenation...")
        assert "Hello" + " " + "World" == "Hello World"
        assert "foo" + "bar" == "foobar"
        print("Concatenation tests passed!")

    def test_string_length(self):
        print("Testing string length...")
        assert len("Hello") == 5
        assert len("Testing") == 7
        print("Length tests passed!")

    def test_string_uppercase(self):
        print("Testing uppercase conversion...")
        assert "hello".upper() == "HELLO"
        assert "world".upper() == "WORLD"
        print("Uppercase tests passed!")

    def test_string_contains(self):
        print("Testing string contains...")
        assert "Hello" in "Hello World"
        assert "test" in "testing"
        print("Contains tests passed!")