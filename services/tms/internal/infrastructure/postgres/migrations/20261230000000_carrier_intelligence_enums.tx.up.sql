ALTER TYPE "integration_type" ADD VALUE IF NOT EXISTS 'CarrierOK';

--bun:split
ALTER TYPE "integration_type" ADD VALUE IF NOT EXISTS 'FMCSAQCMobile';

--bun:split
ALTER TYPE "integration_category" ADD VALUE IF NOT EXISTS 'CarrierCompliance';
