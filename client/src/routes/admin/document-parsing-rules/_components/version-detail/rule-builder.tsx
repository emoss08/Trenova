import { SectionRuleEditor } from "./section-rule-editor";
import { FieldRuleEditor } from "./field-rule-editor";
import { StopRuleEditor } from "./stop-rule-editor";

export function RuleBuilder() {
  return (
    <div className="space-y-4">
      <SectionRuleEditor />
      <FieldRuleEditor />
      <StopRuleEditor />
    </div>
  );
}
