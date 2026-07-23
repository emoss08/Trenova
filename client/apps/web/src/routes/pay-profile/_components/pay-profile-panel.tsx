import { FormCreatePanel } from "@/components/form-create-panel";
import { AssignPayProfileDialog } from "@/components/pay/assign-pay-profile-dialog";
import { TabbedFormEditPanel } from "@/components/tabbed-form-edit-panel";
import { Button } from "@/components/ui/button";
import {
  createPayProfile,
  fetchPayProfileAssignments,
  updatePayProfile,
  type PayProfileRow,
} from "@/lib/graphql/driver-settlement";
import type { DataTablePanelProps } from "@/types/data-table";
import {
  payProfileFormSchema,
  type PayProfileComponentFormValues,
  type PayProfileFormValues,
} from "@/types/driver-pay";
import type { CreatePayProfileInput, PayProfileComponentInput } from "@trenova/graphql/generated/graphql";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { UserPlus, UsersIcon, WalletIcon } from "lucide-react";
import { useState } from "react";
import { useForm, type Resolver } from "react-hook-form";
import { PayProfileForm } from "./pay-profile-form";

function defaultComponent(): PayProfileComponentFormValues {
  return {
    kind: "Linehaul",
    method: "PerLoadedMile",
    description: "",
    rate: "",
    revenueBasis: null,
    bands: [],
    freeTimeMinutes: 120,
    minAmount: null,
    maxAmount: null,
    isActive: true,
  };
}

function buildDefaults(row?: PayProfileRow | null): PayProfileFormValues {
  if (!row) {
    return {
      status: "Active",
      name: "",
      description: "",
      classification: "CompanyDriver",
      guaranteedPeriodMinimum: null,
      perDiemRatePerMile: "",
      perDiemDailyCap: null,
      components: [defaultComponent()],
    };
  }
  return {
    status: row.status === "Inactive" ? "Inactive" : "Active",
    name: row.name,
    description: row.description ?? "",
    classification: row.classification,
    guaranteedPeriodMinimum:
      row.guaranteedPeriodMinimumMinor > 0 ? row.guaranteedPeriodMinimumMinor / 100 : null,
    perDiemRatePerMile: Number(row.perDiemRatePerMile) > 0 ? String(row.perDiemRatePerMile) : "",
    perDiemDailyCap: row.perDiemDailyCapMinor > 0 ? row.perDiemDailyCapMinor / 100 : null,
    components: (row.components ?? []).map((component) => ({
      kind: component.kind,
      method: component.method,
      description: component.description ?? "",
      rate: String(component.rate),
      revenueBasis: component.revenueBasis ?? null,
      bands: (component.bands ?? []).map((band) => ({
        minMiles: band.minMiles,
        maxMiles: band.maxMiles,
        rate: String(band.rate),
      })),
      freeTimeMinutes: component.freeTimeMinutes,
      minAmount: component.minAmountMinor != null ? component.minAmountMinor / 100 : null,
      maxAmount: component.maxAmountMinor != null ? component.maxAmountMinor / 100 : null,
      isActive: component.isActive,
    })),
  };
}

function toComponentInputs(
  components: PayProfileComponentFormValues[],
): PayProfileComponentInput[] {
  return components.map((component) => ({
    kind: component.kind,
    method: component.method,
    description: component.description || undefined,
    rate: component.rate,
    revenueBasis:
      component.method === "PercentOfRevenue" ? (component.revenueBasis ?? undefined) : undefined,
    bands:
      component.bands && component.bands.length > 0
        ? component.bands.map((band) => ({
            minMiles: band.minMiles,
            maxMiles: band.maxMiles,
            rate: band.rate,
          }))
        : undefined,
    freeTimeMinutes: component.freeTimeMinutes ?? 0,
    minAmountMinor: component.minAmount != null ? Math.round(component.minAmount * 100) : undefined,
    maxAmountMinor: component.maxAmount != null ? Math.round(component.maxAmount * 100) : undefined,
    isActive: component.isActive,
  }));
}

function toCreateInput(values: PayProfileFormValues): CreatePayProfileInput {
  return {
    status: values.status,
    name: values.name,
    description: values.description || undefined,
    classification: values.classification,
    guaranteedPeriodMinimumMinor:
      values.guaranteedPeriodMinimum != null
        ? Math.round(values.guaranteedPeriodMinimum * 100)
        : undefined,
    perDiemRatePerMile: values.perDiemRatePerMile || undefined,
    perDiemDailyCapMinor:
      values.perDiemDailyCap != null ? Math.round(values.perDiemDailyCap * 100) : undefined,
    components: toComponentInputs(values.components),
  };
}

