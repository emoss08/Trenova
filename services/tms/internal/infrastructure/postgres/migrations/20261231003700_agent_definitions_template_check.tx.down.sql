-- NOT VALID restores the constraint for new rows without checking the rows the
-- original eight-value list never allowed, which would otherwise make this
-- rollback fail on exactly the data the up migration exists to accept.
ALTER TABLE "agent_definitions"
    ADD CONSTRAINT "ck_agent_definitions_template" CHECK ("template" IS NULL OR "template" IN ('DispatchAssistant', 'BillingAssistant', 'ComplianceAssistant', 'CustomerAssistant', 'GeneralAssistant', 'BillingException', 'DispatchAssignment', 'ImportAssistant')) NOT VALID;
