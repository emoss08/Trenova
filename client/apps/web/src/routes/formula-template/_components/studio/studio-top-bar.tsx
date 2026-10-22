import { usePermission } from "@/hooks/use-permission";
import { formulaTemplateRoutes } from "@/lib/formula-template-routes";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@trenova/shared/components/ui/hover-card";
import { Kbd, kbdOnSolidClassName } from "@trenova/shared/components/ui/kbd";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import type {
  FormulaTemplate,
  TemplateUsageResponse,
} from "@trenova/shared/types/formula-template";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  ArrowLeftIcon,
  CheckIcon,
  ClockIcon,
  ClockRewindIcon,
  Dataflow04Icon,
  DotsHorizontalIcon,
  Download01Icon,
  FileUploadIcon,
  GitBranch02Icon,
  LayoutRightIcon,
  MessageAlertSquareIcon,
  Send01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Fragment } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { useNavigate } from "react-router";
import type { ApprovalAction } from "../approval-action-dialog";
import { billingLabel } from "./model/billing";
import { useStudio } from "./studio-context";

const USAGE_LABELS: Readonly<Record<string, string>> = defineLabels({
  shipment: "Shipments",
  rate_agreement_rule: "Rate agreement rules",
  rate_agreement: "Rate agreements",
  billing_control_fallback: "Billing fallback",
});

/** What editing does to the places a template is used, by where it is. */
export function usageConsequence(
  status: FormulaTemplate["status"] | undefined,
  version: number,
  t: TranslateFn,
): string {
  switch (status) {
    case "Active":
      return t(
        "Editing the source returns it to Draft; shipments keep rating with v{0} until the new version is approved. Name and description edits are safe.",
        version,
      );
    case "InReview":
      return t("A material edit cancels the review and returns it to Draft.");
    case "Inactive":
      return t("These stopped rating with it when it was archived; reactivate it through review.");
    default:
      return t("These will rate with the new content once it is approved.");
  }
}

export function UsageRows({ usage }: { usage: TemplateUsageResponse }) {
  return (
    <div className="flex flex-col gap-1">
      {usage.usages.map((item) => (
        <div key={item.type} className="flex items-center justify-between gap-3 text-xs">
          <span className="text-foreground-muted">{USAGE_LABELS[item.type] ?? item.type}</span>
          <span className="font-mono tabular-nums">{item.count}</span>
        </div>
      ))}
    </div>
  );
}

type Stage = { key: "Draft" | "InReview" | "Active"; label: string };

const STAGE_TONE: Record<Stage["key"], string> = {
  Draft: "text-foreground-subtle bg-foreground-subtle",
  InReview: "text-info bg-info",
  Active: "text-success bg-success",
};

function VersionChip({ version }: { version: number }) {
  return (
    <span className="bg-surface-hover text-foreground-muted rounded-badge px-1.5 font-mono text-2xs tabular-nums">
      v{version}
    </span>
  );
}

function Connector({ past }: { past: boolean }) {
  return (
    <li
      aria-hidden
      className={cn("h-px w-4.5 shrink-0", past ? "bg-success opacity-60" : "bg-border")}
    />
  );
}

/**
 * Where the template is in review: New or Draft, then In review, then
 * Active, with the version on the stage it is at. An unsaved edit that
 * changes what the template computes marks Draft as where saving will put it.
 */
