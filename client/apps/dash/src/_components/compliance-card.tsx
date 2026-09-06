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
import {
  fetchMyComplianceProfile,
  fetchMyProfileChangeRequests,
  updateMyContactInfo,
  withdrawMyProfileChange,
  type PortalComplianceProfile,
} from "@trenova/shared/lib/graphql/driver-portal";
import { changeRequestTone, describeChanges } from "@trenova/shared/lib/self-service";
import { cn } from "@trenova/shared/lib/utils";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PencilIcon, ShieldCheckIcon, ShieldAlertIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { useDashFeatures } from "./use-dash-features";

export function ComplianceCard() {
  const features = useDashFeatures();
  const [editOpen, setEditOpen] = useState(false);
  const profile = useQuery({
    queryKey: ["dash-compliance-profile"],
    queryFn: ({ signal }) => fetchMyComplianceProfile({ signal }),
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

      <p className="mt-2 text-xs text-muted-foreground">
        Expiry dates for every card and endorsement are tracked under Credentials below.
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

      <PendingChangeNotice />

      <ContactEditDrawer profile={data} open={editOpen} onOpenChange={setEditOpen} />
    </div>
  );
}

/**
 * A change the driver asked for that is still with the office, or the most
 * recent answer. Shown beside the details it would change, so the record and
 * the request are never read apart from each other.
 */
function PendingChangeNotice() {
  const queryClient = useQueryClient();
  const requests = useQuery({
    queryKey: ["dash-profile-change-requests"],
    queryFn: ({ signal }) => fetchMyProfileChangeRequests({ signal }),
    staleTime: 60 * 1000,
  });

  const withdraw = useMutation({
    mutationFn: (id: string) => withdrawMyProfileChange(id),
    onSuccess: async () => {
      toast.success("Request withdrawn.");
      await queryClient.invalidateQueries({ queryKey: ["dash-profile-change-requests"] });
    },
    onError: (error: Error) => toast.error(error.message || "Could not withdraw it."),
  });

  const latest = requests.data?.[0];
  if (!latest) return null;
  const pending = latest.status === "Pending";
  if (!pending && latest.status !== "Rejected") return null;
  const tone = changeRequestTone(latest.status);

  return (
    <div className="border-border bg-muted/30 mt-3 rounded-lg border p-3 text-xs">
      <div className="flex items-center justify-between gap-2">
        <Badge className={cn("border", tone.badge)}>{tone.label}</Badge>
        {pending ? (
          <Button
            variant="ghost"
            size="sm"
            className="h-7"
            disabled={withdraw.isPending}
            onClick={() => withdraw.mutate(latest.id)}
          >
            Withdraw
          </Button>
        ) : null}
      </div>
      <ul className="mt-2 flex flex-col gap-0.5 tabular-nums">
        {describeChanges(latest.changes).map((line) => (
          <li key={line}>{line}</li>
        ))}
      </ul>
      {latest.decisionNote ? (
        <p className="text-muted-foreground mt-1">Your carrier said: {latest.decisionNote}</p>
      ) : null}
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
  const requiresApproval = useDashFeatures().requireContactChangeApproval;
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
      toast.success(
        requiresApproval
          ? "Sent to your carrier — your record changes once they approve it."
          : "Contact details updated.",
      );
      await queryClient.invalidateQueries({ queryKey: ["dash-compliance-profile"] });
      await queryClient.invalidateQueries({ queryKey: ["dash-profile"] });
      await queryClient.invalidateQueries({ queryKey: ["dash-profile-change-requests"] });
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
            {requiresApproval
              ? "Your carrier checks contact changes before they land on your record. Only the fields you change are sent."
              : "Keep your phone and address current so dispatch and payroll can reach you."}
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
            {save.isPending ? "Sending..." : requiresApproval ? "Send for approval" : "Save"}
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
