import type { PreviewMessage } from "@/lib/graphql/agent-preview";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { channelLabel, visibilityLabel } from "./preview-format";

function Recipients({ addresses }: { addresses: readonly string[] }) {
  return addresses.length === 0 ? (
    <DescriptionEmpty />
  ) : (
    <span className="break-words">{addresses.join(", ")}</span>
  );
}

/**
 * A message as it would go out: rendered from the organization's template,
 * addressed to the people it would actually reach. The agent's own draft can
 * name an address the send would not use; this is what the send would do.
 */
export function MessagePreview({
  message,
  density = "full",
}: {
  message: PreviewMessage;
  density?: "compact" | "full";
}) {
  const t = useT();

  return (
    <div className="flex min-w-0 flex-col gap-2">
      <DescriptionList layout="inline">
        <DescriptionItem label={t("Channel")}>{channelLabel(message.channel, t)}</DescriptionItem>
        {message.from !== "" && <DescriptionItem label={t("From")}>{message.from}</DescriptionItem>}
        <DescriptionItem label={t("To")}>
          <Recipients addresses={message.to} />
        </DescriptionItem>
        {message.cc.length > 0 && (
          <DescriptionItem label={t("Cc")}>
            <Recipients addresses={message.cc} />
          </DescriptionItem>
        )}
        {message.bcc.length > 0 && (
          <DescriptionItem label={t("Bcc")}>
            <Recipients addresses={message.bcc} />
          </DescriptionItem>
        )}
        {message.subject !== "" && (
          <DescriptionItem label={t("Subject")}>{message.subject}</DescriptionItem>
        )}
        {message.visibility !== "" && (
          <DescriptionItem label={t("Seen by")}>
            {visibilityLabel(message.visibility, t)}
          </DescriptionItem>
        )}
        {message.cadence !== "" && (
          <DescriptionItem label={t("Repeats")}>{message.cadence}</DescriptionItem>
        )}
        {message.attachments.length > 0 && (
          <DescriptionItem label={t("Attachments")}>
            <span className="break-words">{message.attachments.join(", ")}</span>
          </DescriptionItem>
        )}
      </DescriptionList>

      {message.body !== "" ? (
        <div
          className={cn(
            "bg-sunken overflow-y-auto rounded-lg px-3 py-2.5 text-sm leading-relaxed break-words whitespace-pre-wrap",
            density === "compact" ? "max-h-40" : "max-h-96",
          )}
        >
          {message.body}
        </div>
      ) : (
        <p className="text-foreground-muted text-xs">{t("The message has no body.")}</p>
      )}
      {message.bodyTruncated && (
        <p className="text-foreground-subtle text-xs">
          {t("The message is longer than a preview keeps; the rest is not shown.")}
        </p>
      )}
    </div>
  );
}
