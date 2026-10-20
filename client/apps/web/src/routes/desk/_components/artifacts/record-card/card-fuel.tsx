import { FuelCardMark } from "@/components/fuel-card-mark";
import { recordPath } from "@/config/record-links";
import type { DeskFuelCard } from "@/lib/graphql/fuel-desk-cards";
import { FuelCardStatusBadge } from "@trenova/shared/components/status-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  FUEL_CARD_PROVIDER_LABELS,
  type FuelCardProvider,
} from "@trenova/shared/types/fuel-ifta-enums";
import { CardIcon, CardRecordRow } from "./record-card";

/*
The parts the fuel purchase and fuel card cards share: the network's card in
small, and a row naming a card that opens it.
*/

/** The network's card in small, where a card's head leads with it; greyed once it is cancelled. */
export function FuelCardLead({
  provider,
  cancelled = false,
}: {
  provider: FuelCardProvider;
  cancelled?: boolean;
}) {
  return (
    <FuelCardMark
      provider={provider}
      className={cn("dk-fu-lead h-8.5 w-13 rounded-md px-1.5", cancelled && "dk-off")}
    />
  );
}

/** One fuel card as a row: its network, its label and last four, and where it stands. */
export function FuelCardRow({ card, onOpen }: { card: DeskFuelCard; onOpen?: () => void }) {
  const t = useT();
  return (
    <CardRecordRow className="dk-fu-cardrow" to={recordPath("fuel_card", card.id)} onOpen={onOpen}>
      <FuelCardMark provider={card.provider} />
      <span className="dk-rec-main">
        <b>{card.label}</b>
        <span className="dk-rec-sub">
          {t("{0} · ending {1}", FUEL_CARD_PROVIDER_LABELS[card.provider], card.lastFour)}
        </span>
      </span>
      <FuelCardStatusBadge status={card.status} />
      <CardIcon name="forward" size={12} />
    </CardRecordRow>
  );
}
