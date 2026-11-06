-- A proposal raised by a run an event, a schedule, a continuous pass, a finished wait or a
-- background start had no reason in its call, and its rationale quoted the runtime's own
-- instruction as if a person had asked: "Asked to assign dispatch move in reply to: An event
-- started this run: ...". The runtime now quotes only what a person typed and otherwise says
-- the agent gave no reason; this rewrites the rationales stored before that to the same
-- words. Requests people typed in chat keep their quote.
UPDATE "agent_proposals"
SET "rationale" = regexp_replace(
    "rationale",
    '^Asked to (.+?) in reply to: “.*$',
    'The agent asked to \1 without explaining why.'
)
WHERE "rationale" ~ '^Asked to .+? in reply to: “(An event started this run|This is a scheduled run\.|This is one pass of a continuous run\.|An earlier run parked this work on a wait|A person started this run\.)';
