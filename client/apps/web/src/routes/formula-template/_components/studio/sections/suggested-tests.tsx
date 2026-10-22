import { TariffCode } from "@/components/tariff-code/tariff-code";
import { StatusIcon } from "@/components/tariff-studio/tests-panel";
import { describeApiError } from "@/lib/api-error-message";
import { CheckIcon, PlusIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { translateMessage } from "@trenova/shared/i18n/message";
import { useT } from "@trenova/shared/i18n/use-t";
import type { SuggestedTestsResponse } from "@trenova/shared/types/formula-template";

export type SuggestedTestsState =
  | { status: "loading" }
  | { status: "ready"; data: SuggestedTestsResponse }
  | { status: "error"; error: unknown };

type SuggestedTest = SuggestedTestsResponse["tests"][number];

/**
 * The tests Nova suggested for the draft, each already run against it: a passing one pins what
 * the tariff does today, a failing one shows where Nova expected something else, with what the
 * expectation saw. Each is added to the source on its own, or every passing one at once.
 */
export function SuggestedTests({
  state,
  added,
  onAdd,
  onClose,
  onRetry,
}: {
  state: SuggestedTestsState;
  /** The names of the suggestions already added to the source. */
  added: ReadonlySet<string>;
  onAdd: (tests: SuggestedTest[]) => void;
  onClose: () => void;
  onRetry: () => void;
}) {
  const t = useT();
  const loading = state.status === "loading";
  const tests = state.status === "ready" ? state.data.tests : [];
  const addable = tests.filter((test) => test.status === "passed" && !added.has(test.name));

  return (
    <div className="bg-card overflow-hidden rounded-lg border" aria-busy={loading || undefined}>
      <div className="flex items-center gap-2 border-b px-3 py-2">
        <NovaBadge size="sm" spin={loading ? "always" : "none"} />
        <span className="text-sm font-semibold">{t("Suggested tests")}</span>
        <span className="flex-1" />
        {addable.length > 1 && (
          <Button
            type="button"
            variant="outline"
            size="xs"
            className="gap-1"
            onClick={() => onAdd(addable)}
          >
            <PlusIcon className="size-3" />
            {t("Add the {0} that pass", addable.length)}
          </Button>
        )}
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label={t("Close the suggestions")}
          onClick={onClose}
        >
          <XCloseIcon className="size-3.5" />
        </Button>
      </div>

      {loading && (
        <p className="ui-shimmer text-foreground-muted m-3 w-fit rounded-sm text-sm">
          {t("Writing tests for the tariff's edges and running them…")}
        </p>
      )}
      {state.status === "error" && (
        <div className="flex items-center gap-2 px-3 py-3">
          <p className="text-danger text-sm">
            {describeApiError(state.error, t("Nova could not suggest tests."))}
          </p>
          <Button type="button" variant="outline" size="xs" onClick={onRetry}>
            {t("Try again")}
          </Button>
        </div>
      )}
      {state.status === "ready" && tests.length === 0 && (
        <p className="text-foreground-muted px-3 py-3 text-sm">
          {t("Nova found no case the tests don't already cover.")}
        </p>
      )}

      {tests.length > 0 && (
        <ul className="divide-border divide-y">
          {tests.map((test) => (
            <li key={test.name} className="flex flex-col gap-2 px-3 py-2.5">
              <div className="flex items-center gap-2">
                <StatusIcon status={test.status} />
                <span className="min-w-0 flex-1 truncate text-sm font-medium">{test.name}</span>
                {added.has(test.name) ? (
                  <span className="text-success flex items-center gap-1 text-xs">
                    <CheckIcon className="size-3" />
                    {t("Added")}
                  </span>
                ) : (
                  <Button
                    type="button"
                    variant={test.status === "passed" ? "outline" : "quiet"}
                    size="xs"
                    className="gap-1"
                    onClick={() => onAdd([test])}
                  >
                    <PlusIcon className="size-3" />
                    {t("Add")}
                  </Button>
                )}
              </div>
              {test.status !== "passed" && (
                <div className="text-foreground-muted flex flex-col gap-1 text-xs">
                  {test.failures.length === 0 ? (
                    <span>{t("It does not run on this draft.")}</span>
                  ) : (
                    test.failures.map((failure, index) => (
                      <span
                        // oxlint-disable-next-line react/no-array-index-key -- a test's failures have no identity but their order
                        key={index}
                      >
                        <span className="text-danger">{translateMessage(failure.message, t)}</span>
                        {failure.values.length > 0 && (
                          <span className="font-mono">
                            {" "}
                            ({failure.values.map((value) => value.display).join(", ")})
                          </span>
                        )}
                      </span>
                    ))
                  )}
                </div>
              )}
              <TariffCode
                code={test.block.trim()}
                className="bg-sunken max-h-48 overflow-auto rounded-md text-xs"
              />
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
