# Test Group and Subgroup Status Rules

This document defines how 3pio derives group (file) and subgroup statuses across all test runners.

Key concepts
- Group: A logical container for tests (often a file). Groups can contain direct tests and/or subgroups.
- Subgroup: A nested container (e.g., a describe/context, or a child file-like unit in some runners).

Statuses used
- PASS, FAIL, SKIP, PENDING, RUNNING, ERROR, XFAIL (expected fail), XPASS (unexpected pass)

Derivation rules (uniform across runners)
- PASS: Only if all direct tests PASS and all subgroups PASS (no SKIP/FAIL/ERROR/XFAIL/XPASS present).
- SKIP: Only if all direct tests are SKIP and all subgroups are SKIP (i.e., everything under this group was skipped).
- FAIL: Otherwise. Any presence of FAIL/ERROR in the group or any subgroup, or any mixture (e.g., PASS + SKIP) yields FAIL.
- PENDING/RUNNING: Used mid-run; not considered “complete”. When all children reach a complete state, the group transitions to PASS/SKIP/FAIL per the above rules.

Setup/teardown failures
- Group-level errors during setup/teardown (e.g., beforeAll crashes, missing browser) are classified as FAIL at the group level and preserved. 3pio emits a group-level error event with:
  - errorType: SETUP_FAILURE (or similar),
  - phase: setup,
  - message: runner-provided error details.
- Group status will not be overridden by a later generic “group result” message.

Test-level reconciliation (Vitest v2 specifics)
- Planned-but-never-executed tests (e.g., worker aborts) are reconciled at the end:
  - When 3pio learns about a test (via collection or queued/running update), it marks it “planned”.
  - On terminal updates, tests emit PASS/FAIL/SKIP and are marked complete.
  - At end-of-run, any planned-but-unfinished tests are emitted as SKIP by default; for setup failures in the group, 3pio may classify them as ERROR or as part of an overall FAIL, per the above rules.

Progressive reporting
- Optionally, 3pio can emit PENDING for tests seen as queued/running; final statuses replace PENDING as results arrive.
- Console summary and reports reflect deduplicated final counts.

Rationale
- This policy makes non-fully-passing files easy to identify (FAIL surfaces issues), while preserving SKIP only for files that truly had no executed tests.

