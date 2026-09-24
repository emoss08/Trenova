ALTER TABLE "agent_definitions"
    DROP CONSTRAINT IF EXISTS "ck_agent_definitions_template";

--bun:split

ALTER TABLE "agent_definitions"
    ADD CONSTRAINT "ck_agent_definitions_template" CHECK (
        "template" IS NULL OR "template" IN (
            'DispatchAssistant', 'BillingAssistant', 'ComplianceAssistant', 'CustomerAssistant',
            'GeneralAssistant', 'BillingException', 'DispatchAssignment', 'ImportAssistant',
            'LoadMonitor', 'ShipmentIntake', 'CashApplication', 'DetentionDesk',
            'CredentialDesk', 'CustomerUpdateDesk', 'CarrierRiskDesk', 'IntakeDesk'
        )
    ) NOT VALID;
