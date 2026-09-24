import { useT } from "@trenova/shared/i18n/use-t";
import { SectionPanel } from "@/components/section-panel";
import { Badge } from "@trenova/shared/components/ui/badge";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { CHECK_LABEL, readCaseChecks, type CaseCheck } from "./case-checks";

function percent(value: number): string {
  return `${Math.round(value * 100)}%`;
}

function CheckValue({ check }: { check: CaseCheck }) {
  const t = useT();
  if (!check.applies) {
    return <span className="text-muted-foreground text-xs">{t("Not checked")}</span>;
  }
  if (check.kind === "hard") {
    return check.passed ? (
      <Badge variant="success">{t("Held")}</Badge>
    ) : (
      <Badge variant="danger">{t("Broken")}</Badge>
    );
  }

  return <span className="tabular-nums">{percent(check.score)}</span>;
}

/**
 * How a case replay scored: the hard checks that fail it outright, the
 * weighted checks that make up its score, and what each one found.
 */
export function CaseScore({ checks, caseScore }: { checks: unknown; caseScore: number | null }) {
  const t = useT();
  const read = readCaseChecks(checks);
  if (!read) {
    return null;
  }

  const findings = read.checks.filter((check) => check.applies && check.findings.length > 0);

  return (
    <SectionPanel
      title={t("Case score")}
      hint={caseScore === null ? undefined : percent(caseScore)}
      action={
        read.passed ? (
          <Badge variant="success">{t("Passed")}</Badge>
        ) : (
          <Badge variant="danger">
            {read.hardFailure ? t("Failed a hard check") : t("Below the pass mark")}
          </Badge>
        )
      }
      help={t(
        "A hard check that breaks scores the case zero. Otherwise the score is the weighted checks that apply, blended with a judge's score when there is one; a case passes at 80%.",
      )}
    >
      <DescriptionList layout="split" className="px-3 py-1">
        {read.checks.map((check) => (
          <DescriptionItem key={check.name} label={t(CHECK_LABEL[check.name] ?? check.name)}>
            <CheckValue check={check} />
          </DescriptionItem>
        ))}
      </DescriptionList>
      {findings.length > 0 ? (
        <ul className="border-border text-muted-foreground flex flex-col gap-0.5 border-t px-3 py-2 text-xs">
          {findings.map((check) => (
            <li key={check.name}>
              {t(CHECK_LABEL[check.name] ?? check.name)}: {check.findings.join(", ")}
            </li>
          ))}
        </ul>
      ) : null}
    </SectionPanel>
  );
}
