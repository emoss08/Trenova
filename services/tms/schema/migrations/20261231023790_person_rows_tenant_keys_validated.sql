-- Validates the keys 20261231023770 added NOT VALID. VALIDATE CONSTRAINT holds SHARE UPDATE
-- EXCLUSIVE on the referring table and ROW SHARE on the referenced one, so reads and writes go on
-- while each table is checked.

ALTER TABLE "user_role_assignments" VALIDATE CONSTRAINT "fk_user_role_assignments_business_unit";
ALTER TABLE "user_role_assignments" VALIDATE CONSTRAINT "tk_80defa39a224dc9fb1c525f2";
ALTER TABLE "user_role_assignments" VALIDATE CONSTRAINT "tk_857bef43d07b52ed5702ad7d";
ALTER TABLE "user_role_assignments" VALIDATE CONSTRAINT "tk_700e10ca42bf930a785b5549";
ALTER TABLE "mfa_authenticators" VALIDATE CONSTRAINT "fk_mfa_authenticators_business_unit";
ALTER TABLE "mfa_authenticators" VALIDATE CONSTRAINT "tk_fd4147c0d23417a221516c3a";
ALTER TABLE "mfa_authenticators" VALIDATE CONSTRAINT "tk_e4891f3a11f5859fe64dcf52";
ALTER TABLE "mfa_recovery_codes" VALIDATE CONSTRAINT "fk_mfa_recovery_codes_business_unit";
ALTER TABLE "mfa_recovery_codes" VALIDATE CONSTRAINT "tk_f0bf252abab7056ce9aa5215";
ALTER TABLE "mfa_recovery_codes" VALIDATE CONSTRAINT "tk_aed69198d06e38b0729d9648";
