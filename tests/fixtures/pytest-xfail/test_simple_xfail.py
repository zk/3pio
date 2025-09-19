import pytest

@pytest.mark.xfail
def test_expected_failure():
    """Test that is expected to fail"""
    assert False  # This will be XFAIL

@pytest.mark.xfail
def test_unexpected_pass():
    """Test that is expected to fail but passes"""
    assert True  # This will be XPASS