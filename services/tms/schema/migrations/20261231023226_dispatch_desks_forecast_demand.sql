-- dispatch_forecast_demand reads the organization's freight forecast: which areas will be
-- busy and where to send trucks. Desks keep the tools they were created with, so the
-- dispatch desks made before the tool existed could not answer "where should I send trucks
-- tomorrow?" from it. Every dispatch desk and coverage agent gets it, as new ones do.
UPDATE "agent_definitions"
SET "tool_names" = array_append(coalesce("tool_names", '{}'::text[]), 'dispatch_forecast_demand')
WHERE "template" IN ('DispatchAssistant', 'DispatchAssignment')
  AND NOT ('dispatch_forecast_demand' = ANY(coalesce("tool_names", '{}'::text[])));
