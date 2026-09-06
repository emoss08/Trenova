import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
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
  acknowledgeMyPolicy,
  fetchMyPolicies,
  fetchMyPolicyDocumentUrl,
  type MyPolicy,
} from "@trenova/shared/lib/graphql/driver-portal";
import { formatShiftDate } from "@trenova/shared/lib/scheduling";
import {
  POLICY_STANDING_TONES,
  policyStanding,
  signatureMatches,
} from "@trenova/shared/lib/self-service";
import { cn } from "@trenova/shared/lib/utils";
import { FileTextIcon, PenLineIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { useDashProfile } from "./dash-layout";

/**
 * Policies the driver is bound by, signed or not. Signing is a typed full
 * name and an explicit "I have read this" — a record that a specific person,
 * at a specific time, agreed to a specific text. The card says what it is
 * asking before the tap, not after.
 */
export function PoliciesCard() {
  const [open, setOpen] = useState<MyPolicy | null>(null);

  const policies = useQuery({
    queryKey: ["dash-policies"],
    queryFn: ({ signal }) => fetchMyPolicies({ signal }),
    staleTime: 60 * 1000,
  });

  if (policies.isPending) {
    return <Skeleton className="h-32 w-full rounded-2xl" />;
  }
  const rows = policies.data ?? [];
  if (rows.length === 0) return null;

  const outstanding = rows.filter((row) => !row.acknowledgedAt);

  return (
    <div className="border-border bg-card rounded-2xl border p-4" data-testid="policies-card">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-sm font-semibold">Policies</p>
          <p className="text-muted-foreground text-xs">
            {outstanding.length === 0
              ? "You are up to date."
              : `${outstanding.length} to read and sign.`}
          </p>
        </div>
        {outstanding.length > 0 ? <Badge variant="warning">{outstanding.length}</Badge> : null}
      </div>

      <ul className="mt-3 flex flex-col gap-2">
        {rows.map((policy) => {
          const standing = policyStanding(policy);
          const tone = POLICY_STANDING_TONES[standing];
          return (
            <li key={policy.id} className="flex items-start justify-between gap-2 text-sm">
              <button
                type="button"
                className="flex min-w-0 flex-1 items-start gap-2 text-left"
                onClick={() => setOpen(policy)}
              >
                {policy.hasDocument ? (
                  <FileTextIcon className="text-muted-foreground mt-0.5 size-4 shrink-0" />
                ) : (
                  <PenLineIcon className="text-muted-foreground mt-0.5 size-4 shrink-0" />
                )}
                <span className="min-w-0">
                  <span className="block truncate font-medium">{policy.title}</span>
                  <span className="text-muted-foreground block text-xs">
                    Version {policy.versionLabel}
                    {policy.acknowledgedAt
                      ? ` · ${formatShiftDate(policy.acknowledgedAt)}`
                      : policy.summary
                        ? ` · ${policy.summary}`
                        : ""}
                  </span>
                </span>
              </button>
              <Badge className={cn("shrink-0 border", tone.badge)}>{tone.label}</Badge>
            </li>
          );
        })}
      </ul>

      <PolicyDrawer policy={open} onOpenChange={(next) => !next && setOpen(null)} />
    </div>
  );
}

function PolicyDrawer({
  policy,
  onOpenChange,
}: {
  policy: MyPolicy | null;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const profile = useDashProfile();
  const [signature, setSignature] = useState("");
  const [agreed, setAgreed] = useState(false);

  const firstName = profile.data?.firstName ?? "";
  const lastName = profile.data?.lastName ?? "";
  const signed = Boolean(policy?.acknowledgedAt);
  const needsSignature = Boolean(policy?.requiresSignature);
  const nameOk = !needsSignature || signatureMatches(signature, firstName, lastName);

  const openDocument = useMutation({
    mutationFn: () => fetchMyPolicyDocumentUrl(policy?.id ?? ""),
    onSuccess: (url) => {
      window.open(url, "_blank", "noopener");
    },
    onError: (error: Error) => toast.error(error.message || "Could not open the document"),
  });

  const sign = useMutation({
    mutationFn: () =>
      acknowledgeMyPolicy({
        policyId: policy?.id ?? "",
        signatureName: needsSignature ? signature.trim() : undefined,
      }),
    onSuccess: async () => {
      toast.success(needsSignature ? "Signed — thank you." : "Marked as read.");
      setSignature("");
      setAgreed(false);
      await queryClient.invalidateQueries({ queryKey: ["dash-policies"] });
      await queryClient.invalidateQueries({ queryKey: ["dash-features"] });
      onOpenChange(false);
    },
    onError: (error: Error) => toast.error(error.message || "That did not go through."),
  });

  return (
    <Drawer open={policy !== null} onOpenChange={onOpenChange}>
      <DrawerContent>
        <DrawerHeader>
          <DrawerTitle>{policy?.title}</DrawerTitle>
          <DrawerDescription>
            Version {policy?.versionLabel}
            {policy?.effectiveFrom
              ? ` · in force from ${formatShiftDate(policy.effectiveFrom)}`
              : ""}
          </DrawerDescription>
        </DrawerHeader>

        <div className="flex max-h-[55vh] flex-col gap-3 overflow-y-auto px-4">
          {policy?.summary ? <p className="text-sm">{policy.summary}</p> : null}
          {policy?.body ? (
            <div className="border-border bg-muted/30 rounded-lg border p-3 text-sm whitespace-pre-wrap">
              {policy.body}
            </div>
          ) : null}
          {policy?.hasDocument ? (
            <Button
              variant="outline"
              className="h-11"
              disabled={openDocument.isPending}
              onClick={() => openDocument.mutate()}
            >
              <FileTextIcon className="size-4" />
              {openDocument.isPending ? "Opening…" : "Open the document"}
            </Button>
          ) : null}

          {signed ? (
            <p className="text-muted-foreground text-xs">
              {policy?.signatureName
                ? `Signed “${policy.signatureName}” on ${formatShiftDate(policy.acknowledgedAt ?? 0)}.`
                : `Read on ${formatShiftDate(policy?.acknowledgedAt ?? 0)}.`}
            </p>
          ) : (
            <div className="border-border flex flex-col gap-3 border-t pt-3">
              <label className="flex items-start gap-2 text-sm">
                <Checkbox
                  checked={agreed}
                  onCheckedChange={(value) => setAgreed(value === true)}
                  className="mt-0.5"
                />
                <span>
                  I have read {policy?.title ?? "this policy"}
                  {needsSignature ? " and agree to it." : "."}
                </span>
              </label>
              {needsSignature ? (
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="policy-signature" className="text-muted-foreground text-xs">
                    Type your full name to sign
                  </Label>
                  <Input
                    id="policy-signature"
                    autoComplete="off"
                    placeholder={`${firstName} ${lastName}`.trim()}
                    value={signature}
                    onChange={(event) => setSignature(event.target.value)}
                  />
                  {signature && !nameOk ? (
                    <p className="text-destructive text-xs">
                      Sign as {firstName} {lastName} — the name on your record.
                    </p>
                  ) : (
                    <p className="text-muted-foreground text-xs">
                      Your name, the time, and the device you signed from are kept with the
                      signature.
                    </p>
                  )}
                </div>
              ) : null}
            </div>
          )}
        </div>

        <DrawerFooter>
          {signed ? (
            <Button variant="outline" className="h-11" onClick={() => onOpenChange(false)}>
              Close
            </Button>
          ) : (
            <Button
              className="h-11"
              disabled={!agreed || !nameOk || sign.isPending}
              onClick={() => sign.mutate()}
            >
              {sign.isPending ? "Sending…" : needsSignature ? "Sign" : "Mark as read"}
            </Button>
          )}
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  );
}
