--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- Enum values cannot be removed in place; rows using them are cleaned up by the
-- follow-on migration's down step and the values are left in the type.
SELECT 1;
