CREATE TABLE
    IF NOT EXISTS "locations"
(
    "id"                   uuid         NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id"     uuid         NOT NULL,
    "organization_id"      uuid         NOT NULL,
    "status"               status_enum  NOT NULL DEFAULT 'Active',
    "code"                 VARCHAR(10)  NOT NULL,
    "name"                 VARCHAR(255) NOT NULL,
    "address_line_1"       VARCHAR(150),
    "address_line_2"       VARCHAR(150),
    "city"                 VARCHAR(150),
    "state_id"             uuid,
    "postal_code"          VARCHAR(10),
    "longitude"            FLOAT,
    "latitude"             FLOAT,
    "place_id"             VARCHAR(255),
    "description"          TEXT,
    "is_geocoded"          BOOLEAN      NOT NULL DEFAULT FALSE,
    "location_category_id" uuid         NOT NULL,
    "version"              bigint       NOT NULL,
    "created_at"           TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    "updated_at"           TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("state_id") REFERENCES us_states ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("location_category_id") REFERENCES location_categories ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "locations_code_organization_id_unq" ON "locations" (LOWER("code"), organization_id);
CREATE INDEX idx_location_name ON locations (name);
CREATE INDEX idx_location_org_bu ON locations (organization_id, business_unit_id);
CREATE INDEX idx_location_description ON locations USING GIN (description gin_trgm_ops);
CREATE INDEX idx_location_created_at ON locations (created_at);

-- bun:split

COMMENT ON COLUMN locations.id IS 'Unique identifier for the location, generated as a UUID';
COMMENT ON COLUMN locations.business_unit_id IS 'Foreign key referencing the business unit that this location belongs to';
COMMENT ON COLUMN locations.organization_id IS 'Foreign key referencing the organization that this location belongs to';
COMMENT ON COLUMN locations.status IS 'The current status of the location, using the status_enum (e.g., Active, Inactive)';
COMMENT ON COLUMN locations.code IS 'A short, unique code for identifying the location, limited to 10 characters';
COMMENT ON COLUMN locations.name IS 'The name of the location';
COMMENT ON COLUMN locations.address_line_1 IS 'The first line of the address for the location';
COMMENT ON COLUMN locations.address_line_2 IS 'The second line of the address for the location';
COMMENT ON COLUMN locations.city IS 'The city where the location is located';
COMMENT ON COLUMN locations.postal_code IS 'The postal code for the location';
COMMENT ON COLUMN locations.longitude IS 'The longitude of the location';
COMMENT ON COLUMN locations.latitude IS 'The latitude of the location';
COMMENT ON COLUMN locations.place_id IS 'The Google Maps place ID for the location';
COMMENT ON COLUMN locations.description IS 'A detailed description of the location';
COMMENT ON COLUMN locations.is_geocoded IS 'A flag indicating whether the location has been geocoded';
COMMENT ON COLUMN locations.state_id IS 'Foreign key referencing the state where the location is located';
COMMENT ON COLUMN locations.location_category_id IS 'Foreign key referencing the category of the location';
COMMENT ON COLUMN locations.created_at IS 'Timestamp of when the location was created, defaults to the current timestamp';
COMMENT ON COLUMN locations.updated_at IS 'Timestamp of the last update to the location, defaults to the current timestamp';

-- bun:split

CREATE TABLE
    IF NOT EXISTS "location_comments"
(
    "id"               uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id" uuid        NOT NULL,
    "organization_id"  uuid        NOT NULL,
    "location_id"      uuid        NOT NULL,
    "user_id"          uuid        NOT NULL,
    "comment_type_id"  uuid        NOT NULL,
    "comment"          TEXT        NOT NULL,
    "version"          bigint      NOT NULL,
    "created_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("location_id") REFERENCES locations ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("user_id") REFERENCES users ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("comment_type_id") REFERENCES comment_types ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- bun:split

CREATE INDEX idx_location_comment_comment_type_id ON location_comments (comment_type_id);
CREATE INDEX idx_location_comment_created_at ON location_comments (created_at);

-- bun:split

COMMENT ON COLUMN location_comments.id IS 'Unique identifier for the location comment, generated as a UUID';
COMMENT ON COLUMN location_comments.business_unit_id IS 'Foreign key referencing the business unit that this location comment belongs to';
COMMENT ON COLUMN location_comments.organization_id IS 'Foreign key referencing the organization that this location comment belongs to';
COMMENT ON COLUMN location_comments.location_id IS 'Foreign key referencing the location that this comment is associated with';
COMMENT ON COLUMN location_comments.user_id IS 'Foreign key referencing the user that created the comment';
COMMENT ON COLUMN location_comments.comment_type_id IS 'Foreign key referencing the type of comment';
COMMENT ON COLUMN location_comments.comment IS 'The comment text';
COMMENT ON COLUMN location_comments.created_at IS 'Timestamp of when the comment was created, defaults to the current timestamp';
COMMENT ON COLUMN location_comments.updated_at IS 'Timestamp of the last update to the comment, defaults to the current timestamp';

-- bun:split

CREATE TABLE
    IF NOT EXISTS "location_contacts"
(
    "id"               uuid         NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id" uuid         NOT NULL,
    "organization_id"  uuid         NOT NULL,
    "location_id"      uuid         NOT NULL,
    "name"             VARCHAR(255) NOT NULL,
    "email_address"    VARCHAR(255),
    "phone_number"     VARCHAR(20),
    "version"          bigint      NOT NULL,
    "created_at"       TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("location_id") REFERENCES locations ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- bun:split

CREATE INDEX idx_location_contact_name ON location_contacts (name);
CREATE INDEX idx_location_contact_created_at ON location_contacts (created_at);

-- bun:split

COMMENT ON COLUMN location_contacts.id IS 'Unique identifier for the location contact, generated as a UUID';
COMMENT ON COLUMN location_contacts.business_unit_id IS 'Foreign key referencing the business unit that this location contact belongs to';
COMMENT ON COLUMN location_contacts.organization_id IS 'Foreign key referencing the organization that this location contact belongs to';
COMMENT ON COLUMN location_contacts.location_id IS 'Foreign key referencing the location that this contact is associated with';
COMMENT ON COLUMN location_contacts.name IS 'The name of the contact';
COMMENT ON COLUMN location_contacts.email_address IS 'The email address of the contact';
COMMENT ON COLUMN location_contacts.phone_number IS 'The phone number of the contact';
COMMENT ON COLUMN location_contacts.created_at IS 'Timestamp of when the contact was created, defaults to the current timestamp';
COMMENT ON COLUMN location_contacts.updated_at IS 'Timestamp of the last update to the contact, defaults to the current timestamp';