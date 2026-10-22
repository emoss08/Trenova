import { usePermission } from "@/hooks/use-permission";
import { formulaLibraryRoutes } from "@/lib/formula-library-routes";
import {
  ArchiveIcon,
  ArrowLeftIcon,
  CheckIcon,
  Copy01Icon,
  DotsHorizontalIcon,
  LayoutRightIcon,
  MessageAlertSquareIcon,
  Send01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { Kbd, kbdOnSolidClassName } from "@trenova/shared/components/ui/kbd";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import type { FormulaLibraryFormValues } from "@trenova/shared/types/formula-library";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useFormContext, useWatch } from "react-hook-form";
import { useNavigate } from "react-router";
import type { LibraryDecision } from "./library-review-dialogs";
import { useLibraryStudio } from "./library-studio-context";
import { libraryPath } from "./starter";
import { StatusLabel } from "./status-label";

function SaveControl() {
  const t = useT();
  const { mode, dirty, isSubmitting, onSave } = useLibraryStudio();

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
      onClick={onSave}
      isLoading={isSubmitting}
      loadingText={mode === "create" ? t("Creating…") : t("Saving…")}
      className="gap-2"
    >
      {mode === "edit" && <span aria-hidden className="bg-warning size-1.5 rounded-full" />}
      {mode === "create" ? t("Create library") : t("Save changes")}
      <Kbd className={cn(kbdOnSolidClassName, "border-b")}>{formatShortcut("S")}</Kbd>
    </Button>
  );
}

/**
 * The library studio's bar: back, the import path, where it is in review, what can be done to
 * it now, the save state and the toggle for the side panel.
 */
export function LibraryTopBar({
  onSubmitForReview,
  onDecision,
  onDuplicate,
  onArchive,
}: {
  onSubmitForReview: () => void;
  onDecision: (decision: LibraryDecision) => void;
  onDuplicate: () => void;
  onArchive: () => void;
}) {
  const t = useT();
  const navigate = useNavigate();
  const { mode, library, dirty, readiness, railOpen, setRailOpen } = useLibraryStudio();
  const { control } = useFormContext<FormulaLibraryFormValues>();
  const name = useWatch({ control, name: "name" })?.trim() ?? "";
  const { allowed: canSubmit } = usePermission(Resource.FormulaLibrary, Operation.Submit);
  const { allowed: canApprove } = usePermission(Resource.FormulaLibrary, Operation.Approve);
  const { allowed: canReject } = usePermission(Resource.FormulaLibrary, Operation.Reject);
  const { allowed: canDuplicate } = usePermission(Resource.FormulaLibrary, Operation.Duplicate);
  const { allowed: canArchive } = usePermission(Resource.FormulaLibrary, Operation.Archive);

  const status = library?.status;
  const blocking = readiness.some((check) => check.status === "fail");

  return (
    <header className="bg-canvas @container flex h-13 shrink-0 items-center gap-3 border-b px-3">
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        aria-label={t("Back to formula libraries")}
        onClick={() => void navigate(formulaLibraryRoutes.list)}
      >
        <ArrowLeftIcon className="size-4" />
      </Button>
      <h1
        className={cn(
          "min-w-0 truncate font-mono text-lg font-medium",
          !name && "text-foreground-subtle",
        )}
      >
        {name ? libraryPath(name) : mode === "create" ? t("New library") : t("Untitled")}
      </h1>
      {library && (
        <>
          <span aria-hidden className="bg-border h-5 w-px shrink-0 @max-[760px]:hidden" />
          <span className="flex shrink-0 items-center gap-2 @max-[760px]:hidden">
            <StatusLabel status={library.status} />
            <span className="text-foreground-muted font-mono text-xs">
              {t("v{0}", library.currentVersionNumber)}
            </span>
            {library.approvedVersionNumber &&
              library.approvedVersionNumber !== library.currentVersionNumber && (
                <span className="text-foreground-subtle text-xs @max-[1100px]:hidden">
                  {t("templates pin up to v{0}", library.approvedVersionNumber)}
                </span>
              )}
          </span>
        </>
      )}
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
              : t("Send the saved library to a reviewer")}
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
              onClick={() => onDecision("approve")}
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
              onClick={() => onDecision("requestChanges")}
            >
              <MessageAlertSquareIcon className="size-3.5" />
              {t("Request changes")}
            </Button>
          )}
        </>
      )}

      <SaveControl />

      {mode === "edit" && (canDuplicate || canArchive || (status === "InReview" && canReject)) && (
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
              {canDuplicate && (
                <DropdownMenuItem
                  title={t("Duplicate")}
                  description={t("A draft copy under a new name")}
                  startContent={<Copy01Icon className="size-4" />}
                  onClick={onDuplicate}
                />
              )}
              {canArchive && status !== "Inactive" && (
                <DropdownMenuItem
                  title={t("Archive")}
                  description={t("No new template version can import it")}
                  startContent={<ArchiveIcon className="size-4" />}
                  onClick={onArchive}
                />
              )}
            </DropdownMenuGroup>
            {status === "InReview" && canReject && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  color="danger"
                  title={t("Reject and archive")}
                  startContent={<XCloseIcon className="size-4" />}
                  onClick={() => onDecision("reject")}
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
