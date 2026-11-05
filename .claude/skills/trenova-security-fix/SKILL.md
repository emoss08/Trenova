---
name: trenova-security-fix
description: How to fix a security finding in Trenova so it stays fixed and breaks nothing. Use when asked to fix a vulnerability, an audit finding or a permission, tenancy, sign-in, money-race or secret-handling bug, and when reviewing such a fix. Covers checking the finding against source, sweeping sibling paths, keeping legitimate flows working, regression tests and what to say in the handoff. For finding vulnerabilities in the first place, use security-audit.
---

# Fixing a security finding in Trenova

A finding names one path. The defect is usually a rule that isn't enforced everywhere, and a fix
that closes only the reported path leaves the rest open. Fixes also break things: in one audit the
worst regression was a fix that locked people out of their own uploads.

Read [security-review.md](../../../docs/engineering/backend/security-review.md) first; it is the
list of what goes wrong here. [trust-boundaries.md](../../../docs/engineering/trust-boundaries.md)
says who can reach what. Load `trenova-backend` or `trenova-frontend` for the layer you touch.

## 1. Confirm it in source

A report, including one from an agent, is a claim. Read the path yourself before changing anything.

- Name the lower-trust actor, what they send, the control crossed and the result. If you can't,
  it isn't confirmed yet.
- Look for what the report missed. Three things have turned "findings" into nothing: the data
  really is withheld one layer down, the policy reads `false` and admits nobody, or a test failed
  on its own setup before it checked anything.
- A test named for tenant isolation that fails is not evidence of a leak until you've read why.

## 2. Find the rule, then every path

State the invariant in one sentence. Search for each writer or reader of the effect (the column,
the repository method), not the feature's name: REST, GraphQL, bulk actions, agent tools, Temporal
activities. Fix them all in the same change, or name the ones you left in the report.

## 3. Read the callers of what you change

Before reusing a helper, read what its callers do with the answer. `SecretInUse` looked like the
way to stop asking for an AS2 key again, and its caller deletes what it returns false for.

A test that asserts the old behaviour may be defending a rule on purpose. Find out which before you
change it. `TestOutboundTools_AreGatedOnTheCommunicationTheySend` and
`TestEveryToolADeskRunsOnIsOneAnAgentMayCall` both rejected a static permission change that looked
right and pointed to a better design.

## 4. Fix at the last trusted decision point

- Prefer the write proving its precondition over a check before it.
- Reuse the pattern a sibling surface already has, and an existing type before adding a twin.
- Route the rare case through the flow that already exists rather than building a second approval
  path: a short pay that needs approval goes through an invoice adjustment.
- Prefer a narrow rule to a blanket refusal when real customers need the case: a federated session
  may switch to an organization that trusts the same issuer, rather than being barred from
  switching.
- No nil guards or optional dependencies: `fx` builds every dependency.

## 5. Keep legitimate flows working

Walk the "A fix can break the thing it guards" list in `security-review.md`: agent and system
actors, rows already in the database, what a default now does, and statuses the client reads. If
the behaviour a person sees changes, it goes in the handoff.

## 6. Test both sides

Each fix gets a test that the bad case is refused and a test that the good case still works. Include
the actor the rule doesn't apply to, such as an agent for a person-only grant. A lost race is
asserted against a real database, in an integration test. Name tests for the behaviour, without
describing how to attack it.

Then run the generators the change touches. A tool policy changes the catalog snapshot; new text
needs `task i18n`, every locale and `task i18n-check`; a migration needs `task db-stamp`
([generated-artifacts.md](../../../docs/engineering/generated-artifacts.md)).

## 7. Hand off

The repository is public, so a commit message says what now holds, never how to break it. Report
what changed, each behaviour a person will notice, what you did not verify and why, and any sibling
path you left. Say what is proven by a test and what only by reading.

After a batch of fixes, ask for an independent read of the diff by someone who hasn't seen your
reasoning. Don't mark your own work.
