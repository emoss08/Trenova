-- An approval that has not gone through yet: the person has a few seconds to
-- undo it before it commits. Non-transactional: a value added to an enum
-- inside a transaction cannot be used until it commits.
ALTER TYPE "agent_proposal_status_enum" ADD VALUE IF NOT EXISTS 'Approving';
