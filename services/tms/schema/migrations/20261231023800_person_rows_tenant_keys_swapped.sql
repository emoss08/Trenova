-- Drops the keys that named a person or an organization by id alone and gives their
-- tenant-carrying replacements the old names, so model constraint messages keep resolving. The
-- row-level security policies now compare the business unit too: a role assignment is the
-- standard policy on both columns, and a second factor keeps showing a person their own rows
-- under any organization's scope.

ALTER TABLE "user_role_assignments"
    DROP CONSTRAINT "fk_user_role_assignments_user",
    DROP CONSTRAINT "fk_user_role_assignments_assigned_by",
    DROP CONSTRAINT "fk_user_role_assignments_organization";
ALTER TABLE "user_role_assignments" RENAME CONSTRAINT "tk_80defa39a224dc9fb1c525f2" TO "fk_user_role_assignments_user";
ALTER TABLE "user_role_assignments" RENAME CONSTRAINT "tk_857bef43d07b52ed5702ad7d" TO "fk_user_role_assignments_assigned_by";
ALTER TABLE "user_role_assignments" RENAME CONSTRAINT "tk_700e10ca42bf930a785b5549" TO "fk_user_role_assignments_organization";

ALTER TABLE "mfa_authenticators"
    DROP CONSTRAINT "mfa_authenticators_user_id_fkey",
    DROP CONSTRAINT "mfa_authenticators_organization_id_fkey";
ALTER TABLE "mfa_authenticators" RENAME CONSTRAINT "tk_fd4147c0d23417a221516c3a" TO "mfa_authenticators_user_id_fkey";
ALTER TABLE "mfa_authenticators" RENAME CONSTRAINT "tk_e4891f3a11f5859fe64dcf52" TO "mfa_authenticators_organization_id_fkey";

ALTER TABLE "mfa_recovery_codes"
    DROP CONSTRAINT "fk_mfa_recovery_codes_user",
    DROP CONSTRAINT "fk_mfa_recovery_codes_organization";
ALTER TABLE "mfa_recovery_codes" RENAME CONSTRAINT "tk_f0bf252abab7056ce9aa5215" TO "fk_mfa_recovery_codes_user";
ALTER TABLE "mfa_recovery_codes" RENAME CONSTRAINT "tk_aed69198d06e38b0729d9648" TO "fk_mfa_recovery_codes_organization";

SELECT trenova_rls.apply_policy('public.user_role_assignments', trenova_rls.standard_expr('organization_id', 'business_unit_id'));

SELECT
    trenova_rls.apply_policy('public.mfa_authenticators', '(' || trenova_rls.standard_expr('organization_id', 'business_unit_id') || ') OR user_id = (SELECT trenova_rls.user_id())');

SELECT
    trenova_rls.apply_policy('public.mfa_recovery_codes', '(' || trenova_rls.standard_expr('organization_id', 'business_unit_id') || ') OR user_id = (SELECT trenova_rls.user_id())');
