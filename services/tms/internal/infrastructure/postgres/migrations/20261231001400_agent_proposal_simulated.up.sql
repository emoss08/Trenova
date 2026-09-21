-- A write an agent in simulation was cleared to make, previewed instead of
-- made. Non-transactional: a value added to an enum inside a transaction
-- cannot be used until it commits.
ALTER TYPE "agent_proposal_status_enum" ADD VALUE IF NOT EXISTS 'Simulated';
