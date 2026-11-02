---
type: llm
focus: trace
weight: 1
---

The program never panics on bad input: main returns a Result, or returns an ExitCode derived from a Result-returning function, and every failure is reported as a message
