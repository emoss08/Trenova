import { useDeskStore } from "@/stores/desk-store";
import { useT } from "@trenova/shared/i18n/use-t";
import { PRIVACY_URL, TERMS_URL } from "@trenova/shared/lib/constants";
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

  if (seen) {
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
          {t("Make sure you agree to our")}{" "}
          <a href={TERMS_URL} target="_blank" rel="noreferrer">
            {t("terms")}
          </a>{" "}
          {t("and our")}{" "}
          <a href={PRIVACY_URL} target="_blank" rel="noreferrer">
            {t("privacy policy")}
          </a>
          .
        </span>
        <button type="button" className="dk-tnote-x" onClick={close} aria-label={t("Dismiss")}>
          <DeskIcon name="x" size={12} stroke={2.2} />
        </button>
      </div>
    </div>
  );
}
