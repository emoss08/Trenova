--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "osha_case_classification_enum" AS ENUM(
    'NotRecordable',
    'FirstAidOnly',
    'OtherRecordable',
    'JobTransferOrRestriction',
    'DaysAway',
    'Death'
);

CREATE TYPE "osha_illness_type_enum" AS ENUM(
    'Injury',
    'SkinDisorder',
    'RespiratoryCondition',
    'Poisoning',
    'HearingLoss',
    'OtherIllness'
);

CREATE TYPE "injury_treatment_enum" AS ENUM(
    'None',
    'FirstAid',
    'MedicalTreatment',
    'EmergencyRoom',
    'Hospitalized'
);

CREATE TYPE "injury_case_status_enum" AS ENUM(
    'Open',
    'Closed'
);

CREATE TYPE "workers_comp_claim_status_enum" AS ENUM(
    'NotFiled',
    'Filed',
    'Accepted',
    'Denied',
    'Closed'
);

CREATE TYPE "osha_summary_status_enum" AS ENUM(
    'Draft',
    'Certified'
);

--bun:split
CREATE TABLE IF NOT EXISTS "worker_injuries"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "case_number" integer NOT NULL,
    "case_year" smallint NOT NULL,
    "classification" osha_case_classification_enum NOT NULL DEFAULT 'NotRecordable',
    "illness_type" osha_illness_type_enum NOT NULL DEFAULT 'Injury',
    "treatment" injury_treatment_enum NOT NULL DEFAULT 'None',
    "status" injury_case_status_enum NOT NULL DEFAULT 'Open',
    "occurred_at" bigint NOT NULL,
    "reported_at" bigint,
    "returned_to_work_at" bigint,
    "location" varchar(255),
    "description" text NOT NULL,
    "body_part" varchar(100),
    "harmful_agent" varchar(255),
    "days_away" integer NOT NULL DEFAULT 0,
    "days_restricted" integer NOT NULL DEFAULT 0,
    "privacy_case" boolean NOT NULL DEFAULT FALSE,
    "claim_status" workers_comp_claim_status_enum NOT NULL DEFAULT 'NotFiled',
    "claim_number" varchar(100),
    "claim_carrier" varchar(150),
    "claim_filed_at" bigint,
    "claim_closed_at" bigint,
    "safety_event_id" varchar(100),
    "document_id" varchar(100),
    "notes" text,
    "recorded_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_injuries" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_injuries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_injuries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_injuries_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_injuries_safety_event" FOREIGN KEY ("safety_event_id", "organization_id", "business_unit_id") REFERENCES "worker_safety_events"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_worker_injuries_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_injuries_days" CHECK ("days_away" >= 0 AND "days_away" <= 180 AND "days_restricted" >= 0 AND "days_restricted" <= 180),
    CONSTRAINT "chk_worker_injuries_case_number" CHECK ("case_number" > 0),
    CONSTRAINT "chk_worker_injuries_claim" CHECK ("claim_status" = 'NotFiled' OR "claim_filed_at" IS NOT NULL),
    CONSTRAINT "chk_worker_injuries_returned" CHECK ("returned_to_work_at" IS NULL OR "returned_to_work_at" >= "occurred_at")
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_injuries_case" ON "worker_injuries"("organization_id", "business_unit_id", "case_year", "case_number");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_injuries_worker" ON "worker_injuries"("worker_id", "occurred_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_injuries_log" ON "worker_injuries"("organization_id", "business_unit_id", "case_year", "classification");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_injuries_open" ON "worker_injuries"("organization_id", "business_unit_id", "status")
WHERE
    "status" = 'Open';

--bun:split
COMMENT ON TABLE worker_injuries IS 'One row per injury or illness case; the recordable ones are the OSHA 300 log. Case numbers restart each calendar year, which is how the log is read.';

--bun:split
COMMENT ON COLUMN worker_injuries.days_away IS 'Calendar days away from work, capped at 180 as the 300 log requires.';

--bun:split
COMMENT ON COLUMN worker_injuries.privacy_case IS 'A privacy concern case (29 CFR 1904.29(b)(6)): the name is withheld from the posted log and replaced with "Privacy Case".';

--bun:split
CREATE TABLE IF NOT EXISTS "osha_annual_summaries"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "year" smallint NOT NULL,
    "status" osha_summary_status_enum NOT NULL DEFAULT 'Draft',
    "naics_code" varchar(10),
    "average_employees" integer NOT NULL DEFAULT 0,
    "total_hours_worked" bigint NOT NULL DEFAULT 0,
    "executive_name" varchar(100),
    "executive_title" varchar(100),
    "executive_phone" varchar(30),
    "certified_at" bigint,
    "certified_by_id" varchar(100),
    "posted_from" bigint,
    "posted_through" bigint,
    "submitted_at" bigint,
    "submission_reference" varchar(100),
    "notes" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_osha_annual_summaries" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_osha_annual_summaries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_osha_annual_summaries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_osha_annual_summaries_year" CHECK ("year" >= 1971 AND "year" <= 2200),
    CONSTRAINT "chk_osha_annual_summaries_counts" CHECK ("average_employees" >= 0 AND "total_hours_worked" >= 0),
    CONSTRAINT "chk_osha_annual_summaries_certified" CHECK (("status" = 'Certified') = ("certified_at" IS NOT NULL))
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_osha_annual_summaries_year" ON "osha_annual_summaries"("organization_id", "business_unit_id", "year");

--bun:split
COMMENT ON TABLE osha_annual_summaries IS 'The 300A summary for one establishment and year. The organisation is the establishment: it carries one address, which is what OSHA means by one. The case totals are not stored — they are derived from worker_injuries so an amended case cannot leave a stale summary behind.';
