-- The tool that runs a script over the organization's data, agent_run_script, is
-- analysis_run in a namespace of its own: people ask for an analysis, and under agent_ a
-- model read it as the agent's own housekeeping. Saved agent configuration that names it
-- follows the rename.
SELECT "agent_rename_tool"('agent_run_script', 'analysis_run');
