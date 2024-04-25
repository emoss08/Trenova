-- Add a new index for the `permissions` table
CREATE UNIQUE INDEX unq_permissions_name_organization_id ON permissions (LOWER(name), organization_id);

-- Add a new index for the `roles` table
CREATE UNIQUE INDEX unq_roles_name_organization_id ON roles (LOWER(name), organization_id);