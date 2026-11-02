---
type: llm
focus: trace
weight: 1
---

src/lib.rs implements std::str::FromStr for Version, so callers can use "1.2.3".parse::<Version>()
