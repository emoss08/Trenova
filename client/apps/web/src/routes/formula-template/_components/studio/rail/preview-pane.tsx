import { ControlledShipmentAutocompleteField } from "@/components/autocomplete-fields";
import { BillView } from "@/components/tariff-bill/bill-view";
import { BillViewSkeleton } from "@/components/tariff-bill/bill-view-skeleton";
import { refusalText, tariffMessageText } from "@/components/tariff-bill/refusal-text";
import { TraceView } from "@/components/tariff-bill/trace-view";
import { ProblemsList } from "@/components/tariff-studio/problems-list";
import { describeApiError } from "@/lib/api-error-message";
import { apiService } from "@/services/api";
import { useMutation } from "@tanstack/react-query";
import {
  AlertCircleIcon,
  Beaker02Icon,
  ChevronRightIcon,
  Save01Icon,
  XCircleIcon,
} from "@trenova/shared/components/icons";
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
} from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { toast } from "@trenova/shared/components/ui/toast";
import { translateMessage } from "@trenova/shared/i18n/message";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { tariffScaffoldTest } from "@trenova/shared/rules/tariff";
import type { FormulaTemplateFormValues } from "@trenova/shared/types/formula-template";
import { EMPTY_TARIFF_CONTEXT, type TariffInputValues } from "@trenova/shared/types/tariff";
import { useState } from "react";
import { useFormContext } from "react-hook-form";
import { useStudio } from "../studio-context";
import { presentInputs, type PreviewSource } from "../use-live-preview";

/**
 * "Save as test": names the test, then writes a `test` block pinning the bill the preview got.
 * For a real shipment the server builds the block from the shipment's record
 * (`POST /formula-templates/scaffold-test`); for the sample record the browser's module does.
 */
function SaveAsTest() {
  const t = useT();
  const { preview, context, rules, previewInputs, appendToSource, setTab } = useStudio();
  const { getValues } = useFormContext<FormulaTemplateFormValues>();
  const usingShipment = preview.source === "shipment" && preview.shipmentId !== "";
  const [naming, setNaming] = useState(false);
  const [name, setName] = useState("");

  const save = useMutation({
    mutationFn: async (title: string) => {
      const source = getValues("source") ?? "";
      const inputs = presentInputs(previewInputs.getValues("inputs") as TariffInputValues);
      if (usingShipment) {
        const response = await apiService.formulaTemplateService.scaffoldTest({
          source,
          shipmentId: preview.shipmentId,
          inputs,
          name: title,
        });
        return response.test;
      }
      const scaffold = tariffScaffoldTest(rules, {
        source,
        context: context ?? EMPTY_TARIFF_CONTEXT,
        record: null,
        inputs,
        name: title,
      });
      if (!scaffold.test) {
        throw new Error(
          scaffold.error
            ? translateMessage(scaffold.error.message, t)
            : t("The test could not be written."),
        );
      }
      return scaffold.test;
    },
    onSuccess: (block) => {
      appendToSource(block);
      setNaming(false);
      setName("");
      setTab("tests");
    },
    onError: (error) =>
      toast.error(t("Could not save the test"), {
        description: describeApiError(error, t("The test could not be written.")),
      }),
  });

  const fallback = usingShipment
    ? t("rates the picked shipment as billed")
    : t("bills the sample load as previewed");

  if (!naming) {
    return (
      <Button
        type="button"
        variant="ghost"
        size="xs"
        className="gap-1"
        onClick={() => {
          setName(fallback);
          setNaming(true);
        }}
      >
        <Save01Icon className="size-3" />
        {t("Save as test")}
      </Button>
    );
  }

  return (
    <form
      className="flex w-full items-center gap-1.5"
      onSubmit={(event) => {
        event.preventDefault();
        save.mutate(name.trim() || fallback);
      }}
    >
      <Input
        autoFocus
        value={name}
        onChange={(event) => setName(event.target.value)}
        aria-label={t("Test name")}
        className="h-7 text-xs md:text-xs"
        inputContainerClassName="flex-1"
      />
      <Button type="submit" size="xs" isLoading={save.isPending}>
        {t("Save")}
      </Button>
      <Button type="button" variant="ghost" size="xs" onClick={() => setNaming(false)}>
        {t("Cancel")}
      </Button>
    </form>
  );
}

/**
 * The preview: what the tariff in the editor bills one load, rated on the server: the sample
 * record with the sample inputs, or a real shipment. It shows the bill line by line, or why the
 * tariff refused, or the error that stopped it, then the trace of how it got there.
 */
