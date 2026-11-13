import { usePageTourStore } from "@/stores/page-tour-store";
import { Popover as PopoverPrimitive } from "@base-ui/react/popover";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * "Take the tour", for the page header's About note. It shows only while the page's tour can
 * run (its page is on screen and drawn), and closes the note it sits in as the tour opens.
 */
export function TourReplayButton({ tour }: { tour: string }) {
  const t = useT();
  const available = usePageTourStore((state) => state.available.includes(tour));
  const request = usePageTourStore((state) => state.request);

  if (!available) return null;

  return (
    <PopoverPrimitive.Close
      render={<Button size="xs" className="self-start" onClick={() => request(tour)} />}
    >
      {t("Take the tour")}
    </PopoverPrimitive.Close>
  );
}
