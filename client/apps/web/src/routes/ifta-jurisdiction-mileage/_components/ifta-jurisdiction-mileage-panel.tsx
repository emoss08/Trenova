import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import {
  createIftaMileageEntry,
  IFTA_MILEAGE_ENTRY_LIST_KEY,
  updateIftaMileageEntry,
  type IftaMileageEntryRow,
} from "@/lib/graphql/ifta-jurisdiction-mileage";
import type { IftaMileageEntryInput } from "@trenova/graphql/generated/graphql";
import { getTodayDate } from "@trenova/shared/lib/date";
import { blankToNull } from "@trenova/shared/lib/utils";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  createIftaMileageEntryFormSchema,
  type IftaMileageEntryFormValues,
} from "@trenova/shared/types/ifta-jurisdiction-mileage";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMemo } from "react";
import { useForm, type Resolver } from "react-hook-form";
import { IftaJurisdictionMileageForm } from "./ifta-jurisdiction-mileage-form";

const SECONDS_PER_DAY = 86_400;

function endOfTodaySeconds(): number {
  return getTodayDate() + SECONDS_PER_DAY - 1;
}

export function isComputedEntry(row: Pick<IftaMileageEntryRow, "source">): boolean {
  return row.source !== "Manual";
}

export function buildIftaMileageEntryDefaults(
  row: IftaMileageEntryRow | null | undefined,
  today: number,
): IftaMileageEntryFormValues {
  if (!row) {
    return {
      tractorId: "",
      jurisdictionId: "",
      traveledAt: today,
      miles: "",
      loaded: true,
      notes: null,
    };
  }
  return {
    tractorId: row.tractorId,
    jurisdictionId: row.jurisdictionId,
    traveledAt: row.traveledAt,
    miles: row.miles,
    loaded: row.loaded,
    notes: row.notes ?? null,
  };
}

export function toIftaMileageEntryInput(
  values: IftaMileageEntryFormValues,
  existing?: Pick<IftaMileageEntryRow, "source" | "shipmentMoveId"> | null,
): IftaMileageEntryInput {
  return {
    tractorId: values.tractorId,
    jurisdictionId: values.jurisdictionId,
    traveledAt: values.traveledAt,
    miles: values.miles.trim(),
    loaded: values.loaded,
    source: existing?.source ?? "Manual",
    shipmentMoveId: existing?.shipmentMoveId ?? null,
    notes: blankToNull(values.notes),
  };
}

function useIftaMileageEntryResolver(): Resolver<IftaMileageEntryFormValues> {
  return useMemo<Resolver<IftaMileageEntryFormValues>>(
    () => (values, context, options) =>
      (
        zodResolver(
          createIftaMileageEntryFormSchema(endOfTodaySeconds()),
        ) as Resolver<IftaMileageEntryFormValues>
      )(values, context, options),
    [],
  );
}

export function IftaJurisdictionMileagePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<IftaMileageEntryRow>) {
  if (mode === "edit" && row) {
    return <IftaMileageEntryEditPanel open={open} onOpenChange={onOpenChange} row={row} />;
  }
  return <IftaMileageEntryCreatePanel open={open} onOpenChange={onOpenChange} />;
}

function IftaMileageEntryCreatePanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  const resolver = useIftaMileageEntryResolver();
  const form = useForm<IftaMileageEntryFormValues>({
    resolver,
    defaultValues: buildIftaMileageEntryDefaults(null, getTodayDate()),
  });

  return (
    <FormCreatePanel<IftaMileageEntryFormValues, IftaMileageEntryRow>
      open={open}
      onOpenChange={onOpenChange}
      title={t("Jurisdiction Mileage")}
      description={t("Record miles a tractor ran in a jurisdiction that routing did not see, such as repositioning between shipments. The entry lands on the quarter's return at its next recompute.")}
      queryKey={IFTA_MILEAGE_ENTRY_LIST_KEY}
      form={form}
      size="md"
      formComponent={<IftaJurisdictionMileageForm isEdit={false} computed={false} />}
      mutationFn={async (values) => {
        await createIftaMileageEntry(toIftaMileageEntryInput(values));
        return values;
      }}
    />
  );
}

function IftaMileageEntryEditPanel({
  open,
  onOpenChange,
  row,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  row: IftaMileageEntryRow;
}) {
  const t = useT();

  const resolver = useIftaMileageEntryResolver();
  const computed = isComputedEntry(row);
  const formRow = {
    ...row,
    ...buildIftaMileageEntryDefaults(row, getTodayDate()),
  } as unknown as IftaMileageEntryRow & Record<string, unknown>;
  const form = useForm<IftaMileageEntryFormValues>({
    resolver,
    defaultValues: buildIftaMileageEntryDefaults(row, getTodayDate()),
  });

  return (
    <FormEditPanel<IftaMileageEntryFormValues, IftaMileageEntryRow & Record<string, unknown>>
      open={open}
      onOpenChange={onOpenChange}
      row={formRow}
      title={t("Jurisdiction Mileage")}
      titleComponent={(record) => (
        <span>
          {t("{0} {1} · {2} mi", record.tractor?.code ? `${record.tractor.code} · ` : "", record.jurisdiction.code, record.miles)}
        </span>
      )}
      queryKey={IFTA_MILEAGE_ENTRY_LIST_KEY}
      form={form}
      size="md"
      formComponent={
        <IftaJurisdictionMileageForm
          isEdit
          computed={computed}
          source={row.source}
          shipmentMoveId={row.shipmentMoveId}
        />
      }
      mutationFn={async (values) => {
        await updateIftaMileageEntry(row.id, row.version, toIftaMileageEntryInput(values, row));
        return values;
      }}
    />
  );
}
