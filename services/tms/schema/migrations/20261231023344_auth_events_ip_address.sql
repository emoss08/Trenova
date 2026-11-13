-- Every other address the product records is text a person reads, and nothing
-- filters these with network operators. As inet the column decoded in binary
-- into a Go string, which pgx cannot do, so no auth event could be read.
ALTER TABLE "auth_events"
    ALTER COLUMN "ip_address" TYPE varchar(45) USING host("ip_address");
