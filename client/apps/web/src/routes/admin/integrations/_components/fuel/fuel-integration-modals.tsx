import { Dialog, DialogContent } from "@trenova/shared/components/ui/dialog";
import type { TableSheetProps } from "@trenova/shared/types/data-table";
import { FuelFeedForm, type FuelFeedVendor } from "./fuel-feed-form";

/**
 * The two fleet networks gate production API access behind their own partner
 * agreements, so what a customer can actually turn on today is the transaction
 * export they already receive. The prerequisite copy says so plainly rather than
 * leaving somebody hunting for credentials that are not available to them.
 */

export const wexFuelVendor: FuelFeedVendor = {
  integrationType: "WEXFuel",
  logoLight: "/integrations/logos/wex-logo-light.png",
  logoDark: "/integrations/logos/wex-logo-dark.svg",
  name: "WEX",
  headline: "Connect WEX and EFS fuel cards",
  blurb: "Post fuel purchases and IFTA gallons from your",
  docsLabel: "WEX fleet card reporting.",
  docsUrl: "https://www.wexinc.com/products/business-payment-solutions/fleet-cards/",
  prerequisite:
    "Schedule a transaction export from your WEX or EFS account to an SFTP server, then point Trenova at it. WEX acquired EFS in 2016, so both card brands run on this one connection.",
};

export const comdataFuelVendor: FuelFeedVendor = {
  integrationType: "ComdataFuel",
  logoLight: "/integrations/logos/comdata-logo-light.png",
  logoDark: "/integrations/logos/comdata-logo-dark.png",
  name: "Comdata",
  headline: "Connect Comdata fuel cards",
  blurb: "Post fuel purchases and IFTA gallons from your",
  docsLabel: "Comdata iConnectData reports.",
  docsUrl: "https://www.comdata.com/",
  prerequisite:
    "Schedule a transaction report from iConnectData to an SFTP server, then point Trenova at it. Comdata's reconciliation reports are fixed width, so paste the record layout from their documentation into the layout box below.",
};

export const rampFuelVendor: FuelFeedVendor = {
  integrationType: "RampFuel",
  logoLight: "/integrations/logos/ramp-logo-light.svg",
  logoDark: "/integrations/logos/ramp-logo-dark.svg",
  name: "Ramp",
  headline: "Connect Ramp cards",
  blurb: "Read card transactions with the",
  docsLabel: "Ramp developer API.",
  docsUrl: "https://docs.ramp.com/developer-api/v1/overview/introduction",
  prerequisite:
    "Ramp runs on commercial Visa rather than a fleet network, so a transaction carries the merchant and the amount but not the gallons. Those rows wait in the review queue for somebody to add the pump detail before they can count toward IFTA.",
};

export function WEXFuelIntegrationModal({ open, onOpenChange }: TableSheetProps) {
  return <FuelFeedModal vendor={wexFuelVendor} open={open} onOpenChange={onOpenChange} />;
}

export function ComdataFuelIntegrationModal({ open, onOpenChange }: TableSheetProps) {
  return <FuelFeedModal vendor={comdataFuelVendor} open={open} onOpenChange={onOpenChange} />;
}

export function RampFuelIntegrationModal({ open, onOpenChange }: TableSheetProps) {
  return <FuelFeedModal vendor={rampFuelVendor} open={open} onOpenChange={onOpenChange} />;
}

function FuelFeedModal({
  vendor,
  open,
  onOpenChange,
}: TableSheetProps & { vendor: FuelFeedVendor }) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-md">
        <FuelFeedForm vendor={vendor} open={open} onClose={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  );
}
