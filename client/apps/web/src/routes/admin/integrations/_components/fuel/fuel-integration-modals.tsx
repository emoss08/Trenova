import { Dialog, DialogContent } from "@trenova/shared/components/ui/dialog";
import type { TableSheetProps } from "@trenova/shared/types/data-table";
import { FuelFeedForm, type FuelFeedVendor } from "./fuel-feed-form";
import { translateRich } from "@trenova/shared/i18n/rich";
import { translate } from "@trenova/shared/i18n/runtime";

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
  get headline() {
    return translate("Connect WEX and EFS fuel cards");
  },
  docs: (link) =>
    translateRich(
      "Post fuel purchases and IFTA gallons from your <link>WEX fleet card reporting.</link>",
      { link },
    ),
  docsUrl: "https://www.wexinc.com/products/business-payment-solutions/fleet-cards/",
  get prerequisite() {
    return translate(
      "Schedule a transaction export from your WEX or EFS account to an SFTP server, then point Trenova at it. WEX acquired EFS in 2016, so both card brands run on this one connection.",
    );
  },
};

export const comdataFuelVendor: FuelFeedVendor = {
  integrationType: "ComdataFuel",
  logoLight: "/integrations/logos/comdata-logo-light.png",
  logoDark: "/integrations/logos/comdata-logo-dark.png",
  name: "Comdata",
  get headline() {
    return translate("Connect Comdata fuel cards");
  },
  docs: (link) =>
    translateRich(
      "Post fuel purchases and IFTA gallons from your <link>Comdata iConnectData reports.</link>",
      { link },
    ),
  docsUrl: "https://www.comdata.com/",
  get prerequisite() {
    return translate(
      "Schedule a transaction report from iConnectData to an SFTP server, then point Trenova at it. Comdata's reconciliation reports are fixed width, so paste the record layout from their documentation into the layout box below.",
    );
  },
};

export const rampFuelVendor: FuelFeedVendor = {
  integrationType: "RampFuel",
  logoLight: "/integrations/logos/ramp-logo-light.svg",
  logoDark: "/integrations/logos/ramp-logo-dark.svg",
  name: "Ramp",
  get headline() {
    return translate("Connect Ramp cards");
  },
  docs: (link) =>
    translateRich("Read card transactions with the <link>Ramp developer API.</link>", { link }),
  docsUrl: "https://docs.ramp.com/developer-api/v1/overview/introduction",
  get prerequisite() {
    return translate(
      "Ramp runs on commercial Visa rather than a fleet network, so a transaction carries the merchant and the amount but not the gallons. Those rows wait in the review queue for somebody to add the pump detail before they can count toward IFTA.",
    );
  },
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
      <DialogContent className="max-h-[85vh] overflow-y-auto">
        <FuelFeedForm vendor={vendor} open={open} onClose={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  );
}
