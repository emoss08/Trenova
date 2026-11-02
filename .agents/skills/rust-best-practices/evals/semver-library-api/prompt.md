---
tags: [rust-best-practices]
max_turns: 30
timeout_seconds: 900
allowed_tools: [Read, Glob, Grep, Skill, Write, Edit, Bash]
---

I need a small Rust library crate for parsing and comparing semantic version strings, like `1.2.3`, `1.2.3-beta.1`, and `1.2.3+build.5` (prerelease and build metadata per the semver 2.0 spec). Give me the full `src/lib.rs`: a `Version` type, parsing from a string, ordering that follows semver precedence rules (prerelease versions sort before the release, build metadata is ignored for ordering), and a way to print a version back out as a string. Include unit tests. Keep it dependency-free. Write the file to the current directory.
