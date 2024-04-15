-- Create "business_units" table
CREATE TABLE "business_units"
(
    "id"                uuid              NOT NULL,
    "created_at"        timestamptz       NOT NULL,
    "updated_at"        timestamptz       NOT NULL,
    "status"            character varying NOT NULL DEFAULT 'A',
    "name"              character varying NOT NULL,
    "entity_key"        character varying NOT NULL,
    "phone_number"      character varying NOT NULL,
    "address"           character varying NULL,
    "city"              character varying NOT NULL,
    "state"             character varying NOT NULL,
    "country"           character varying NOT NULL,
    "postal_code"       character varying NOT NULL,
    "tax_id"            character varying NOT NULL,
    "subscription_plan" character varying NOT NULL,
    "description"       character varying NULL,
    "legal_name"        character varying NOT NULL,
    "contact_name"      character varying NULL,
    "contact_email"     character varying NULL,
    "paid_until"        timestamptz       NULL,
    "settings"          jsonb             NULL,
    "free_trial"        boolean           NOT NULL DEFAULT false,
    "parent_id"         uuid              NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "business_units_business_units_parent" FOREIGN KEY ("parent_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "business_units_parent_id_key" to table: "business_units"
CREATE UNIQUE INDEX "business_units_parent_id_key" ON "business_units" ("parent_id");
-- Create index "businessunit_entity_key" to table: "business_units"
CREATE UNIQUE INDEX "businessunit_entity_key" ON "business_units" ("entity_key");
-- Create index "businessunit_name" to table: "business_units"
CREATE UNIQUE INDEX "businessunit_name" ON "business_units" ("name");
-- Create "organizations" table
CREATE TABLE "organizations"
(
    "id"         uuid        NOT NULL,
    "created_at" timestamptz NOT NULL,
    "updated_at" timestamptz NOT NULL,
    PRIMARY KEY ("id")
);
-- Create "users" table
CREATE TABLE "users"
(
    "id"         uuid        NOT NULL,
    "created_at" timestamptz NOT NULL,
    "updated_at" timestamptz NOT NULL,
    PRIMARY KEY ("id")
);
