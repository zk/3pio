import pytest

@pytest.mark.xfail
def test_expected_failure():
    """Test that is expected to fail"""
    assert False  # This will be XFAIL

@pytest.mark.xfail
def test_unexpected_pass():
    """Test that is expected to fail but passes"""
    assert True  # This will be XPASS

@pytest.mark.xfail(reason="Feature not implemented")
def test_xfail_with_reason():
    """Test with xfail reason"""
    raise NotImplementedError()

def test_normal_pass():
    """Normal passing test"""
    assert True

def test_normal_failure():
    """Normal failing test"""
    assert False, "This is a normal failure"

@pytest.mark.skip(reason="Skipped test")
def test_normal_skip():
    """Normal skipped test"""
    assert True

@pytest.mark.xfail(condition=True, reason="Condition is true")
def test_conditional_xfail():
    """Conditionally expected to fail"""
    assert False

@pytest.mark.xfail(strict=True)
def test_strict_xpass():
    """Strict xfail that passes - should fail the suite"""
    assert True  # This will fail the suite in strict mode

class TestXFailClass:
    """Test class with xfail tests"""

    @pytest.mark.xfail
    def test_class_xfail(self):
        """xfail in a class"""
        assert 1 == 2

    @pytest.mark.xfail
    def test_class_xpass(self):
        """xpass in a class"""
        assert 1 == 1

    def test_class_normal(self):
        """Normal test in class"""
        assert True

@pytest.mark.parametrize("value", [1, 2, 3, 4])
@pytest.mark.xfail(condition="value > 2", reason="Values > 2 expected to fail")
def test_parametrized_xfail(value):
    """Parametrized test with conditional xfail"""
    assert value <= 2  # Will xfail for 3 and 4