export function PayProfilePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<PayProfileRow>) {
  if (mode === "edit" && row) {
    return <PayProfileEditPanel open={open} onOpenChange={onOpenChange} row={row} />;
  }
  return <PayProfileCreatePanel open={open} onOpenChange={onOpenChange} />;
}

function PayProfileCreatePanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const form = useForm<PayProfileFormValues>({
    resolver: zodResolver(payProfileFormSchema) as Resolver<PayProfileFormValues>,
    defaultValues: buildDefaults(null),
  });

  return (
    <FormCreatePanel<PayProfileFormValues, PayProfileRow>
      open={open}
      onOpenChange={onOpenChange}
      title="Pay Profile"
      description="A reusable pay package; assign it to drivers and add per-driver rate overrides where rates differ."
      queryKey="pay-profile-list"
      form={form}
      size="xl"
      formComponent={<PayProfileForm />}
      mutationFn={async (values) => {
        await createPayProfile(toCreateInput(values));
        return values;
      }}
    />
  );
}

function PayProfileEditPanel({
  open,
  onOpenChange,
  row,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  row: PayProfileRow;
}) {
  const formRow = { ...row, ...buildDefaults(row) } as unknown as PayProfileRow &
    Record<string, unknown>;
  const form = useForm<PayProfileFormValues>({
    resolver: zodResolver(payProfileFormSchema) as Resolver<PayProfileFormValues>,
    defaultValues: buildDefaults(row),
  });

  return (
    <TabbedFormEditPanel<PayProfileFormValues, PayProfileRow & Record<string, unknown>>
      open={open}
      onOpenChange={onOpenChange}
      row={formRow}
      title="Pay Profile"
      fieldKey="name"
      queryKey="pay-profile-list"
      form={form}
      size="xl"
      formTabs={[
        {
          value: "profile",
          label: "Profile",
          icon: WalletIcon,
          content: <PayProfileForm />,
        },
        {
          value: "assigned-drivers",
          label: "Assigned Drivers",
          icon: UsersIcon,
          content: <AssignedDriversSection profileId={row.id} />,
        },
      ]}
      mutationFn={async (values) => {
        await updatePayProfile({
          id: row.id,
          version: row.version,
          ...toCreateInput(values),
        });
        return values;
      }}
    />
  );
}

function AssignedDriversSection({ profileId }: { profileId: string }) {
  const [assignOpen, setAssignOpen] = useState(false);
  const { data: assignments } = useQuery({
    queryKey: ["pay-profile-assignments", profileId],
    queryFn: () => fetchPayProfileAssignments(profileId),
  });

  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold">Assigned Drivers</h3>
          <p className="text-xs text-muted-foreground">
            Drivers currently paid under this profile. Overrides show where a driver&apos;s rate
            differs from the template — prefer overrides over cloning profiles.
          </p>
        </div>
        <Button type="button" size="sm" variant="outline" onClick={() => setAssignOpen(true)}>
          <UserPlus className="size-3.5" />
          Assign Driver
        </Button>
      </div>
      {(assignments ?? []).length > 0 ? (
        <div className="mt-3 overflow-hidden rounded-lg border">
          <table className="w-full text-xs">
            <thead className="bg-muted/50 text-left">
              <tr>
                <th className="px-3 py-2 font-medium">Driver</th>
                <th className="px-3 py-2 font-medium">Since</th>
                <th className="px-3 py-2 text-right font-medium">Split</th>
                <th className="px-3 py-2 text-right font-medium">Overrides</th>
              </tr>
            </thead>
            <tbody>
              {(assignments ?? []).map((assignment) => (
                <tr key={assignment.id} className="border-t">
                  <td className="px-3 py-2 font-medium">
                    {assignment.worker
                      ? `${assignment.worker.firstName} ${assignment.worker.lastName}`.trim()
                      : "—"}
                  </td>
                  <td className="px-3 py-2">
                    {new Date(assignment.effectiveFrom * 1000).toLocaleDateString("en-US", {
                      month: "short",
                      day: "numeric",
                      year: "numeric",
                    })}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {Number(assignment.splitPercent)}%
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {assignment.rateOverrides?.length ?? 0}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p className="mt-3 text-xs text-muted-foreground">
          No drivers assigned yet. Assign drivers here, or from the Pay tab on the worker.
        </p>
      )}
      <AssignPayProfileDialog
        open={assignOpen}
        onOpenChange={setAssignOpen}
        payProfileId={profileId}
      />
    </div>
  );
}
