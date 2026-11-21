-- On a shared server the support key set is the server's: each assertion names its tenant's
-- instance as its audience, so the server holds no instance id. A dedicated instance still
-- stores its own (the service refuses a dedicated key set without one).

ALTER TABLE "managed_support_access_states"
    ALTER COLUMN "instance_id" DROP NOT NULL;

COMMENT ON TABLE "managed_support_access_states" IS 'The keys that verify the control plane''s support assertions, the staff domain and, on a dedicated instance, its id (null on a shared server, whose assertions name their tenant''s instance); no tenant data';
