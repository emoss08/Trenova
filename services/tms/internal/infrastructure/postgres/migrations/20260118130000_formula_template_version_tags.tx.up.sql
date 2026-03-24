ALTER TABLE formula_template_versions
ADD COLUMN IF NOT EXISTS tags TEXT[] DEFAULT '{}';
