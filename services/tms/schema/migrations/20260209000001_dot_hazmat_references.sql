CREATE TABLE dot_hazmat_references (
  id                  VARCHAR(100) PRIMARY KEY,
  un_number           VARCHAR(4) NOT NULL,
  proper_shipping_name TEXT NOT NULL,
  hazard_class        VARCHAR(20) NOT NULL,
  subsidiary_hazard   VARCHAR(50) DEFAULT '',
  packing_group       VARCHAR(20) DEFAULT '',
  special_provisions  TEXT DEFAULT '',
  packaging_exceptions VARCHAR(20) DEFAULT '',
  packaging_non_bulk  VARCHAR(20) DEFAULT '',
  packaging_bulk      VARCHAR(20) DEFAULT '',
  quantity_passenger  VARCHAR(50) DEFAULT '',
  quantity_cargo      VARCHAR(50) DEFAULT '',
  vessel_stowage      VARCHAR(50) DEFAULT '',
  erg_guide           VARCHAR(50) DEFAULT '',
  symbols             VARCHAR(20) DEFAULT '',
  created_at          BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
  updated_at          BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint
);

CREATE INDEX idx_dot_hazmat_ref_un_number ON dot_hazmat_references (un_number);
CREATE INDEX idx_dot_hazmat_ref_proper_shipping_name_trgm
  ON dot_hazmat_references USING gin (proper_shipping_name gin_trgm_ops);
