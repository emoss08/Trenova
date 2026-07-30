-- The backfilled document types are only removed where nothing was filed under
-- them. A type that already has documents is a type an organization is using; a
-- rollback of a policy flag has no business deleting the filing cabinet.
DELETE FROM document_types dt
WHERE dt.id = 'dt_' || substr(md5(dt.organization_id || 'DETNOTICE'), 1, 26)
    AND dt.code = 'DETNOTICE'
    AND NOT EXISTS (
        SELECT
            1
        FROM
            documents d
        WHERE
            d.document_type_id = dt.id);

--bun:split
ALTER TABLE "detention_policies"
    DROP COLUMN IF EXISTS "attach_notice_pdf";
