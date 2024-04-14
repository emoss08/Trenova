-- Create "shipment_charges" table
CREATE TABLE
    "shipment_charges" (
        "id" uuid NOT NULL,
        "created_at" timestamptz NOT NULL,
        "updated_at" timestamptz NOT NULL,
        "version" bigint NOT NULL DEFAULT 1,
        "business_unit_id" uuid NOT NULL,
        "organization_id" uuid NOT NULL,
        PRIMARY KEY ("id"),
        CONSTRAINT "shipment_charges_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
        CONSTRAINT "shipment_charges_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
    );

-- Create "shipment_comments" table
CREATE TABLE
    "shipment_comments" (
        "id" uuid NOT NULL,
        "created_at" timestamptz NOT NULL,
        "updated_at" timestamptz NOT NULL,
        "version" bigint NOT NULL DEFAULT 1,
        "comment" text NOT NULL,
        "comment_type_id" uuid NOT NULL,
        "shipment_id" uuid NOT NULL,
        "business_unit_id" uuid NOT NULL,
        "organization_id" uuid NOT NULL,
        "created_by" uuid NOT NULL,
        PRIMARY KEY ("id"),
        CONSTRAINT "shipment_comments_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
        CONSTRAINT "shipment_comments_comment_types_shipment_comments" FOREIGN KEY ("comment_type_id") REFERENCES "comment_types" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
        CONSTRAINT "shipment_comments_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
        CONSTRAINT "shipment_comments_shipments_shipment_comments" FOREIGN KEY ("shipment_id") REFERENCES "shipments" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
        CONSTRAINT "shipment_comments_users_shipment_comments" FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
    );