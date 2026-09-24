import type { AgentToolPolicy } from "@/lib/graphql/agent-safety";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { kindLabel, tierLabel } from "./safety-model";

/**
 * The long half of a tool's rule, opened under its row: why it is held where
 * it is, how far a call may go, and the condition a record must meet. Kept out
 * of the table so the columns stay one line tall.
 */
export function PolicyDetails({ policy }: { policy: AgentToolPolicy }) {
  const t = useT();

  return (
    <DescriptionList layout="stacked" columns={4}>
      <DescriptionItem label={t("Rationale")} span="full">
        {policy.rationale}
      </DescriptionItem>
      <DescriptionItem label={t("How far it may go")} span="full">
        {policy.explanation}
      </DescriptionItem>
      {policy.conditionDescription ? (
        <DescriptionItem label={t("Record condition")} span="full">
          {policy.conditionDescription}
        </DescriptionItem>
      ) : null}
      <DescriptionItem label={t("Kind")}>{kindLabel(t, policy.kind)}</DescriptionItem>
      <DescriptionItem label={t("Default tier")}>
        {tierLabel(t, policy.defaultTier)}
      </DescriptionItem>
      <DescriptionItem label={t("Tool maximum")}>{tierLabel(t, policy.maxTier)}</DescriptionItem>
      <DescriptionItem label={t("Produces")}>
        {policy.artifact === "" ? <DescriptionEmpty /> : policy.artifact}
      </DescriptionItem>
      <DescriptionItem label={t("Reversible")}>
        {policy.reversible ? t("Yes") : t("No")}
      </DescriptionItem>
      <DescriptionItem label={t("Safe to retry")}>
        {policy.idempotent ? t("Yes") : t("No")}
      </DescriptionItem>
    </DescriptionList>
  );
}
