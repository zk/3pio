describe('Suite with extremely long name that should be handled properly by the reporting system', () => {
  it('should handle this incredibly long test name that goes on and on and on without breaking the markdown formatting', () => {
    expect(true).toBe(true);
  });
  
  it('another test with a reasonably long name to verify consistent formatting', () => {
    expect(1 + 1).toBe(2);
  });
});