INSERT INTO integrations (
    id,
    business_unit_id,
    organization_id,
    type,
    name,
    description,
    enabled,
    built_by,
    category,
    configuration,
    docs_url,
    featured,
    logo_url,
    website_url,
    enabled_by_id,
    version,
    created_at,
    updated_at
)
SELECT
    'intg_' || regexp_replace(id::text, '^aic_', ''),
    business_unit_id,
    organization_id,
    'OpenAI'::integration_type,
    'OpenAI',
    'AI-powered document classification and structured extraction for document intelligence workflows.',
    FALSE,
    'Trenova',
    'ArtificialIntelligence'::integration_category,
    jsonb_build_object('apiKey', api_key),
    'https://platform.openai.com/docs',
    TRUE,
    '/integrations/logos/openai.svg',
    'https://openai.com/',
    NULL,
    version,
    created_at,
    updated_at
FROM ai_configs
ON CONFLICT (organization_id, business_unit_id, type) DO UPDATE
SET
    configuration = EXCLUDED.configuration,
    updated_at = GREATEST(integrations.updated_at, EXCLUDED.updated_at);

--bun:split
DROP TABLE IF EXISTS ai_configs;

--bun:split
DROP FUNCTION IF EXISTS ai_configs_update_timestamp();
