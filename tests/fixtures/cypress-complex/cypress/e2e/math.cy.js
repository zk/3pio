describe('Math Suite', () => {
  describe('Addition', () => {
    it('adds positive numbers', () => {
      expect(1 + 2).to.equal(3);
    });

    it.skip('adds negatives (skipped)', () => {
      expect(-1 + -2).to.equal(-3);
    });
  });

  describe('Subtraction', () => {
    it('subtracts properly', () => {
      expect(5 - 2).to.equal(3);
    });
  });
});

