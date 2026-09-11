import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { TCASubscriptionRow } from "@/lib/graphql/table-change-alert-table";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  tcaSubscriptionFormSchema,
  type TCASubscription,
  type TCASubscriptionFormValues,
} from "@/types/table-change-alert";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { SubscriptionForm } from "./subscription-form";

const DEFAULT_VALUES: TCASubscriptionFormValues = {
  name: "",
  tableName: "",
  recordId: "",
  eventTypes: ["INSERT", "UPDATE", "DELETE"],
  conditions: [],
  conditionMatch: "all",
  watchedColumns: [],
  customTitle: "",
  customMessage: "",
  topic: "",
  priority: "medium",
  status: "Active",
};

export function SubscriptionPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<TCASubscriptionRow>) {
  const t = useT();

  const form = useForm<TCASubscriptionFormValues>({
    resolver: zodResolver(tcaSubscriptionFormSchema),
    defaultValues: DEFAULT_VALUES,
  });

  if (mode === "edit") {
    return (
      <FormEditPanel<TCASubscriptionFormValues, TCASubscriptionRow>
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        url="/tca/subscriptions/"
        queryKey="tca-subscription-list"
        title={t("Subscription")}
        fieldKey="name"
        size="lg"
        formComponent={<SubscriptionForm />}
      />
    );
  }

  return (
    <FormCreatePanel<TCASubscriptionFormValues, TCASubscription>
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/tca/subscriptions/"
      queryKey="tca-subscription-list"
      title={t("Subscription")}
      size="lg"
      formComponent={<SubscriptionForm />}
    />
  );
}
