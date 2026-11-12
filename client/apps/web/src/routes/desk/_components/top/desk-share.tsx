import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { ShareDialog, type SharedRecord } from "@/components/share/share-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useLocation } from "react-router";

/**
 * Share what is in front on the Desk, with the dialog every share uses. A
 * case, or the item open in Needs you, is about a record: teammates can be
 * invited to it and the dialog remembers who was; being invited brings a
 * case they do not own into their view to read. A conversation or a list is
 * no one record, so only its link is offered. The Desk keeps what is being
 * looked at in the address, so the address is what the link opens.
 */
export function DeskShare({ record }: { record: SharedRecord | null }) {
  const t = useT();
  const location = useLocation();

  return (
    <ShareDialog
      record={record}
      path={`${location.pathname}${location.search}`}
      description={
        record
          ? t(
              "Send teammates a link to this. A case someone else owns comes into their view to read and ask about; its owner still decides it.",
            )
          : undefined
      }
      trigger={(open) => (
        <Button variant="outline" size="sm" onClick={open}>
          <DeskIcon name="link" size={13} />
          {t("Share")}
        </Button>
      )}
    />
  );
}
