import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from "@trenova/shared/components/ui/drawer";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { daysUntil, formatUnixDate } from "@trenova/shared/lib/date";
import {
  fetchMyComplianceProfile,
  updateMyContactInfo,
  type PortalComplianceProfile,
} from "@trenova/shared/lib/graphql/driver-portal";
import { cn } from "@trenova/shared/lib/utils";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PencilIcon, ShieldCheckIcon, ShieldAlertIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { useDashFeatures } from "./use-dash-features";

type ExpiryTone = "ok" | "soon" | "overdue";

function expiryTone(unixSeconds: number | null | undefined): ExpiryTone | null {
  if (!unixSeconds) return null;
  const days = daysUntil(unixSeconds);
  if (days < 0) return "overdue";
  if (days <= 30) return "soon";
  return "ok";
}

function ExpiryRow({ label, value }: { label: string; value: number | null | undefined }) {
  if (!value) return null;
  const tone = expiryTone(value);
  const days = daysUntil(value);
  return (
    <div className="flex items-center justify-between gap-4 py-2">
      <span className="text-sm text-muted-foreground">{label}</span>
      <span className="flex items-center gap-2">
        <span
          className={cn(
            "text-sm font-medium tabular-nums",
            tone === "overdue" && "text-red-600 dark:text-red-400",
            tone === "soon" && "text-amber-600 dark:text-amber-400",
          )}
        >
          {formatUnixDate(value)}
        </span>
        {tone === "overdue" ? (
          <Badge variant="inactive">Expired</Badge>
        ) : tone === "soon" ? (
          <Badge variant="warning">{days === 0 ? "Today" : `${days}d`}</Badge>
        ) : null}
      </span>
    </div>
  );
}

export function ComplianceCard() {
  const features = useDashFeatures();
  const [editOpen, setEditOpen] = useState(false);
  const profile = useQuery({
    queryKey: ["dash-compliance-profile"],
    queryFn: fetchMyComplianceProfile,
  });

  if (profile.isPending) {
    return <Skeleton className="h-52 w-full rounded-2xl" />;
  }
  if (!profile.data) {
    return null;
  }
  const data = profile.data;

  return (
    <div className="rounded-2xl border border-border bg-card p-4">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          {data.isQualified ? (
            <ShieldCheckIcon className="size-4 text-green-600 dark:text-green-400" />
          ) : (
            <ShieldAlertIcon className="size-4 text-red-600 dark:text-red-400" />
          )}
          <h2 className="text-sm font-semibold">Qualification file</h2>
        </div>
        <Badge variant={data.isQualified ? "active" : "inactive"}>
          {data.isQualified ? "Qualified" : "Action needed"}
        </Badge>
      </div>

      <dl className="mt-3 flex flex-col gap-1.5 border-t border-border pt-3 text-sm">
        <div className="flex items-center justify-between gap-4">
          <dt className="text-muted-foreground">CDL</dt>
          <dd className="font-medium">
            {data.licenseNumber}
            {data.licenseState ? ` · ${data.licenseState}` : ""}
            {data.cdlClass ? ` · Class ${data.cdlClass}` : ""}
          </dd>
        </div>
        {data.endorsement ? (
          <div className="flex items-center justify-between gap-4">
            <dt className="text-muted-foreground">Endorsements</dt>
            <dd className="font-medium">{data.endorsement}</dd>
          </div>
        ) : null}
      </dl>

      <div className="mt-2 divide-y divide-border border-t border-border">
        <ExpiryRow label="CDL expires" value={data.licenseExpiry} />
        <ExpiryRow label="Medical card" value={data.medicalCardExpiry} />
        <ExpiryRow label="Hazmat" value={data.hazmatExpiry} />
        <ExpiryRow label="Physical due" value={data.physicalDueDate} />
        <ExpiryRow label="MVR review" value={data.mvrDueDate} />
        <ExpiryRow label="TWIC" value={data.twicExpiry} />
      </div>
      <p className="mt-2 text-xs text-muted-foreground">
        License and medical dates are managed by your carrier — if something is wrong, upload the
        updated document below and tell your fleet manager.
      </p>

      <div className="mt-3 border-t border-border pt-3">
        <div className="flex items-center justify-between gap-2">
          <h3 className="text-sm font-semibold">Contact details</h3>
          {features.allowContactInfoEdit ? (
            <Button variant="outline" size="sm" className="h-8" onClick={() => setEditOpen(true)}>
              <PencilIcon className="size-3.5" />
              Edit
            </Button>
          ) : null}
        </div>
        <dl className="mt-2 flex flex-col gap-1.5 text-sm">
          <ContactRow label="Phone" value={data.phoneNumber} />
          <ContactRow
            label="Address"
            value={[
              data.addressLine1,
              data.addressLine2,
              data.city,
              data.stateAbbreviation,
              data.postalCode,
            ]
              .filter(Boolean)
              .join(", ")}
          />
          <ContactRow
            label="Emergency"
            value={[data.emergencyContactName, data.emergencyContactPhone]
              .filter(Boolean)
              .join(" · ")}
          />
        </dl>
      </div>

      <ContactEditDrawer profile={data} open={editOpen} onOpenChange={setEditOpen} />
    </div>
  );
}

