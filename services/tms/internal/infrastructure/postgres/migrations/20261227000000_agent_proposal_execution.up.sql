-- An accepted proposal previously went nowhere: the decision was recorded, the
-- status flipped, and the Temporal signal was discarded by the workflow, so no
-- tool ever ran. These values and columns give an execution attempt somewhere to
-- land, so "approved" can be distinguished from "approved and done" and from
-- "approved and then failed".
--
-- ALTER TYPE ... ADD VALUE cannot run inside a transaction, hence no .tx. prefix.
ALTER TYPE "agent_proposal_status_enum" ADD VALUE IF NOT EXISTS 'Executed';

ALTER TYPE "agent_proposal_status_enum" ADD VALUE IF NOT EXISTS 'ExecutionFailed';

-- The assistant reasons in-process rather than in a workflow, so its runs are
-- opened inline; the subject is the conversation the proposal came out of.
ALTER TYPE "agent_type_enum" ADD VALUE IF NOT EXISTS 'AssistantChat';

ALTER TYPE "agent_subject_type_enum" ADD VALUE IF NOT EXISTS 'AssistantThread';
