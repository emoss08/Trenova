import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import {
  createPerformanceReviewTemplate,
  REVIEW_TEMPLATE_LIST_KEY,
  updatePerformanceReviewTemplate,
  type ReviewTemplateRow,
} from "@/lib/graphql/performance-review";
import type { PerformanceReviewTemplateInput } from "@trenova/graphql/generated/graphql";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  reviewTemplateFormSchema,
  type ReviewTemplateFormValues,
} from "@trenova/shared/types/performance-review";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, type Resolver } from "react-hook-form";
import { ReviewTemplateForm } from "./review-template-form";

export function buildReviewTemplateDefaults(
  row?: ReviewTemplateRow | null,
): ReviewTemplateFormValues {
  if (!row) {
    return {
      code: "",
      name: "",
      description: null,
      status: "Active",
      isDefault: false,
      cadenceMonths: 12,
      items: [{ key: "safety", label: translate("Safe driving"), description: null, weight: 3 }],
    };
  }
  return {
    code: row.code,
    name: row.name,
    description: row.description ?? null,
    status: row.status === "Inactive" ? "Inactive" : "Active",
    isDefault: row.isDefault,
    cadenceMonths: row.cadenceMonths ?? null,
    items: row.items.map((item) => ({
      key: item.key,
      label: item.label,
      description: item.description ?? null,
      weight: item.weight,
    })),
  };
}

export function toReviewTemplateInput(
  values: ReviewTemplateFormValues,
  version?: number,
): PerformanceReviewTemplateInput {
  return {
    code: values.code.toUpperCase(),
    name: values.name,
    description: values.description ?? undefined,
    status: values.status,
    isDefault: values.isDefault,
    cadenceMonths: values.cadenceMonths ?? undefined,
    items: values.items.map((item) => ({
      key: item.key,
      label: item.label,
      description: item.description ?? undefined,
      weight: item.weight,
    })),
    version,
  };
}

export function ReviewTemplatePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<ReviewTemplateRow>) {
  if (mode === "edit" && row) {
    return <ReviewTemplateEditPanel open={open} onOpenChange={onOpenChange} row={row} />;
  }
  return <ReviewTemplateCreatePanel open={open} onOpenChange={onOpenChange} />;
}

function ReviewTemplateCreatePanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  const form = useForm<ReviewTemplateFormValues>({
    resolver: zodResolver(reviewTemplateFormSchema) as Resolver<ReviewTemplateFormValues>,
    defaultValues: buildReviewTemplateDefaults(null),
  });

  return (
    <FormCreatePanel<ReviewTemplateFormValues, ReviewTemplateRow>
      open={open}
      onOpenChange={onOpenChange}
      title={t("Review Template")}
      description={t(
        "Decide what a review rates, how much each item counts, and how often the review comes round.",
      )}
      queryKey={REVIEW_TEMPLATE_LIST_KEY}
      form={form}
      size="lg"
      formComponent={<ReviewTemplateForm isEdit={false} />}
      mutationFn={async (values) => {
        await createPerformanceReviewTemplate(toReviewTemplateInput(values));
        return values;
      }}
    />
  );
}

function ReviewTemplateEditPanel({
  open,
  onOpenChange,
  row,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  row: ReviewTemplateRow;
}) {
  const t = useT();

  const formRow = {
    ...row,
    ...buildReviewTemplateDefaults(row),
  } as unknown as ReviewTemplateRow & Record<string, unknown>;
  const form = useForm<ReviewTemplateFormValues>({
    resolver: zodResolver(reviewTemplateFormSchema) as Resolver<ReviewTemplateFormValues>,
    defaultValues: buildReviewTemplateDefaults(row),
  });

  return (
    <FormEditPanel<ReviewTemplateFormValues, ReviewTemplateRow & Record<string, unknown>>
      open={open}
      onOpenChange={onOpenChange}
      row={formRow}
      title={t("Review Template")}
      fieldKey="code"
      queryKey={REVIEW_TEMPLATE_LIST_KEY}
      form={form}
      size="lg"
      formComponent={<ReviewTemplateForm isEdit openReviewCount={row.openReviewCount} />}
      mutationFn={async (values) => {
        await updatePerformanceReviewTemplate(row.id, toReviewTemplateInput(values, row.version));
        return values;
      }}
    />
  );
}
