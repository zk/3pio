describe('String Operations', () => {
  describe('Upper/Lower', () => {
    it('uppercases', () => {
      expect('abc'.toUpperCase()).to.equal('ABC');
    });

    it('lowercases', () => {
      expect('XYZ'.toLowerCase()).to.equal('xyz');
    });
  });
});

