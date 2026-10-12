import { useDeskStore } from "@/stores/desk-store";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { LegalLinks, hasLegalUrls } from "@/components/legal-links";
import { useLegalUrls } from "@/hooks/use-legal-urls";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { DeskIcon } from "./desk-icons";

/** How long the note takes to fade before it leaves. */
const LEAVE_MS = 220;

/**
 * A one-time reminder about the terms above the composer. Closing it, or
 * sending a first message, puts it away for good.
 */
export function DeskTermsNote() {
  const t = useT();
  const seen = useDeskStore((state) => state.termsSeen);
  const markTermsSeen = useDeskStore((state) => state.markTermsSeen);
  const [leaving, setLeaving] = useState(false);
  const urls = useLegalUrls();

  if (seen || !hasLegalUrls(urls)) {
    return null;
  }

  const close = () => {
    setLeaving(true);
    window.setTimeout(markTermsSeen, LEAVE_MS);
  };

  return (
    <div className="dk-tnote-w">
      <div className={cn("dk-tnote", leaving && "dk-out")}>
        <span>
          {t("Make sure you agree to our")} <LegalLinks urls={urls} />.
        </span>
        <Button
          variant="bare"
          size="bare"
          className="dk-tnote-x absolute top-1/2 right-1.5 size-5.5 -translate-y-1/2 justify-center rounded-md text-inherit opacity-65 transition-[opacity,background-color] duration-120 hover:opacity-100"
          onClick={close}
          aria-label={t("Dismiss")}
        >
          <DeskIcon name="x" size={12} stroke={2.2} />
        </Button>
      </div>
    </div>
  );
}
