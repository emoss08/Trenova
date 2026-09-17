import { useT } from "@trenova/shared/i18n/use-t";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import type { AgentTemplate, AgentTemplateKind } from "@/types/assistant";
import {
  BotIcon,
  CheckIcon,
  FileInputIcon,
  HeadsetIcon,
  type LucideIcon,
  ReceiptTextIcon,
  ShieldCheckIcon,
  SlidersHorizontalIcon,
  TruckIcon,
  WalletCardsIcon,
} from "lucide-react";

export const TEMPLATE_ICONS: Record<AgentTemplateKind, LucideIcon> = {
  DispatchAssistant: TruckIcon,
  BillingAssistant: ReceiptTextIcon,
  ComplianceAssistant: ShieldCheckIcon,
  CustomerAssistant: HeadsetIcon,
  GeneralAssistant: BotIcon,
  BillingException: WalletCardsIcon,
  DispatchAssignment: TruckIcon,
  ImportAssistant: FileInputIcon,
};

type TemplatePickerProps = {
  templates: readonly AgentTemplate[];
  value: AgentTemplateKind | null;
  isLoading?: boolean;
  onSelect: (template: AgentTemplate | null) => void;
};

/**
 * Starting points. A template fills in instructions, tools and a trigger
 * for a common job; nothing it sets is locked, and "From scratch" leaves
 * every field as it is.
 */
export function TemplatePicker({
  templates,
  value,
  isLoading = false,
  onSelect,
}: TemplatePickerProps) {
  const t = useT();

  if (isLoading) {
    return (
      <div className="grid grid-cols-2 gap-2 md:grid-cols-3">
        {Array.from({ length: 6 }).map((_, index) => (
          <Skeleton key={index} className="h-16" />
        ))}
      </div>
    );
  }

  return (
    <div
      role="radiogroup"
      aria-label={t("Template")}
      className="grid grid-cols-2 gap-2 md:grid-cols-3"
    >
      <TemplateTile
        selected={value === null}
        icon={SlidersHorizontalIcon}
        label={t("From scratch")}
        caption={t("Write every instruction yourself")}
        onClick={() => onSelect(null)}
      />
      {templates.map((template) => (
        <TemplateTile
          key={template.template}
          selected={value === template.template}
          icon={TEMPLATE_ICONS[template.template] ?? BotIcon}
          label={template.label}
          caption={template.description}
          onClick={() => onSelect(template)}
        />
      ))}
    </div>
  );
}

function TemplateTile({
  selected,
  icon: Icon,
  label,
  caption,
  onClick,
}: {
  selected: boolean;
  icon: LucideIcon;
  label: string;
  caption: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      onClick={onClick}
      className={cn(
        "bg-card hover:bg-muted/50 focus-visible:ring-ring/50 relative flex items-start gap-2.5 rounded-lg border p-2.5 text-left transition-colors outline-none focus-visible:ring-[3px]",
        selected ? "border-primary ring-primary/20 ring-2" : "border-border",
      )}
    >
      <span
        className={cn(
          "flex size-8 shrink-0 items-center justify-center rounded-lg",
          selected ? "bg-primary/15 text-primary" : "bg-muted text-muted-foreground",
        )}
      >
        <Icon className="size-4" />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-medium">{label}</span>
        <span className="text-muted-foreground line-clamp-2 block text-[11px]">{caption}</span>
      </span>
      {selected && (
        <span className="bg-primary text-primary-foreground absolute top-1.5 right-1.5 flex size-4 items-center justify-center rounded-full">
          <CheckIcon className="size-3" />
        </span>
      )}
    </button>
  );
}
