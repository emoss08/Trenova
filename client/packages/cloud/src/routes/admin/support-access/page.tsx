import { SelectField } from "@/components/fields/select-field";
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
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
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
import { AlertCircleIcon, InfoCircleIcon } from "@trenova/shared/components/icons";
import { usePublicConfig } from "@trenova/shared/hooks/use-public-config";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { supportAccess } from "../../../lib/queries/support-access";
import { accessModeLabel, durationLabel, endReasonLabel } from "../../../lib/support-access";
import { supportAccessService } from "../../../services/support-access";
import {
  createGrantRequestSchema,
  type CreateGrantRequest,
  type CustomerSession,
  type GrantState,
  type SupportGrant,
} from "../../../types/support-access";

const DEFAULT_DURATION_HOURS = 24;

/**
 * Where an organization's administrators decide whether Trenova support may come in:
 * for how long, read-only or able to make changes, and what support has done while
 * it was here. Revoking access ends any support session at once.
 */
export function SupportAccessPage() {
  const t = useT();
  const { isCloud } = usePublicConfig();
  const canManage = usePermissionStore((state) =>
    state.hasPermission(Resource.Organization, Operation.Update),
  );
  const state = useQuery({ ...supportAccess.grantState(), enabled: isCloud });
  const [changing, setChanging] = useState(false);

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Support access"),
        description: t(
          "Allow Trenova support into this organization for a limited time. Every support session and every change it makes is recorded in the audit log.",
        ),
      }}
    >
      {!isCloud ? (
        <Alert variant="info">
          <InfoCircleIcon />
          <AlertTitle>{t("Not available on this installation")}</AlertTitle>
          <AlertDescription>{t("Support access applies only on Trenova Cloud.")}</AlertDescription>
        </Alert>
      ) : state.isPending ? (
        <Skeleton className="h-40 w-full" />
      ) : state.isError ? (
        <Alert variant="destructive">
          <AlertCircleIcon />
          <AlertTitle>{t("Unable to load support access")}</AlertTitle>
          <AlertDescription>{t("Refresh the page to try again.")}</AlertDescription>
        </Alert>
      ) : (
        <>
          {state.data.grant && !changing ? (
            <ActiveGrant
              grant={state.data.grant}
              canManage={canManage}
              onChange={() => setChanging(true)}
            />
          ) : (
            <GrantForm
              state={state.data}
              canManage={canManage}
              replacing={state.data.grant !== null}
              onDone={() => setChanging(false)}
            />
          )}
          <SessionsPanel sessions={state.data.sessions} />
        </>
      )}
    </PageLayout>
  );
}

function ActiveGrant({
  grant,
  canManage,
  onChange,
}: {
  grant: SupportGrant;
  canManage: boolean;
  onChange: () => void;
}) {
  const t = useT();
  const [confirming, setConfirming] = useState(false);

  return (
    <SectionPanel
      title={t("Trenova support can access this organization")}
      action={
        canManage ? (
          <div className="flex gap-2">
            <Button type="button" size="sm" variant="outline" onClick={onChange}>
              {t("Change access")}
            </Button>
            <Button
              type="button"
              size="sm"
              variant="destructive"
              onClick={() => setConfirming(true)}
            >
              {t("Revoke access")}
            </Button>
          </div>
        ) : undefined
      }
    >
      <div className="p-3">
        <DescriptionList columns={2}>
          <DescriptionItem label={t("Access")}>
            <Badge variant={grant.accessMode === "read_write" ? "warning" : "neutral"}>
              {accessModeLabel(grant.accessMode, t)}
            </Badge>
          </DescriptionItem>
          <DescriptionItem label={t("Allowed until")}>
            {formatUnixDateTimeMedium(grant.expiresAt)}
          </DescriptionItem>
          <DescriptionItem label={t("Allowed since")}>
            {formatUnixDateTimeMedium(grant.startsAt)}
          </DescriptionItem>
          <DescriptionItem label={t("Note")}>
            {grant.note ? grant.note : <DescriptionEmpty />}
          </DescriptionItem>
        </DescriptionList>
      </div>
      <RevokeDialog open={confirming} onClose={() => setConfirming(false)} />
    </SectionPanel>
  );
}

function RevokeDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT();
  const queryClient = useQueryClient();
  const revoke = useApiMutation({
    mutationFn: () => supportAccessService.revokeGrant(),
    resourceName: "Support access",
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: supportAccess.grantState().queryKey });
      onClose();
    },
  });

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{t("Revoke Trenova support access?")}</DialogTitle>
          <DialogDescription>
            {t("Any support session in progress ends immediately.")}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            variant="destructive"
            isLoading={revoke.isPending}
            loadingText={t("Revoking...")}
            onClick={() => void revoke.mutateAsync()}
          >
            {t("Revoke access")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function GrantForm({
  state,
  canManage,
  replacing,
  onDone,
}: {
  state: GrantState;
  canManage: boolean;
  replacing: boolean;
  onDone: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const durationOptions = useMemo(
    () =>
      state.durationHours.map((hours) => ({
        value: hours,
        label: durationLabel(hours, t),
      })),
    [state.durationHours, t],
  );
  const modeOptions = useMemo(
    () => [
      {
        value: "read_only",
        label: accessModeLabel("read_only", t),
        description: t("Support can look but not change anything"),
      },
      {
        value: "read_write",
        label: accessModeLabel("read_write", t),
        description: t("Support may make changes after signing in again, 30 minutes at a time"),
      },
    ],
    [t],
  );

  const form = useForm<CreateGrantRequest>({
    resolver: zodResolver(createGrantRequestSchema),
    defaultValues: {
      durationHours: state.durationHours.includes(DEFAULT_DURATION_HOURS)
        ? DEFAULT_DURATION_HOURS
        : (state.durationHours[0] ?? DEFAULT_DURATION_HOURS),
      accessMode: "read_only",
      note: "",
    },
  });

  const create = useApiMutation<SupportGrant, CreateGrantRequest, unknown, CreateGrantRequest>({
    mutationFn: (values) => supportAccessService.createGrant(values),
    form,
    resourceName: "Support access",
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: supportAccess.grantState().queryKey });
      onDone();
    },
  });

  const submit = form.handleSubmit((values) => void create.mutateAsync(values));

  return (
    <SectionPanel
      title={replacing ? t("Change Trenova support access") : t("Allow Trenova support access")}
    >
      <div className="flex flex-col gap-3 p-3">
        {!canManage ? (
          <Alert variant="info" size="sm">
            <InfoCircleIcon />
            <AlertDescription>
              {t("Only administrators who can change organization settings can allow support access.")}
            </AlertDescription>
          </Alert>
        ) : null}
        {replacing ? (
          <Alert variant="warning" size="sm">
            <AlertCircleIcon />
            <AlertDescription>
              {t("Changing access ends any support session in progress.")}
            </AlertDescription>
          </Alert>
        ) : null}
        <Form onSubmit={submit}>
          <FormGroup cols={2}>
            <FormControl>
              <SelectField
                control={form.control}
                name="durationHours"
                label={t("For")}
                options={durationOptions}
                isReadOnly={!canManage}
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={form.control}
                name="accessMode"
                label={t("Access")}
                options={modeOptions}
                isReadOnly={!canManage}
                rules={{ required: true }}
              />
            </FormControl>
          </FormGroup>
          <FormGroup cols={1} className="mt-2">
            <FormControl>
              <TextareaField
                control={form.control}
                name="note"
                label={t("Note for Trenova support")}
                placeholder={t("What should support look at?")}
                disabled={!canManage}
              />
            </FormControl>
          </FormGroup>
        </Form>
        <div className="flex justify-end gap-2">
          {replacing ? (
            <Button type="button" variant="outline" onClick={onDone}>
              {t("Cancel")}
            </Button>
          ) : null}
          <Button
            type="button"
            disabled={!canManage}
            isLoading={create.isPending}
            loadingText={t("Saving...")}
            onClick={submit}
          >
            {t("Allow access")}
          </Button>
        </div>
      </div>
    </SectionPanel>
  );
}

function sessionStatusLabel(session: CustomerSession, t: TranslateFn): string {
  switch (session.status) {
    case "active":
      return t("In progress");
    case "expired":
      return t("Expired");
    case "ended":
      return session.endReason ? endReasonLabel(session.endReason, t) : t("Ended");
  }
}

function SessionsPanel({ sessions }: { sessions: CustomerSession[] }) {
  const t = useT();

  return (
    <SectionPanel title={t("Support sessions")} count={sessions.length}>
      {sessions.length === 0 ? (
        <SectionPanelQuiet>{t("Trenova support has not opened a session here.")}</SectionPanelQuiet>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("Staff member")}</TableHead>
              <TableHead>{t("Status")}</TableHead>
              <TableHead>{t("Access")}</TableHead>
              <TableHead>{t("Reason")}</TableHead>
              <TableHead>{t("Ticket")}</TableHead>
              <TableHead>{t("Started")}</TableHead>
              <TableHead>{t("Write access")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sessions.map((session) => (
              <TableRow key={session.id}>
                <TableCell>{session.staffName}</TableCell>
                <TableCell>
                  <Badge variant={session.status === "active" ? "info" : "neutral"}>
                    {sessionStatusLabel(session, t)}
                  </Badge>
                </TableCell>
                <TableCell>{accessModeLabel(session.mode, t)}</TableCell>
                <TableCell className="max-w-xs truncate" title={session.reason}>
                  {session.reason}
                </TableCell>
                <TableCell>{session.ticketReference || <DescriptionEmpty />}</TableCell>
                <TableCell>{formatUnixDateTimeMedium(session.startedAt)}</TableCell>
                <TableCell>
                  {session.elevationCount > 0
                    ? t(
                        "{0, plural, one {Elevated # time} other {Elevated # times}}",
                        session.elevationCount,
                      )
                    : t("Never")}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </SectionPanel>
  );
}
