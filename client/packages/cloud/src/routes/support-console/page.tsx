import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { AlertTriangleIcon, Lock01Icon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useSearchParams } from "react-router";
import { useStaffProfile } from "../../hooks/use-support-session";
import { supportAccess } from "../../lib/queries/support-access";
import { accessModeLabel, endReasonLabel } from "../../lib/support-access";
import { supportAccessService } from "../../services/support-access";
import {
  startSessionRequestSchema,
  type GrantedOrganization,
  type SessionView,
  type StartSessionRequest,
} from "../../types/support-access";

function enterOrganization() {
  window.location.assign("/");
}

/**
 * Where Trenova platform staff choose an organization that allowed support access and
 * open a support session in it. Opening a session needs a two-factor sign-in, and every
 * session starts read-only.
 */
export function SupportConsolePage() {
  const t = useT();
  const [searchParams] = useSearchParams();
  const endedReason = searchParams.get("ended");
  const profile = useStaffProfile();
  const isStaff = profile.data?.isStaff ?? false;
  const organizations = useQuery({
    ...supportAccess.grantedOrganizations(),
    enabled: isStaff,
  });
  const [target, setTarget] = useState<GrantedOrganization | null>(null);

  const mfaReady = Boolean(profile.data?.mfaEnrolled && profile.data.sessionVerified);

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Support console"),
        description: t(
          "Organizations that allowed Trenova support access. Sessions start read-only and every action is recorded in the organization's audit log.",
        ),
      }}
    >
      {endedReason ? (
        <Alert variant="warning">
          <AlertTriangleIcon />
          <AlertTitle>{t("Your Trenova support session has ended")}</AlertTitle>
          <AlertDescription>{endReasonLabel(endedReason, t)}</AlertDescription>
        </Alert>
      ) : null}
      {profile.isPending ? (
        <Skeleton className="h-24 w-full" />
      ) : !isStaff ? (
        <Alert variant="destructive">
          <Lock01Icon />
          <AlertTitle>{t("Trenova staff only")}</AlertTitle>
          <AlertDescription>
            {t("Only Trenova platform staff can open support sessions.")}
          </AlertDescription>
        </Alert>
      ) : (
        <>
          {!profile.data?.mfaEnrolled ? (
            <Alert variant="warning">
              <AlertTriangleIcon />
              <AlertTitle>{t("Two-factor authentication is required")}</AlertTitle>
              <AlertDescription>
                {t(
                  "Set up an authenticator app in your settings, then sign out and sign in again with a code before opening a support session.",
                )}
              </AlertDescription>
            </Alert>
          ) : !profile.data.sessionVerified ? (
            <Alert variant="warning">
              <AlertTriangleIcon />
              <AlertTitle>{t("Sign in again with your authenticator code")}</AlertTitle>
              <AlertDescription>
                {t(
                  "This sign-in did not use two-factor authentication. Sign out and sign in again to open support sessions.",
                )}
              </AlertDescription>
            </Alert>
          ) : null}

          <OpenSessions sessions={profile.data?.openSessions ?? []} />

          <SectionPanel
            title={t("Organizations allowing support access")}
            count={organizations.data?.length}
          >
            {organizations.isPending ? (
              <Skeleton className="m-3 h-16" />
            ) : organizations.isError ? (
              <SectionPanelQuiet>{t("The organizations could not be loaded.")}</SectionPanelQuiet>
            ) : organizations.data.length === 0 ? (
              <SectionPanelQuiet>
                {t("No organization currently allows Trenova support access.")}
              </SectionPanelQuiet>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("Organization")}</TableHead>
                    <TableHead>{t("Access")}</TableHead>
                    <TableHead>{t("Allowed until")}</TableHead>
                    <TableHead>{t("Note")}</TableHead>
                    <TableHead />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {organizations.data.map((organization) => (
                    <TableRow key={organization.grantId}>
                      <TableCell>{organization.organizationName}</TableCell>
                      <TableCell>
                        <Badge
                          variant={organization.accessMode === "read_write" ? "warning" : "neutral"}
                        >
                          {accessModeLabel(organization.accessMode, t)}
                        </Badge>
                      </TableCell>
                      <TableCell>{formatUnixDateTimeMedium(organization.expiresAt)}</TableCell>
                      <TableCell className="text-muted-foreground max-w-xs truncate">
                        {organization.note}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          type="button"
                          size="sm"
                          disabled={!mfaReady}
                          onClick={() => setTarget(organization)}
                        >
                          {t("Open support session")}
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </SectionPanel>
        </>
      )}

      <StartSessionDialog organization={target} onClose={() => setTarget(null)} />
    </PageLayout>
  );
}

function OpenSessions({ sessions }: { sessions: SessionView[] }) {
  const t = useT();

  if (sessions.length === 0) {
    return null;
  }

  return (
    <Alert variant="info">
      <Lock01Icon />
      <AlertTitle>{t("You have an open support session")}</AlertTitle>
      <AlertDescription>
        {sessions.map((session) => (
          <span key={session.id} className="block">
            {t(
              "{0}, until {1}",
              session.organizationName,
              formatUnixDateTimeMedium(session.expiresAt),
            )}
          </span>
        ))}
        {t("Opening another session ends it.")}
      </AlertDescription>
    </Alert>
  );
}

function StartSessionDialog({
  organization,
  onClose,
}: {
  organization: GrantedOrganization | null;
  onClose: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<StartSessionRequest>({
    resolver: zodResolver(startSessionRequestSchema),
    values: {
      organizationId: organization?.organizationId ?? "",
      businessUnitId: organization?.businessUnitId ?? "",
      reason: "",
      ticketReference: "",
    },
  });

  const start = useApiMutation<SessionView, StartSessionRequest, unknown, StartSessionRequest>({
    mutationFn: (values) => supportAccessService.startSession(values),
    form,
    resourceName: "Support session",
    onSuccess: () => {
      queryClient.clear();
      enterOrganization();
    },
  });

  const submit = form.handleSubmit((values) => void start.mutateAsync(values));

  return (
    <Dialog open={organization !== null} onOpenChange={(next) => !next && onClose()}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{t("Open support session")}</DialogTitle>
          <DialogDescription>
            {t(
              "You will see {0} read-only. The organization's administrators are told the session started, and why.",
              organization?.organizationName ?? "",
            )}
          </DialogDescription>
        </DialogHeader>
        <Form onSubmit={submit}>
          <FormGroup cols={1}>
            <FormControl>
              <InputField
                control={form.control}
                name="ticketReference"
                label={t("Ticket")}
                placeholder={t("SUP-1234")}
              />
            </FormControl>
            <FormControl>
              <TextareaField
                control={form.control}
                name="reason"
                label={t("Reason")}
                placeholder={t("Investigate the missing invoice the customer reported")}
                rules={{ required: true }}
              />
            </FormControl>
          </FormGroup>
        </Form>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            isLoading={start.isPending}
            loadingText={t("Opening...")}
            onClick={submit}
          >
            {t("Open support session")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
