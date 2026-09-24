import { useT } from "@trenova/shared/i18n/use-t";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type { AgentEvalCaseDetail } from "@/lib/graphql/agent-eval-cases";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { TriggerBadge } from "../../activity/agent-badges";
import { EvalCaseSourceBadge, EvalCaseStatusBadge } from "./eval-case-badges";
import { readFixtures, readFingerprint, readRedaction } from "./eval-case-frozen";

function Identifier({ value }: { value: string | null | undefined }) {
  return value ? <span className="font-mono text-xs">{value}</span> : <DescriptionEmpty />;
}

/**
 * Where a case came from and what was frozen when it was captured. None of it
 * is editable: the frozen input is what makes two replays comparable.
 */
export function EvalCaseProvenance({ evalCase }: { evalCase: AgentEvalCaseDetail }) {
  const t = useT();
  const fingerprint = readFingerprint(evalCase.capturedFingerprint);
  const redaction = readRedaction(evalCase.redaction);
  const fixtures = readFixtures(evalCase.toolFixtures);

  return (
    <div className="flex flex-col gap-4">
      <SectionPanel title={t("Captured from")}>
        <DescriptionList columns={2} className="p-3">
          <DescriptionItem label={t("Status")}>
            <EvalCaseStatusBadge value={evalCase.status} t={t} />
          </DescriptionItem>
          <DescriptionItem label={t("Source")}>
            <EvalCaseSourceBadge value={evalCase.source} t={t} />
          </DescriptionItem>
          <DescriptionItem label={t("Asked as")}>
            <TriggerBadge value={evalCase.trigger} t={t} />
          </DescriptionItem>
          <DescriptionItem label={t("Weight")} numeric>
            {evalCase.weight}
          </DescriptionItem>
          <DescriptionItem label={t("Conversation")}>
            <Identifier value={evalCase.sourceThreadId} />
          </DescriptionItem>
          <DescriptionItem label={t("Message")}>
            <Identifier value={evalCase.sourceMessageId} />
          </DescriptionItem>
          <DescriptionItem label={t("Proposal")}>
            <Identifier value={evalCase.sourceProposalId} />
          </DescriptionItem>
          <DescriptionItem label={t("Run")}>
            <Identifier value={evalCase.sourceRunId} />
          </DescriptionItem>
          <DescriptionItem label={t("Agent version")} numeric>
            {fingerprint ? fingerprint.definitionVersion : <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Tools held")} numeric>
            {fingerprint ? fingerprint.tools.length : evalCase.heldTools.length}
          </DescriptionItem>
          <DescriptionItem label={t("Redacted fields")} numeric>
            {redaction.length}
          </DescriptionItem>
          <DescriptionItem label={t("Earlier messages")} numeric>
            {Array.isArray(evalCase.history) ? evalCase.history.length : 0}
          </DescriptionItem>
        </DescriptionList>
      </SectionPanel>

      <SectionPanel
        title={t("Tool results it was given")}
        count={fixtures.length}
        help={t(
          "What each tool returned when the case was captured. Fields the registry marks restricted or confidential were replaced before the case was saved.",
        )}
      >
        {fixtures.length === 0 ? (
          <SectionPanelQuiet>{t("The original called no tools.")}</SectionPanelQuiet>
        ) : (
          <ul className="divide-border flex flex-col divide-y">
            {fixtures.map((fixture, index) => (
              <li
                key={`${fixture.tool}-${index}`}
                className="flex items-center justify-between gap-2 px-3 py-2"
              >
                <span className="font-mono text-xs">{fixture.tool}</span>
                {fixture.failed ? (
                  <Badge variant="danger">{t("Failed")}</Badge>
                ) : (
                  <span className="text-muted-foreground text-xs">
                    {t("{0, plural, one {# argument} other {# arguments}}", fixture.argumentCount)}
                  </span>
                )}
              </li>
            ))}
          </ul>
        )}
      </SectionPanel>
    </div>
  );
}
