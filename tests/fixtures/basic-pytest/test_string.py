import pytest
import sys


class TestStringOperations:
    def test_concatenate_strings(self):
        print("Testing string concatenation...")
        assert "Hello" + " " + "World" == "Hello World"
        assert "foo" + "bar" == "foobar"
        print("Concatenation tests passed!")

    def test_fail_this_test(self):
        print("This test is expected to fail")
        print("Error: Intentional failure for testing", file=sys.stderr)
        assert "foo" == "bar"  # This will fail

    @pytest.mark.skip(reason="Skipping this test for consistency with Jest fixture")
    def test_skip_this_test(self):
        print("This should not run")
        assert True == False

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