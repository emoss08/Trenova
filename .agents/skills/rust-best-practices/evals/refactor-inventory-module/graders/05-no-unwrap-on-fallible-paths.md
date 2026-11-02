---
type: llm
focus: trace
weight: 1
---

No unwrap() remains on the file read in load, the numeric parses in parse_line, or the map lookups in total_value and restock; each returns an error or Option instead
