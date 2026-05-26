CREATE TABLE IF NOT EXISTS role_hierarchy_edges(
    "id" varchar(100) NOT NULL,
    "senior_role_id" varchar(100) NOT NULL,
    "junior_role_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "created_by" varchar(100),
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::bigint,
    CONSTRAINT "pk_role_hierarchy_edges" PRIMARY KEY ("id"),
    CONSTRAINT "fk_role_hierarchy_edges_senior_role" FOREIGN KEY ("senior_role_id") REFERENCES "roles"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_role_hierarchy_edges_junior_role" FOREIGN KEY ("junior_role_id") REFERENCES "roles"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_role_hierarchy_edges_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_role_hierarchy_edges_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_role_hierarchy_edges_not_self" CHECK ("senior_role_id" <> "junior_role_id")
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uniq_role_hierarchy_edges_senior_junior"
    ON role_hierarchy_edges("senior_role_id", "junior_role_id");

CREATE INDEX IF NOT EXISTS "idx_role_hierarchy_edges_junior"
    ON role_hierarchy_edges("junior_role_id");

--bun:split
INSERT INTO role_hierarchy_edges(
    id,
    senior_role_id,
    junior_role_id,
    organization_id,
    business_unit_id,
    created_by
)
SELECT
    'rhe_' || substr(md5(r.id || ':' || parent_id), 1, 26),
    r.id,
    parent_id,
    r.organization_id,
    r.business_unit_id,
    r.created_by
FROM roles r
CROSS JOIN LATERAL unnest(COALESCE(r.parent_role_ids, ARRAY[]::text[])) AS parent_id
WHERE parent_id IS NOT NULL
    AND parent_id <> ''
    AND parent_id <> r.id
ON CONFLICT ("senior_role_id", "junior_role_id") DO NOTHING;

--bun:split
CREATE TABLE IF NOT EXISTS role_constraints(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "name" varchar(255) NOT NULL,
    "description" text,
    "type" varchar(20) NOT NULL,
    "max_roles" integer NOT NULL,
    "enabled" boolean NOT NULL DEFAULT TRUE,
    "created_by" varchar(100),
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::bigint,
    CONSTRAINT "pk_role_constraints" PRIMARY KEY ("id"),
    CONSTRAINT "fk_role_constraints_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_role_constraints_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_role_constraints_type" CHECK ("type" IN ('ssd', 'dsd')),
    CONSTRAINT "chk_role_constraints_max_roles" CHECK ("max_roles" >= 1)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_role_constraints_org_type"
    ON role_constraints("organization_id", "type");

--bun:split
CREATE TABLE IF NOT EXISTS role_constraint_roles(
    "id" varchar(100) NOT NULL,
    "role_constraint_id" varchar(100) NOT NULL,
    "role_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::bigint,
    CONSTRAINT "pk_role_constraint_roles" PRIMARY KEY ("id"),
    CONSTRAINT "fk_role_constraint_roles_constraint" FOREIGN KEY ("role_constraint_id") REFERENCES "role_constraints"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_role_constraint_roles_role" FOREIGN KEY ("role_id") REFERENCES "roles"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_role_constraint_roles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_role_constraint_roles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uniq_role_constraint_roles_constraint_role"
    ON role_constraint_roles("role_constraint_id", "role_id");

CREATE INDEX IF NOT EXISTS "idx_role_constraint_roles_role"
    ON role_constraint_roles("role_id");
