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
  orderPoliciesForSigning,
  POLICY_STANDING_TONES,
  policyStanding,
  signatureMatches,
  type PolicyStandingValue,
} from "@trenova/shared/lib/self-service";
import { cn } from "@trenova/shared/lib/utils";
import {
  CheckIcon,
  ChevronRightIcon,
  ExternalLinkIcon,
  FileSignatureIcon,
  FileTextIcon,
  PenLineIcon,
} from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { useDashProfile } from "./dash-layout";

const STANDING_ACCENT: Record<PolicyStandingValue, string> = {
  outstanding: "var(--color-amber-500)",
  unread: "var(--color-blue-500)",
  signed: "var(--color-emerald-500)",
  read: "var(--color-emerald-500)",
};

/**
 * Policies the driver is bound by, signed or not. What still needs doing sits
 * on top; what is done sits underneath. Signing is a typed full name and an
 * explicit "I have read this" — a record that a specific person, at a specific
 * time, agreed to a specific text. The card says what it is asking before the
 * tap, not after.
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
  const rows = orderPoliciesForSigning(policies.data ?? []);
  if (rows.length === 0) return null;

  const outstanding = rows.filter((row) => !row.acknowledgedAt).length;

  return (
    <div className="border-border bg-card rounded-2xl border p-4" data-testid="policies-card">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <span
            className={cn(
              "grid size-9 shrink-0 place-items-center rounded-xl",
              outstanding > 0
                ? "bg-amber-500/12 text-amber-600 dark:text-amber-300"
                : "bg-emerald-500/12 text-emerald-600 dark:text-emerald-300",
            )}
          >
            {outstanding > 0 ? (
              <FileSignatureIcon className="size-4" />
            ) : (
              <CheckIcon className="size-4" />
            )}
          </span>
          <div className="min-w-0">
            <p className="text-sm font-semibold">Policies</p>
            <p className="text-muted-foreground text-xs">
              {outstanding === 0 ? "You are up to date." : `${outstanding} to read and sign.`}
            </p>
          </div>
        </div>
        {outstanding > 0 ? (
          <Badge variant="warning" data-testid="policies-outstanding-count">
            {outstanding}
          </Badge>
        ) : null}
      </div>

      <ul className="mt-3 flex flex-col gap-1.5">
        {rows.map((policy) => {
          const standing = policyStanding(policy);
          const tone = POLICY_STANDING_TONES[standing];
          const accent = STANDING_ACCENT[standing];
          const done = Boolean(policy.acknowledgedAt);
          return (
            <li key={policy.id} data-testid="policy-row" data-policy-id={policy.id}>
              <button
                type="button"
                onClick={() => setOpen(policy)}
                className={cn(
                  "border-border/70 bg-background/60 hover:bg-muted/50 relative flex w-full items-center gap-3 overflow-hidden rounded-xl border p-3 text-left text-sm transition-[transform,background-color] active:scale-[0.99]",
                  done && "opacity-80",
                )}
              >
                <span
                  className="absolute inset-y-0 left-0 w-1"
                  style={{ backgroundColor: accent }}
                  aria-hidden
                />
                <span
                  className="grid size-8 shrink-0 place-items-center rounded-lg"
                  style={{
                    backgroundColor: `color-mix(in oklch, ${accent} 14%, transparent)`,
                    color: accent,
                  }}
                >
                  {policy.hasDocument ? (
                    <FileTextIcon className="size-4" />
                  ) : (
                    <PenLineIcon className="size-4" />
                  )}
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-medium">{policy.title}</span>
                  <span className="text-muted-foreground block truncate text-xs">
                    v{policy.versionLabel}
                    {policy.acknowledgedAt
                      ? ` · ${formatShiftDate(policy.acknowledgedAt)}`
                      : policy.summary
                        ? ` · ${policy.summary}`
                        : ""}
                  </span>
                </span>
                <Badge className={cn("shrink-0 border", tone.badge)}>{tone.label}</Badge>
                <ChevronRightIcon className="text-muted-foreground size-4 shrink-0" />
              </button>
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
  const fullName = `${firstName} ${lastName}`.trim();
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
            <div className="border-border bg-muted/30 rounded-xl border p-3 text-sm leading-relaxed whitespace-pre-wrap">
              {policy.body}
            </div>
          ) : null}
          {policy?.hasDocument ? (
            <Button
              variant="outline"
              className="h-11 justify-between"
              disabled={openDocument.isPending}
              onClick={() => openDocument.mutate()}
            >
              <span className="flex items-center gap-2">
                <FileTextIcon className="size-4" />
                {openDocument.isPending ? "Opening…" : "Open the document"}
              </span>
              <ExternalLinkIcon className="text-muted-foreground size-4" />
            </Button>
          ) : null}

          {signed ? (
            <div className="flex items-center gap-2 rounded-xl border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-xs text-emerald-700 dark:text-emerald-300">
              <CheckIcon className="size-4 shrink-0" />
              {policy?.signatureName
                ? `Signed “${policy.signatureName}” on ${formatShiftDate(policy.acknowledgedAt ?? 0)}.`
                : `Read on ${formatShiftDate(policy?.acknowledgedAt ?? 0)}.`}
            </div>
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
                  <div className="relative">
                    <Input
                      id="policy-signature"
                      autoComplete="off"
                      placeholder={fullName}
                      value={signature}
                      onChange={(event) => setSignature(event.target.value)}
                      aria-invalid={Boolean(signature) && !nameOk}
                      className={cn(
                        "h-11 pr-9 font-serif text-base italic transition-colors",
                        signature && nameOk && "border-emerald-500/60",
                      )}
                    />
                    {signature && nameOk ? (
                      <CheckIcon
                        className="pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 text-emerald-600 dark:text-emerald-300"
                        aria-hidden
                      />
                    ) : null}
                  </div>
                  {signature && !nameOk ? (
                    <p className="text-destructive text-xs">
                      Sign as {fullName} — the name on your record.
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
