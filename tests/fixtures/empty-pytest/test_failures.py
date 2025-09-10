import pytest


def test_should_fail():
    """This test should fail"""
    print("This test is designed to fail")
    assert 1 + 1 == 3


def test_another_failure():
    """Another failing test"""
    print("Another failing test")
    assert "hello" == "world"


def test_skip_this():
    """This test should be skipped"""
    pytest.skip("Skipping this test")
    assert True