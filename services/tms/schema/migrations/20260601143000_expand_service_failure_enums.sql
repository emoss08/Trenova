ALTER TYPE "service_failure_source_enum" ADD VALUE IF NOT EXISTS 'EDI';
ALTER TYPE "service_failure_source_enum" ADD VALUE IF NOT EXISTS 'Integration';

ALTER TYPE "service_failure_type_enum" ADD VALUE IF NOT EXISTS 'MissedPickup';
ALTER TYPE "service_failure_type_enum" ADD VALUE IF NOT EXISTS 'MissedDelivery';
ALTER TYPE "service_failure_type_enum" ADD VALUE IF NOT EXISTS 'AppointmentMissed';
ALTER TYPE "service_failure_type_enum" ADD VALUE IF NOT EXISTS 'Other';

ALTER TYPE "service_failure_reason_applies_to_enum" ADD VALUE IF NOT EXISTS 'All';

ALTER TYPE "service_failure_reason_category_enum" ADD VALUE IF NOT EXISTS 'Driver';
ALTER TYPE "service_failure_reason_category_enum" ADD VALUE IF NOT EXISTS 'Shipper';
ALTER TYPE "service_failure_reason_category_enum" ADD VALUE IF NOT EXISTS 'Consignee';
ALTER TYPE "service_failure_reason_category_enum" ADD VALUE IF NOT EXISTS 'Appointment';
