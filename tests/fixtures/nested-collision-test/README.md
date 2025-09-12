# Nested Collision Test Fixture

This fixture demonstrates how 3pio handles test files with identical names in different directories.

## Test Structure

The fixture contains multiple `utils.test.js` files in different directories:
- `src/utils.test.js`
- `lib/utils.test.js`
- `features/auth/utils.test.js`
- `features/user/utils.test.js`
- `tests/integration/api/utils.test.js`
- `tests/unit/helpers/utils.test.js`

Plus a root-level test:
- `index.test.js`

## Expected Behavior

When running tests with 3pio, the output structure should preserve the directory hierarchy to avoid collisions:

```
.3pio/runs/[timestamp]/reports/
├── features/
│   ├── auth/
│   │   └── utils.test.js.log
│   └── user/
│       └── utils.test.js.log
├── lib/
│   └── utils.test.js.log
├── src/
│   └── utils.test.js.log
├── tests/
│   ├── integration/
│   │   └── api/
│   │       └── utils.test.js.log
│   └── unit/
│       └── helpers/
│           └── utils.test.js.log
└── index.test.js.log
```

## Running the Test

```bash
# From this directory
../../../build/3pio npm test

# Or with Jest directly
../../../build/3pio npx jest
```

This ensures that all test files maintain their unique paths even when they share the same filename.