function ContactRow({ label, value }: { label: string; value: string }) {
  if (!value) return null;
  return (
    <div className="flex items-start justify-between gap-4">
      <dt className="shrink-0 text-muted-foreground">{label}</dt>
      <dd className="text-right font-medium">{value}</dd>
    </div>
  );
}

type ContactEditDrawerProps = {
  profile: PortalComplianceProfile;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

function ContactEditDrawer({ profile, open, onOpenChange }: ContactEditDrawerProps) {
  const queryClient = useQueryClient();
  const [form, setForm] = useState(() => ({
    phoneNumber: profile.phoneNumber,
    addressLine1: profile.addressLine1,
    addressLine2: profile.addressLine2,
    city: profile.city,
    postalCode: profile.postalCode,
    emergencyContactName: profile.emergencyContactName,
    emergencyContactPhone: profile.emergencyContactPhone,
  }));

  const save = useMutation({
    mutationFn: () =>
      updateMyContactInfo({
        phoneNumber: form.phoneNumber.trim(),
        addressLine1: form.addressLine1.trim(),
        addressLine2: form.addressLine2.trim() || undefined,
        city: form.city.trim(),
        postalCode: form.postalCode.trim(),
        emergencyContactName: form.emergencyContactName.trim() || undefined,
        emergencyContactPhone: form.emergencyContactPhone.trim() || undefined,
      }),
    onSuccess: async () => {
      toast.success("Contact details updated.");
      await queryClient.invalidateQueries({ queryKey: ["dash-compliance-profile"] });
      await queryClient.invalidateQueries({ queryKey: ["dash-profile"] });
      onOpenChange(false);
    },
    onError: (error: Error) => toast.error(error.message || "We couldn't save your changes."),
  });

  const setField = (field: keyof typeof form) => (event: React.ChangeEvent<HTMLInputElement>) =>
    setForm((current) => ({ ...current, [field]: event.target.value }));

  const canSave =
    form.phoneNumber.trim().length > 0 &&
    form.addressLine1.trim().length > 0 &&
    form.city.trim().length > 0 &&
    form.postalCode.trim().length > 0;

  return (
    <Drawer open={open} onOpenChange={onOpenChange}>
      <DrawerContent>
        <DrawerHeader>
          <DrawerTitle>Update contact details</DrawerTitle>
          <DrawerDescription>
            Keep your phone and address current so dispatch and payroll can reach you.
          </DrawerDescription>
        </DrawerHeader>

        <div className="flex max-h-[50vh] flex-col gap-3 overflow-y-auto px-4">
          <Field label="Phone">
            <Input
              type="tel"
              inputMode="tel"
              value={form.phoneNumber}
              onChange={setField("phoneNumber")}
            />
          </Field>
          <Field label="Address line 1">
            <Input value={form.addressLine1} onChange={setField("addressLine1")} />
          </Field>
          <Field label="Address line 2">
            <Input value={form.addressLine2} onChange={setField("addressLine2")} />
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="City">
              <Input value={form.city} onChange={setField("city")} />
            </Field>
            <Field label="ZIP code">
              <Input
                inputMode="numeric"
                value={form.postalCode}
                onChange={setField("postalCode")}
              />
            </Field>
          </div>
          <Field label="Emergency contact name">
            <Input value={form.emergencyContactName} onChange={setField("emergencyContactName")} />
          </Field>
          <Field label="Emergency contact phone">
            <Input
              type="tel"
              inputMode="tel"
              value={form.emergencyContactPhone}
              onChange={setField("emergencyContactPhone")}
            />
          </Field>
        </div>

        <DrawerFooter>
          <Button
            className="h-11"
            disabled={!canSave || save.isPending}
            onClick={() => save.mutate()}
          >
            {save.isPending ? "Saving..." : "Save"}
          </Button>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label className="text-xs text-muted-foreground">{label}</Label>
      {children}
    </div>
  );
}
