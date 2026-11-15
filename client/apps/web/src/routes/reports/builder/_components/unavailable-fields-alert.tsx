import { useT } from "@trenova/shared/i18n/use-t";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { AlertTriangleIcon } from "@trenova/shared/components/icons";
import { useCanManagePlan } from "@trenova/shared/components/plan-locked";
import { planLockedGuidance } from "@trenova/shared/lib/plan-copy";
import { Link } from "react-router";
import type { ReportIR } from "@/types/report";
import {
  isRemovable,
  planWithheld,
  unavailableRefs,
  withoutRefs,
  type UnavailableRef,
} from "@/lib/report-catalog-access";
import { pruneOutputReferences, type CatalogIndex } from "./builder-state";
import { PROTECTED_GROUP_LABELS } from "./catalog-field-tree";

function useProblemText() {
  const t = useT();

  return (problem: UnavailableRef): string => {
    const field = t("{0} on {1}", t(problem.label), t(problem.entityLabel));
    switch (problem.access) {
      case "NeedsKindRead":
        return t("{0} needs {1} read", field, t(problem.kindLabel ?? problem.entityLabel));
      case "NeedsFieldGroup":
        return t(
          "{0} needs the {1} field group",
          field,
          PROTECTED_GROUP_LABELS[problem.group ?? ""] ?? problem.group ?? "",
        );
      case "WithheldByCeiling":
        return t("{0} is above your role's sensitivity level", field);
      case "NotOnPlan":
        return problem.addOnName
          ? t("{0} is part of the {1} add-on, which is not on your plan", field, problem.addOnName)
          : t("{0} is not on your plan", field);
      default:
        return t("{0} is no longer in the catalog", field);
    }
  };
}

/**
 * A saved report that names data the reader may not use: a field their role
 * no longer reaches, one someone with a wider role added, or one from an
 * add-on the organization's plan does not include. The server refuses to
 * preview or run it, so the builder says what is out of reach and offers to
 * take out what access withholds. What the plan withholds is listed with its
 * add-on and kept: adding the add-on is the fix.
 */
export function UnavailableFieldsAlert({
  index,
  ir,
  onChange,
}: {
  index: CatalogIndex;
  ir: ReportIR;
  onChange: (ir: ReportIR) => void;
}) {
  const t = useT();
  const problemText = useProblemText();
  const canManagePlan = useCanManagePlan();

  if (!ir.entity) return null;

  const root = index.entities.get(ir.entity);
  const rootPlan = root ? undefined : planWithheld(index, ir.entity);
  if (rootPlan) {
    return (
      <Alert size="sm" variant="warning" className="rounded-none border-x-0 border-t-0">
        <AlertTriangleIcon />
        <AlertTitle>{t("This report's data is not on your plan")}</AlertTitle>
        <AlertDescription className="flex flex-col items-start gap-1.5">
          <span>
            {rootPlan.denial.addOnName
              ? t(
                  "It is built on {0}, from the {1} add-on, which is not on your plan.",
                  t(rootPlan.label),
                  rootPlan.denial.addOnName,
                )
              : t("It is built on {0}, which your plan does not include.", t(rootPlan.label))}
          </span>
          <span>{planLockedGuidance(t, canManagePlan)}</span>
        </AlertDescription>
      </Alert>
    );
  }
  if (!root?.readable) {
    return (
      <Alert size="sm" variant="destructive" className="rounded-none border-x-0 border-t-0">
        <AlertTriangleIcon />
        <AlertTitle>{t("You cannot open this report's data")}</AlertTitle>
        <AlertDescription className="flex flex-col items-start gap-1.5">
          <span>
            {t(
              "It is built on {0}, which your role cannot read. Ask an administrator for access, or start a new report.",
              root ? t(root.pluralLabel) : ir.entity,
            )}
          </span>
          <Button size="xs" variant="outline" render={<Link to="/reports/builder" />}>
            {t("New report")}
          </Button>
        </AlertDescription>
      </Alert>
    );
  }

  const problems = unavailableRefs(index, ir);
  if (problems.length === 0) return null;
  const removable = problems.filter(isRemovable);
  const planned = removable.length < problems.length;

  return (
    <Alert size="sm" variant="warning" className="rounded-none border-x-0 border-t-0">
      <AlertTriangleIcon />
      <AlertTitle>
        {t(
          "{0, plural, one {# field in this report is unavailable to you} other {# fields in this report are unavailable to you}}",
          problems.length,
        )}
      </AlertTitle>
      <AlertDescription className="flex flex-col items-start gap-1.5">
        <ul className="flex list-disc flex-col gap-0.5 pl-4">
          {problems.map((problem) => (
            <li key={[...(problem.ref.path ?? []), problem.ref.field].join(".")}>
              {problemText(problem)}
            </li>
          ))}
        </ul>
        {planned && <span>{planLockedGuidance(t, canManagePlan)}</span>}
        <span>
          {removable.length > 0
            ? t("The preview and runs are refused until they are removed or your access changes.")
            : t("The preview and runs are refused until your plan includes them.")}
        </span>
        {removable.length > 0 && (
          <Button
            size="xs"
            variant="outline"
            onClick={() => onChange(pruneOutputReferences(withoutRefs(ir, removable)))}
          >
            {t("Remove unavailable fields")}
          </Button>
        )}
      </AlertDescription>
    </Alert>
  );
}
