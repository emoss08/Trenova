---
type: llm
focus: trace
weight: 1
---

PartialEq and Hash on Version agree with Ord: either all are derived together, or the hand-written impls compare the same fields (ignoring build metadata consistently), and the docs say which fields equality ignores
