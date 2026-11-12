--keel:notx
-- A record's case has one owner: the person whose work it is. The owner, and
-- whoever covers for them while they are away, are the only people who decide
-- it; everyone else is shown it only when they may see every case. A record
-- with no row here is unowned, and the first person to claim it owns it.
CREATE TABLE IF NOT EXISTS "record_case_owners"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "record_kind" varchar(60) NOT NULL,
    "record_id" varchar(100) NOT NULL,
    "owner_id" varchar(100) NOT NULL,
    "source" varchar(20) NOT NULL DEFAULT 'Auto',
    "assigned_by_id" varchar(100),
    "assigned_at" bigint NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_record_case_owners" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_record_case_owners_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_record_case_owners_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_record_case_owners_owner" FOREIGN KEY ("owner_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_record_case_owners_assigned_by" FOREIGN KEY ("assigned_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_record_case_owners_source" CHECK ("source" IN ('Auto', 'Claimed', 'Assigned'))
);

COMMENT ON TABLE "record_case_owners" IS 'Who owns a record''s case at the Desk: one person per record, set from the record (its owner, its fleet''s manager, its biller or collector), by a claim, or by someone who may assign cases';

COMMENT ON COLUMN "record_case_owners"."source" IS 'How the owner came to own the case: Auto from the record, Claimed by the owner, or Assigned by another person';

-- One owner per record, and the read every case and every list makes: is
-- this record's case someone else's.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_record_case_owners_record" ON "record_case_owners"("organization_id", "business_unit_id", "record_id");

-- What one person owns, for the lists that hide other people's cases.
CREATE INDEX IF NOT EXISTS "idx_record_case_owners_owner" ON "record_case_owners"("organization_id", "business_unit_id", "owner_id", "record_id");

-- Someone away hands their cases to a cover for a while, the same way they
-- hand over approvals.
ALTER TYPE approval_scope_enum ADD VALUE IF NOT EXISTS 'Cases';
