---
tags: [rust-best-practices]
max_turns: 30
timeout_seconds: 900
allowed_tools: [Read, Glob, Grep, Skill, Write, Edit, Bash]
---

Write me a Rust CLI called `logsum`. It takes a directory path as its only argument, reads every `*.jsonl` file in it (one JSON object per line, each with at least a `level` string field and a `ts` timestamp string), and prints a table of how many events there are per level plus the total, sorted by count descending. Skip lines that are not valid JSON but tell me how many were skipped at the end. Use serde and serde_json. I want `Cargo.toml` and `src/main.rs`. It is an internal tool so keep it practical, but it must not crash on bad input (missing directory, unreadable file, garbage lines).
