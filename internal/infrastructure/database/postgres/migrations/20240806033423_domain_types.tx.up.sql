CREATE DOMAIN "us_postal_code" AS text CONSTRAINT "us_postal_code_check" CHECK ((VALUE ~ '^\d{5}$'::text)
    OR (VALUE ~ '^\d{5}-\d{4}$'::text));

CREATE DOMAIN "vin_code" AS varchar(17) CONSTRAINT "vin_code_check" CHECK (VALUE ~ '^[A-HJ-NPR-Z0-9]{17}$');

CREATE DOMAIN "temperature_fahrenheit" AS smallint CONSTRAINT "temperature_fahrenheit_check" CHECK (VALUE BETWEEN -100 AND 150);


CREATE DOMAIN "vin_code_optional" AS varchar(17) 
CONSTRAINT "vin_code_optional_check" CHECK (
    VALUE IS NULL OR VALUE = '' OR VALUE ~ '^[A-HJ-NPR-Z0-9]{17}$'
);