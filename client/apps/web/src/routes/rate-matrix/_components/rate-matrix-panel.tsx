import { useT } from "@trenova/shared/i18n/use-t";
import { TabbedFormCreatePanel } from "@/components/tabbed-form-create-panel";
import { TabbedFormEditPanel } from "@/components/tabbed-form-edit-panel";
import { useEditRecordReset } from "@/hooks/use-edit-record-reset";
import type { RateMatrixRow } from "@/lib/graphql/rate-tables";
import { rateMatrix } from "@/lib/queries/rate-matrix";
import { apiService } from "@/services/api";
import { useQueryClient } from "@tanstack/react-query";
import { zodResolver } from "@hookform/resolvers/zod";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { rateMatrixSchema, type RateMatrix } from "@trenova/shared/types/rate";
import {
  ClockRewindIcon,
  File06Icon,
  Grid01Icon,
  TableIcon,
} from "@trenova/shared/components/icons";
import { useMemo } from "react";
import { useForm, type Resolver } from "react-hook-form";
import { DimensionEditor } from "./dimension-editor";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { MatrixGridEditor } from "./matrix-grid-editor";
import { MatrixGridEditorSkeleton } from "./matrix-grid-editor-skeleton";
import { RateMatrixForm } from "./rate-matrix-form";
import { RateMatrixVersions } from "./rate-matrix-versions";

const DEFAULT_MATRIX: Partial<RateMatrix> = {
  code: "",
  name: "",
  description: "",
  status: "Draft",
  valueType: "Money",
  dimensions: [],
};

export function RateMatrixPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<RateMatrixRow>) {
  const t = useT();
  const queryClient = useQueryClient();

  const form = useForm<RateMatrix>({
    resolver: zodResolver(rateMatrixSchema) as Resolver<RateMatrix>,
    defaultValues: DEFAULT_MATRIX as RateMatrix,
    mode: "onChange",
  });

  // The table row carries no dimensions; without the full record the axes tab
  // opens empty, and saving that emptiness would strip the grid of its axes.
  const { isSeated, isLoading, isError } = useEditRecordReset(form, {
    open,
    mode,
    queryKey: "rate-matrix",
    id: row?.id,
    version: row?.version,
    fetch: (id) => apiService.rateMatrixService.getById(id),
  });

  const formTabs = useMemo(
    () => [
      {
        value: "overview",
        label: t("Overview"),
        icon: File06Icon,
        content: <RateMatrixForm />,
      },
      {
        value: "axes",
        label: t("Axes"),
        icon: Grid01Icon,
        content: <DimensionEditor />,
      },
      {
        value: "rates",
        label: t("Rates"),
        icon: TableIcon,
        // The sheet's coverage checks run in the rules module.
        content: (
          <LazyComponent fallback={<MatrixGridEditorSkeleton />}>
            <MatrixGridEditor rateMatrixId={row?.id} />
          </LazyComponent>
        ),
      },
      ...(mode === "edit"
        ? [
            {
              value: "versions",
              label: t("Versions"),
              icon: ClockRewindIcon,
              content: <RateMatrixVersions rateMatrixId={row?.id} />,
            },
          ]
        : []),
    ],
    [mode, row?.id, t],
  );

  if (mode === "edit") {
    return (
      <TabbedFormEditPanel<RateMatrix, RateMatrixRow>
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        size="xl"
        queryKey="rate-matrix-list"
        title={t("Rate matrix")}
        fieldKey="name"
        formTabs={formTabs}
        isRecordLoading={isLoading || !isSeated}
        recordFailed={isError}
        mutationFn={async (values, currentRow) => {
          if (!currentRow.id) {
            throw new Error("No Rate Matrix ID selected");
          }

          const saved = await apiService.rateMatrixService.update(currentRow.id, values);
          // A change to the draft moves its review state, which the Versions tab shows.
          void queryClient.invalidateQueries({ queryKey: rateMatrix._def });
          return saved;
        }}
      />
    );
  }

  return (
    <TabbedFormCreatePanel<RateMatrix, RateMatrix>
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      size="xl"
      queryKey="rate-matrix-list"
      title={t("Rate matrix")}
      description={t(
        "Enter a published tariff as the grid it was published as, and point any lane at it.",
      )}
      formTabs={formTabs}
      mutationFn={(values) => apiService.rateMatrixService.create(values)}
    />
  );
}
