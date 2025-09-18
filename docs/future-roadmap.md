# Future Roadmap

## Console Output Enhancements

- Historical performance banner for commands: show a concise hint before runs, e.g.,
  "Last time this test command took N seconds, avg M seconds over last O runs."
  - Source: durations from `.3pio/runs/*/test-run.md` and run metadata
  - Goal: quick feedback loop to spot regressions without opening reports
