CREATE TYPE edi_script_language_enum AS ENUM(
    'Starlark'
);

--bun:split
CREATE TABLE IF NOT EXISTS "edi_template_script_libraries"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "template_version_id" varchar(100) NOT NULL,
    "name" varchar(200) NOT NULL,
    "description" text,
    "language" edi_script_language_enum NOT NULL DEFAULT 'Starlark',
    "script" text NOT NULL,
    "status" edi_template_status_enum NOT NULL DEFAULT 'Draft',
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_edi_template_script_libraries" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_template_script_libraries_version" FOREIGN KEY ("template_version_id", "business_unit_id", "organization_id") REFERENCES "edi_template_versions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_template_script_libraries_name"
    ON "edi_template_script_libraries"("template_version_id", "business_unit_id", "organization_id", lower("name"));

CREATE INDEX IF NOT EXISTS "idx_edi_template_script_libraries_version"
    ON "edi_template_script_libraries"("template_version_id", "business_unit_id", "organization_id", "status");
