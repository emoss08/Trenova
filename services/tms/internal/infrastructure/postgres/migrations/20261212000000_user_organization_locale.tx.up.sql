--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- Language preference, alongside the timezone and time_format columns these tables already
-- carry for the same reason: a user's display settings belong with the user, and the worker
-- that renders their emails and documents has no request to read a header from.
--
-- Deliberately varchar rather than an enum. The catalogs are generated from
-- i18n/locales.json, and shipping a new language should be that one edit plus a catalog —
-- an enum type would add an ALTER TYPE migration to every future language. The accepted
-- values are validated in the domain against i18n.Supported(), which is generated from the
-- same file, so the check cannot drift from the catalogs the way a hand-written CHECK
-- constraint would.
--
-- organizations.locale is the default a new user inherits; users.locale is what actually
-- renders. Both default to en, which is what every existing row is today.
ALTER TABLE "organizations"
    ADD COLUMN IF NOT EXISTS "locale" varchar(10) NOT NULL DEFAULT 'en';

--bun:split
ALTER TABLE "users"
    ADD COLUMN IF NOT EXISTS "locale" varchar(10) NOT NULL DEFAULT 'en';
