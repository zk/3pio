# Report Samples

Curated examples of 3pio report outputs for different states and runners. Do not commit `.3pio/` runtime directories — only copy relevant `test-run.md` and selected group `index.md` files here.

Structure:
- docs/report-samples/
  - <scenario>/
    - test-run.md
    - groups/…/index.md (subset)
    - manifest.yaml

Generate/update with `scripts/generate-report-samples.sh`, then selectively refine which `index.md` files you keep per scenario.

Keep samples small and focused; prefer deterministic fixtures.

