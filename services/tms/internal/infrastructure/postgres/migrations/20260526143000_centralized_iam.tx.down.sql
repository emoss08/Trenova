DROP TABLE IF EXISTS access_policies;
DROP TABLE IF EXISTS provisioning_audit_records;
DROP TABLE IF EXISTS scim_group_role_mappings;
DROP TABLE IF EXISTS scim_tokens;
DROP TABLE IF EXISTS scim_directories;
DROP TABLE IF EXISTS auth_events;
DROP TABLE IF EXISTS risk_decisions;
DROP TABLE IF EXISTS mfa_authenticators;
DROP TABLE IF EXISTS external_identities;
DROP TABLE IF EXISTS identity_providers;

DROP TYPE IF EXISTS iam_policy_effect_enum;
DROP TYPE IF EXISTS iam_provisioning_action_enum;
DROP TYPE IF EXISTS iam_scim_token_status_enum;
DROP TYPE IF EXISTS iam_risk_outcome_enum;
DROP TYPE IF EXISTS iam_auth_event_outcome_enum;
DROP TYPE IF EXISTS iam_mfa_authenticator_type_enum;
DROP TYPE IF EXISTS iam_identity_provider_protocol_enum;