export function PreviewPane() {
  const t = useT();
  const { preview, errors, reveal, jumpTo } = useStudio();
  const [traceOpen, setTraceOpen] = useState(false);
  const result = preview.result;
  const serverErrors = (result?.diagnostics ?? []).filter(
    (finding) => finding.severity === "error",
  );

  return (
    <div className="flex flex-col gap-3 p-3">
      <SegmentedControl<PreviewSource>
        aria-label={t("Preview load")}
        value={preview.source}
        onValueChange={preview.setSource}
        items={[
          { value: "sample", label: t("Sample load") },
          { value: "shipment", label: t("Real shipment") },
        ]}
      />

      {preview.source === "shipment" && (
        <ControlledShipmentAutocompleteField
          label=""
          value={preview.shipmentId}
          onValueChange={preview.setShipmentId}
          clearable
        />
      )}

      {preview.requestError && (
        <Alert variant="destructive" size="sm">
          <AlertTitle>{t("Preview could not run")}</AlertTitle>
          <AlertDescription>{preview.requestError}</AlertDescription>
          <AlertAction>
            <Button type="button" variant="outline" size="xs" onClick={preview.runNow}>
              {t("Retry")}
            </Button>
          </AlertAction>
        </Alert>
      )}

      <div
        aria-busy={preview.isPending || undefined}
        className={cn(
          "bg-card rounded-lg border",
          (result?.error || result?.refusal) && "border-danger-border",
          preview.isPending && result && "opacity-80",
        )}
      >
        {preview.blocked ? (
          <div className="text-foreground-muted flex flex-col items-center gap-2 px-4 py-8 text-center text-sm">
            <Beaker02Icon className="text-foreground-subtle size-5" />
            {errors > 0
              ? t(
                  "Fix the {0, plural, one {error} other {# errors}} in the source to see the bill.",
                  errors,
                )
              : t("Write the tariff and its bill appears here as you type.")}
            {errors > 0 && (
              <Button type="button" variant="link" size="xs" onClick={() => jumpTo("source")}>
                {t("Go to the source")}
              </Button>
            )}
          </div>
        ) : !result ? (
          preview.isPending ? (
            <BillViewSkeleton className="p-3.5" />
          ) : (
            <div className="text-foreground-muted flex flex-col items-center gap-2 px-4 py-8 text-center text-sm">
              <Beaker02Icon className="text-foreground-subtle size-5" />
              {preview.source === "shipment" && !preview.shipmentId
                ? t("Pick a shipment to rate it with the tariff in the editor.")
                : t("The bill appears here as you type.")}
            </div>
          )
        ) : serverErrors.length > 0 ? (
          <div className="flex flex-col gap-1 p-3.5">
            <span className="text-sm font-medium">
              {t("The server's check refused the source")}
            </span>
            <ProblemsList
              diagnostics={serverErrors}
              source={preview.lastRequest?.source ?? ""}
              onReveal={reveal}
              onApplyFix={() => undefined}
              className="-mx-3.5"
            />
          </div>
        ) : result.error ? (
          <div className="flex items-start gap-2 p-3.5">
            <AlertCircleIcon className="text-warning mt-0.5 size-4 shrink-0" />
            <div className="flex min-w-0 flex-col gap-1">
              <span className="text-sm font-medium">{t("The rating stopped with an error")}</span>
              <p className="text-foreground-muted text-xs break-words">
                {tariffMessageText(result.error.message, t)}
              </p>
              {result.error.range && (
                <Button
                  type="button"
                  variant="link"
                  size="xs"
                  className="self-start"
                  onClick={() => result.error?.range && reveal(result.error.range)}
                >
                  {t("Show where")}
                </Button>
              )}
            </div>
          </div>
        ) : result.refusal ? (
          <div className="flex items-start gap-2 p-3.5">
            <XCircleIcon className="text-danger mt-0.5 size-4 shrink-0" />
            <div className="flex min-w-0 flex-col gap-1">
              <span className="text-sm font-medium">
                {t("The tariff refused to rate this load")}
              </span>
              <p className="text-sm break-words">{refusalText(result.refusal, t)}</p>
              <div className="flex items-center gap-1">
                {result.refusal.range && (
                  <Button
                    type="button"
                    variant="link"
                    size="xs"
                    onClick={() => result.refusal?.range && reveal(result.refusal.range)}
                  >
                    {t("Show where")}
                  </Button>
                )}
                <SaveAsTest />
              </div>
            </div>
          </div>
        ) : result.bill ? (
          <div className="flex flex-col gap-2 p-3.5">
            <div className="flex items-center gap-2">
              <span className="text-foreground-muted text-xs">{t("Bill")}</span>
              <span className="flex-1" />
              <SaveAsTest />
            </div>
            <BillView bill={result.bill} onSelectRange={reveal} />
          </div>
        ) : (
          <p className="text-foreground-muted px-4 py-6 text-center text-sm">
            {t("The rating billed nothing.")}
          </p>
        )}
      </div>

      {result && !preview.blocked && result.trace.length > 0 && (
        <div className="flex flex-col gap-1.5">
          <button
            type="button"
            onClick={() => setTraceOpen((open) => !open)}
            aria-expanded={traceOpen}
            className="ui-focus-ring text-foreground-muted hover:text-foreground flex items-center gap-1 self-start rounded-sm text-xs"
          >
            <ChevronRightIcon
              className={cn("size-3 transition-transform", traceOpen && "rotate-90")}
            />
            {t("Trace · {0, plural, one {# step} other {# steps}}", result.trace.length)}
          </button>
          {traceOpen && (
            <TraceView trace={result.trace} lookups={result.lookups} onSelectRange={reveal} />
          )}
        </div>
      )}
    </div>
  );
}
