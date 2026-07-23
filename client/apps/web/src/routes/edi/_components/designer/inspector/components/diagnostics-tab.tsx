import { Badge } from "@trenova/shared/components/ui/badge";
import type { EDIInspectionDiagnostic, EDIX12Inspection } from "@trenova/shared/types/edi";
import { diagnosticKey } from "../../utils/edi-designer-utils";
import { groupDiagnostics } from "../../utils/edi-message-utils";

export default function DiagnosticsTab({
  diagnostics,
  inspection,
  onSelectSegment,
}: {
  diagnostics: EDIInspectionDiagnostic[];
  inspection: EDIX12Inspection;
  onSelectSegment: (segmentIndex: number) => void;
}) {
  const groups = groupDiagnostics(diagnostics);
  const summary = {
    errors: diagnostics.filter((diagnostic) => diagnostic.severity === "Error").length,
    warnings: diagnostics.filter((diagnostic) => diagnostic.severity === "Warning").length,
    info: diagnostics.filter((diagnostic) => diagnostic.severity === "Info").length,
  };

  if (diagnostics.length === 0) {
    return <div className="text-sm text-muted-foreground">No diagnostics.</div>;
  }

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-3 gap-2">
        <SummaryCard label="Errors" value={summary.errors} variant="inactive" />
        <SummaryCard label="Warnings" value={summary.warnings} variant="warning" />
        <SummaryCard label="Info" value={summary.info} variant="outline" />
      </div>
      <div className="space-y-2">
        {groups.map((group) => {
          const segment = findSegmentForDiagnostic(
            inspection,
            group.diagnostics[0] as EDIInspectionDiagnostic | undefined,
          );
          return (
            <button
              key={group.key}
              type="button"
              onClick={() => segment && onSelectSegment(segment.index)}
              className="block w-full rounded-md border p-3 text-left hover:bg-muted"
            >
              <div className="flex flex-wrap items-center gap-2">
                <Badge variant={group.severity === "Error" ? "inactive" : "warning"}>
                  {group.severity}
                </Badge>
                <Badge variant="outline">{diagnosticFamilyLabel(group.code)}</Badge>
                <span className="font-mono text-xs">
                  {group.segmentId || "Payload"}
                  {group.elementPosition ? `:${group.elementPosition}` : ""}
                </span>
                <span className="font-mono text-xs text-muted-foreground">{group.code}</span>
                {group.path ? (
                  <span className="font-mono text-xs text-muted-foreground">{group.path}</span>
                ) : null}
              </div>
              <div className="mt-2 space-y-2">
                {group.diagnostics.map((diagnostic) => (
                  <div key={diagnosticKey(diagnostic)} className="text-sm">
                    <div>{diagnostic.message}</div>
                    {diagnostic.suggestedFix ? (
                      <div className="text-xs text-muted-foreground">{diagnostic.suggestedFix}</div>
                    ) : null}
                  </div>
                ))}
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}

function SummaryCard({
  label,
  value,
  variant,
}: {
  label: string;
  value: number;
  variant: "inactive" | "warning" | "outline";
}) {
  return (
    <div className="rounded-md border p-3">
      <div className="text-xs text-muted-foreground">{label}</div>
      <Badge variant={variant} className="mt-2">
        {value}
      </Badge>
    </div>
  );
}

export function diagnosticFamilyLabel(code: string) {
  if (code.startsWith("x12.separator")) return "separator";
  if (code.startsWith("x12.control")) return "control";
  if (code.startsWith("x12.count")) return "count";
  if (code.startsWith("x12.envelope")) return "envelope";
  if (code.startsWith("x12.segment")) return "segment";
  if (code.startsWith("starlark_") || code.startsWith("script_")) return "starlark";
  if (code.startsWith("transform_")) return "transform";
  if (code.startsWith("condition_")) return "condition";
  if (code.includes("source")) return "source context";
  if (code.includes("partner")) return "partner setting";
  if (code.includes("required")) return "required";
  if (code.includes("max_length")) return "max length";
  if (code.includes("render")) return "render";
  return "validation";
}

function findSegmentForDiagnostic(
  inspection: EDIX12Inspection,
  diagnostic?: EDIInspectionDiagnostic,
) {
  if (!diagnostic) return undefined;
  if (diagnostic.segmentIndex) {
    return inspection.segments.find((segment) => segment.index === diagnostic.segmentIndex);
  }
  if (!diagnostic.segmentId) return undefined;
  return inspection.segments.find((segment) => segment.segmentId === diagnostic.segmentId);
}
