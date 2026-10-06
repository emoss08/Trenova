import { useT } from "@trenova/shared/i18n/use-t";
import { SelectField } from "@/components/fields/select-field";
import type { SelectOption } from "@trenova/shared/types/fields";
import type { AgentTemplate, AgentTemplateKind } from "@/types/assistant";
import {
  AlarmClockOffIcon,
  BankNote01Icon,
  BarChart07Icon,
  Bell01Icon,
  BookCheckIcon,
  BotIcon,
  Building08Icon,
  ClipboardCheckIcon,
  ClipboardListIcon,
  CoinsHandIcon,
  Compass03Icon,
  Database01Icon,
  FileAlertIcon,
  FileDownload02Icon,
  FuelIcon,
  HeadsetIcon,
  type IconComponent,
  Inbox01Icon,
  PackagePlusIcon,
  RadarIcon,
  ReceiptTextIcon,
  RouteIcon,
  SearchLgIcon,
  ShieldTickIcon,
  SigmaIcon,
  Speedometer03Icon,
  Truck01Icon,
  Users01Icon,
  Wallet01Icon,
} from "@trenova/shared/components/icons";
import { useMemo } from "react";
import type { Control } from "react-hook-form";
import type { AgentFormValues } from "./agent-form-schema";

export const TEMPLATE_ICONS: Record<AgentTemplateKind, IconComponent> = {
  DispatchAssistant: Truck01Icon,
  BillingAssistant: ReceiptTextIcon,
  ComplianceAssistant: ShieldTickIcon,
  CustomerAssistant: HeadsetIcon,
  GeneralAssistant: BotIcon,
  BillingException: Wallet01Icon,
  DispatchAssignment: RouteIcon,
  ImportAssistant: FileDownload02Icon,
  LoadMonitor: RadarIcon,
  ShipmentIntake: PackagePlusIcon,
  CashApplication: Building08Icon,
  DetentionDesk: Speedometer03Icon,
  CredentialDesk: ClipboardCheckIcon,
  CustomerUpdateDesk: Bell01Icon,
  CarrierRiskDesk: SearchLgIcon,
  IntakeDesk: Inbox01Icon,
  LoadEntryCheck: ClipboardListIcon,
  ServiceFailureDesk: AlarmClockOffIcon,
  InsightAnalyst: Compass03Icon,
  EDIDesk: FileAlertIcon,
  FormulaAssistant: SigmaIcon,
  BooksKeeper: BookCheckIcon,
  SettlementsClerk: BankNote01Icon,
  Receivables: CoinsHandIcon,
  MasterDataSteward: Database01Icon,
  WorkforceCoordinator: Users01Icon,
  FuelTaxClerk: FuelIcon,
  ReportAnalyst: BarChart07Icon,
};

type TemplatePickerProps = {
  control: Control<AgentFormValues>;
  templates: readonly AgentTemplate[];
  isLoading?: boolean;
  onSelect: (template: AgentTemplate | null) => void;
};

/**
 * Where an agent starts from.
 *
 * One field rather than nine tiles. A starter fills in instructions, tools and
 * a trigger for a common job and locks nothing, so it is a convenience, not a
 * decision worth the top third of the form.
 */
export function TemplatePicker({
  control,
  templates,
  isLoading = false,
  onSelect,
}: TemplatePickerProps) {
  const t = useT();

  // No "from scratch" row: an empty field already means it, and clearing the
  // field is the same gesture as picking that row would have been.
  const options = useMemo<SelectOption[]>(
    () =>
      templates.map((template) => {
        const Icon = TEMPLATE_ICONS[template.template] ?? BotIcon;

        return {
          value: template.template,
          label: template.label,
          description: template.description,
          icon: <Icon className="text-muted-foreground size-4 shrink-0" />,
        };
      }),
    [templates],
  );

  return (
    <SelectField
      control={control}
      name="template"
      label={t("Start from")}
      placeholder={isLoading ? t("Loading starters…") : t("From scratch")}
      isReadOnly={isLoading}
      isClearable
      options={options}
      description={t(
        "A starter fills in instructions, tools and a trigger you can change freely. It never limits what the agent may do.",
      )}
      onValueChange={(value) =>
        onSelect(templates.find((template) => template.template === value) ?? null)
      }
    />
  );
}