function Lifecycle() {
  const t = useT();
  const { mode, template, material, usage, usageTotal } = useStudio();
  const stages: Stage[] = [
    { key: "Draft", label: t("Draft") },
    { key: "InReview", label: t("In review") },
    { key: "Active", label: t("Active") },
  ];
  const version = template?.currentVersionNumber || 1;

  if (mode === "create" || !template) {
    return (
      <ol aria-label={t("Lifecycle")} className="flex items-center gap-1.5 text-xs">
        <li className="bg-card flex items-center gap-1.5 rounded-badge border px-2 py-0.5 font-medium">
          <span className="fs-stage-dot bg-brand text-brand size-1.75 rounded-full" />
          {t("New")}
        </li>
        {stages.map((stage) => (
          <Fragment key={stage.key}>
            <Connector past={false} />
            <li className="text-foreground-subtle flex items-center gap-1.5">
              <span className="border-border size-1.75 rounded-full border" />
              {stage.label}
            </li>
          </Fragment>
        ))}
      </ol>
    );
  }

  if (template.status === "Inactive") {
    return (
      <ol aria-label={t("Lifecycle")} className="flex items-center text-xs">
        <li className="bg-card text-foreground-muted flex items-center gap-1.5 rounded-badge border px-2 py-0.5 font-medium">
          <span className="bg-foreground-subtle size-1.75 rounded-full" />
          {t("Archived · v{0}", version)}
        </li>
      </ol>
    );
  }

  const current = stages.findIndex((stage) => stage.key === template.status);
  const goingBack = material && current > 0;
  const billing = billingLabel(template, t);

  return (
    <ol aria-label={t("Lifecycle")} className="flex items-center gap-1.5 text-xs">
      {stages.map((stage, index) => {
        const past = index < current;
        const isCurrent = index === current;
        const marksSave = goingBack && index === 0;
        return (
          <Fragment key={stage.key}>
            {index > 0 && <Connector past={index <= current} />}
            {isCurrent ? (
              <li
                aria-current="step"
                className={cn(
                  "bg-card flex items-center gap-1.5 rounded-badge border px-2 py-0.5 font-medium transition-opacity",
                  goingBack && "opacity-60",
                )}
              >
                <span
                  key={stage.key}
                  className={cn("fs-stage-dot size-1.75 rounded-full", STAGE_TONE[stage.key])}
                />
                {stage.label}
                <VersionChip version={version} />
              </li>
            ) : (
              <li
                className={cn(
                  "flex items-center gap-1.5",
                  past ? "text-foreground-muted" : "text-foreground-subtle",
                  marksSave &&
                    "border-warning-border text-warning rounded-badge border border-dashed px-2 py-0.5",
                )}
              >
                {past && !marksSave ? (
                  <CheckIcon className="text-success size-3" strokeWidth={2.4} />
                ) : (
                  <span
                    className={cn(
                      "size-1.75 rounded-full border",
                      marksSave ? "border-warning" : "border-border",
                    )}
                  />
                )}
                {stage.label}
                {marksSave && (
                  <span className="bg-warning-subtle text-warning-subtle-foreground rounded-badge px-1.5 text-2xs">
                    {t("on save")}
                  </span>
                )}
              </li>
            )}
            {isCurrent && stage.key === "Active" && usage && (
              <li className="@max-[1100px]:hidden">
                <HoverCard>
                  <HoverCardTrigger
                    delay={250}
                    render={
                      <button
                        type="button"
                        className="ui-focus-ring text-foreground-muted hover:text-foreground decoration-border-strong rounded-sm underline decoration-dashed underline-offset-4"
                      >
                        {t("{0} in use", usageTotal)}
                      </button>
                    }
                  />
                  <HoverCardContent side="bottom" align="start" className="w-64">
                    <div className="flex flex-col gap-2">
                      <p className="text-sm font-semibold">{t("Where this template is used")}</p>
                      {usage.usages.length === 0 ? (
                        <p className="text-foreground-muted text-xs">
                          {t("Nothing references this template yet. Editing it is safe.")}
                        </p>
                      ) : (
                        <UsageRows usage={usage} />
                      )}
                      <p className="text-foreground-muted text-2xs leading-relaxed">
                        {usageConsequence(template.status, version, t)}
                      </p>
                    </div>
                  </HoverCardContent>
                </HoverCard>
              </li>
            )}
          </Fragment>
        );
      })}
      {billing && (
        <li className="text-foreground-muted @max-[1100px]:hidden flex items-center gap-1.5 pl-1.5">
          <ClockIcon className="size-3" />
          {billing}
        </li>
      )}
    </ol>
  );
}

function SaveControl() {
  const t = useT();
  const { mode, dirty, isSubmitting, save } = useStudio();
  const keys = formatShortcut("S");

  if (mode === "edit" && !dirty && !isSubmitting) {
    return (
      <span className="text-foreground-muted flex items-center gap-1 px-1 text-sm">
        <CheckIcon className="text-success size-3.5" strokeWidth={2.2} />
        {t("Saved")}
      </span>
    );
  }

  return (
    <Button
      type="button"
      size="sm"
      onClick={save}
      isLoading={isSubmitting}
      loadingText={mode === "create" ? t("Creating…") : t("Saving…")}
      className="gap-2"
    >
      {mode === "edit" && <span aria-hidden className="bg-warning size-1.5 rounded-full" />}
      {mode === "create" ? t("Create template") : t("Save changes")}
      <Kbd className={cn(kbdOnSolidClassName, "border-b")}>{keys}</Kbd>
    </Button>
  );
}

export type StudioTopBarProps = {
  onApprovalAction: (action: ApprovalAction) => void;
  onSubmitForReview: () => void;
  onVersionHistory: () => void;
  onBacktest: () => void;
  onFork: () => void;
  onLineage: () => void;
  onExport: () => void;
  onImport: () => void;
};

/**
 * The studio's one bar: back, the template's name, where it is in review,
 * then what can be done to it now and the save state, and last the toggle for
 * the floating panel. Save carries the state itself, quiet once everything is
 * saved.
 */
