import {
  HoverCardHeading,
  HoverCardRow,
  PointerHoverCard,
  usePointerHoverTarget,
} from "@/components/pointer-hover-card";
import type { DetentionFacility } from "@/lib/graphql/detention-intelligence";
import { useT } from "@trenova/shared/i18n/use-t";
import type { FocusEvent, PointerEvent, ReactNode } from "react";
import { CAUSES, CAUSE_LABELS, causeMinutes, hours } from "./intelligence-model";
import { useFacilityHighlight } from "./intelligence-parts";

/** A facility's hours by whose wait they were, then the total, as the mockup lists them. */
function FacilityDetail({ facility }: { facility: DetentionFacility }) {
  const t = useT();
  const total = facility.pastFreeMinutes;
  const pastHours = hours(total);

  return (
    <>
      <HoverCardHeading>{facility.locationName}</HoverCardHeading>
      {CAUSES.map((cause) => (
        <HoverCardRow
          key={cause.key}
          swatchClass={cause.bgClass}
          label={CAUSE_LABELS[cause.key]}
          value={t("{0} h", hours(causeMinutes(facility, cause.key)).toLocaleString())}
        />
      ))}
      <HoverCardRow
        className="mt-0.5"
        label={t("Past free time")}
        value={pastHours === 0 && total > 0 ? t("<1 h") : t("{0} h", pastHours.toLocaleString())}
      />
    </>
  );
}

/**
 * The one hover card for the facility marks inside it. Without `render` it lists the facility's
 * hours by cause; a chart that says something else of a facility passes its own.
 */
export function FacilityHoverCard({
  render,
  children,
}: {
  render?: (facility: DetentionFacility) => ReactNode;
  children: ReactNode;
}) {
  return (
    <PointerHoverCard<DetentionFacility>
      follow="pointer"
      itemId={(facility) => facility.locationId}
      render={render ?? ((facility) => <FacilityDetail facility={facility} />)}
    >
      {children}
    </PointerHoverCard>
  );
}

/**
 * A facility mark's hover: it lights the facility on every chart and opens its card. Both run on
 * the same pointer and focus events, so a mark spreads one set of handlers.
 */
export function useFacilityMark(facility: DetentionFacility) {
  const { attributes, handlers: highlight } = useFacilityHighlight(facility.locationId);
  const hover = usePointerHoverTarget(facility, facility.locationId);

  return {
    attributes,
    handlers: {
      onPointerEnter: (event: PointerEvent<Element>) => {
        highlight.onPointerEnter();
        hover.onPointerEnter(event);
      },
      onPointerMove: hover.onPointerMove,
      onPointerLeave: () => {
        highlight.onPointerLeave();
        hover.onPointerLeave();
      },
      onFocus: (event: FocusEvent<Element>) => {
        highlight.onFocus();
        hover.onFocus(event);
      },
      onBlur: () => {
        highlight.onBlur();
        hover.onBlur();
      },
    },
  };
}
