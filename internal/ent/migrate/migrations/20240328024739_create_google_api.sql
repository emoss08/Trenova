-- Create "google_apis" table
CREATE TABLE "google_apis" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "api_key" character varying NOT NULL, "mileage_unit" character varying NOT NULL DEFAULT 'Imperial', "add_customer_location" boolean NOT NULL DEFAULT false, "auto_geocode" boolean NOT NULL DEFAULT false, "traffic_model" character varying NOT NULL DEFAULT 'BestGuess', "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "google_apis_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "google_apis_organizations_google_api" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Create index "google_apis_api_key_key" to table: "google_apis"
CREATE UNIQUE INDEX "google_apis_api_key_key" ON "google_apis" ("api_key");
-- Create index "google_apis_organization_id_key" to table: "google_apis"
CREATE UNIQUE INDEX "google_apis_organization_id_key" ON "google_apis" ("organization_id");
