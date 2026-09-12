import { useT } from "@trenova/shared/i18n/use-t";
import { EmptyState } from "@/components/empty-state";
import { FileTextIcon, ListFilterIcon, ScanTextIcon } from "lucide-react";
import { useCallback, useState } from "react";
import { RuleSetDetail } from "./rule-set-detail";
import { RuleSetList } from "./rule-set-list";

export default function DocumentParsingRulePageContent() {
  const t = useT();

  const [selectedRuleSetId, setSelectedRuleSetId] = useState<string | null>(null);

  const handleDeleted = useCallback(() => setSelectedRuleSetId(null), []);

  return (
    <div className="grid h-full grid-cols-[280px_1fr] overflow-hidden">
      <div className="overflow-y-auto border-r">
        <RuleSetList selectedId={selectedRuleSetId} onSelect={setSelectedRuleSetId} />
      </div>
      <div className="overflow-y-auto">
        {selectedRuleSetId ? (
          <RuleSetDetail ruleSetId={selectedRuleSetId} onDeleted={handleDeleted} />
        ) : (
          <div className="flex h-full items-center justify-center">
            <EmptyState
              title={t("Select a Rule Set")}
              description={t(
                "Choose an existing rule set from the sidebar to view and edit its parsing configuration, or create a new one to define extraction rules for a document provider.",
              )}
              icons={[ListFilterIcon, FileTextIcon, ScanTextIcon]}
            />
          </div>
        )}
      </div>
    </div>
  );
}
