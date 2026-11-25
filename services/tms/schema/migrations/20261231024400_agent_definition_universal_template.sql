-- The general assistant became the universal agent: an empty scope reaches every area.
ALTER TABLE "agent_definitions"
    DROP CONSTRAINT IF EXISTS "ck_agent_definitions_template";

UPDATE
    "agent_definitions"
SET
    "template" = 'Universal'
WHERE
    "template" = 'GeneralAssistant';

ALTER TABLE "agent_definitions"
    ADD CONSTRAINT "ck_agent_definitions_template" CHECK (
        "template" IS NULL OR "template" IN (
            'DispatchAssistant', 'BillingAssistant', 'ComplianceAssistant', 'CustomerAssistant',
            'Universal', 'BillingException', 'DispatchAssignment', 'ImportAssistant',
            'LoadMonitor', 'ShipmentIntake', 'CashApplication', 'DetentionDesk',
            'CredentialDesk', 'CustomerUpdateDesk', 'CarrierRiskDesk', 'IntakeDesk',
            'LoadEntryCheck', 'ServiceFailureDesk', 'InsightAnalyst', 'EDIDesk',
            'FormulaAssistant', 'BooksKeeper', 'SettlementsClerk', 'Receivables',
            'MasterDataSteward', 'WorkforceCoordinator', 'FuelTaxClerk', 'ReportAnalyst',
            'FactoringClerk'
        )
    );
