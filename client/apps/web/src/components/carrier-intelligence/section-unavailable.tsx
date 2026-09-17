import { useT } from "@trenova/shared/i18n/use-t";
import { carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import { cn } from "@trenova/shared/lib/utils";
import { CircleSlashIcon } from "lucide-react";

export type SectionUnavailableProps = {
  provider: string | null | undefined;
  reason?: "not-covered" | "empty";
  message?: string;
  className?: string;
};

export function SectionUnavailable({
  provider,
  reason = "not-covered",
  message,
  className,
}: SectionUnavailableProps) {
  const t = useT();
  const providerName = carrierIntelProviderLabel(provider);

  const text =
    message ??
    (reason === "not-covered"
      ? t("Not provided by {0}", providerName)
      : t("{0} returned no data for this section", providerName));

  return (
    <div
      role="note"
      data-section-unavailable={reason}
      className={cn(
        "text-muted-foreground flex items-center gap-2 rounded-md border border-dashed px-3 py-3 text-xs",
        className,
      )}
    >
      <CircleSlashIcon className="size-3.5 shrink-0" aria-hidden />
      <span>{text}</span>
    </div>
  );
}