export function StudioTopBar({
  onApprovalAction,
  onSubmitForReview,
  onVersionHistory,
  onBacktest,
  onFork,
  onLineage,
  onExport,
  onImport,
}: StudioTopBarProps) {
  const t = useT();
  const navigate = useNavigate();
  const { mode, template, dirty, readiness, railOpen, setRailOpen } = useStudio();
  const { control } = useFormContext();
  const name = (useWatch({ control, name: "name" }) as string | undefined)?.trim() ?? "";
  const { allowed: canSubmit } = usePermission(Resource.FormulaTemplate, Operation.Submit);
  const { allowed: canApprove } = usePermission(Resource.FormulaTemplate, Operation.Approve);
  const { allowed: canReject } = usePermission(Resource.FormulaTemplate, Operation.Reject);

  const status = template?.status;
  const version = template?.currentVersionNumber || 1;
  const blocking = readiness.some((check) => check.status === "fail");

  return (
    <header className="bg-canvas @container flex h-13 shrink-0 items-center gap-3 border-b px-3">
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        aria-label={t("Back to formula templates")}
        onClick={() => void navigate(formulaTemplateRoutes.list)}
      >
        <ArrowLeftIcon className="size-4" />
      </Button>
      <h1 className={cn("min-w-0 truncate text-lg font-medium", !name && "text-foreground-subtle")}>
        {name || (mode === "create" ? t("New rating method") : t("Untitled"))}
      </h1>
      <span aria-hidden className="bg-border h-5 w-px shrink-0 @max-[1100px]:hidden" />
      <div className="shrink-0 @max-[760px]:hidden">
        <Lifecycle />
      </div>
      <span className="flex-1" />

      {mode === "edit" && (status === "Draft" || status === "Inactive") && canSubmit && (
        <Tooltip>
          <TooltipTrigger
            render={
              <span>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="gap-1.5"
                  disabled={dirty}
                  onClick={onSubmitForReview}
                >
                  <Send01Icon className="size-3.5" />
                  {status === "Inactive" ? t("Reactivate via review") : t("Submit for review")}
                </Button>
              </span>
            }
          />
          <TooltipContent>
            {dirty
              ? t("Save first; review covers what is saved")
              : t("Send the saved content to a reviewer")}
          </TooltipContent>
        </Tooltip>
      )}
      {mode === "edit" && status === "InReview" && (
        <>
          {canApprove && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="gap-1.5"
              disabled={dirty || blocking}
              onClick={() => onApprovalAction("approve")}
            >
              <CheckIcon className="text-success size-3.5" />
              {t("Approve")}
            </Button>
          )}
          {canReject && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="gap-1.5"
              onClick={() => onApprovalAction("requestChanges")}
            >
              <MessageAlertSquareIcon className="size-3.5" />
              {t("Request changes")}
            </Button>
          )}
        </>
      )}

      <SaveControl />

      {mode === "edit" && (
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button type="button" variant="ghost" size="icon-sm" aria-label={t("More actions")}>
                <DotsHorizontalIcon className="size-4" />
              </Button>
            }
          />
          <DropdownMenuContent align="end" className="min-w-56">
            <DropdownMenuGroup>
              <DropdownMenuItem
                title={t("Version history")}
                description={version > 1 ? t("v1 – v{0}", version) : t("v1")}
                startContent={<ClockIcon className="size-4" />}
                onClick={onVersionHistory}
              />
              <DropdownMenuItem
                title={t("Backtest")}
                description={t("Re-rate recent shipments")}
                startContent={<ClockRewindIcon className="size-4" />}
                onClick={onBacktest}
              />
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <DropdownMenuItem
                title={t("Fork template")}
                startContent={<GitBranch02Icon className="size-4" />}
                onClick={onFork}
              />
              {template?.sourceTemplateId && (
                <DropdownMenuItem
                  title={t("View lineage")}
                  description={t("Forked from v{0}", template.sourceVersionNumber ?? 1)}
                  startContent={<Dataflow04Icon className="size-4" />}
                  onClick={onLineage}
                />
              )}
              <DropdownMenuItem
                title={t("Export JSON")}
                startContent={<Download01Icon className="size-4" />}
                onClick={onExport}
              />
              <DropdownMenuItem
                title={t("Import templates")}
                startContent={<FileUploadIcon className="size-4" />}
                onClick={onImport}
              />
            </DropdownMenuGroup>
            {status === "InReview" && canReject && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  color="danger"
                  title={t("Reject and archive")}
                  startContent={<XCloseIcon className="size-4" />}
                  onClick={() => onApprovalAction("reject")}
                />
              </>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      )}

      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label={t("Toggle side panel")}
              aria-expanded={railOpen}
              onClick={() => setRailOpen(!railOpen)}
              className="aria-expanded:bg-surface-active"
            >
              <LayoutRightIcon className="size-4" />
            </Button>
          }
        />
        <TooltipContent>{railOpen ? t("Hide panel") : t("Show panel")}</TooltipContent>
      </Tooltip>
    </header>
  );
}
