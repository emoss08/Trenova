import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { darkTheme, lightTheme } from "@/components/formula-editor/editor-theme";
import { useTheme } from "@/components/theme-provider";
import { parseEDIDocumentPayload } from "@/lib/edi/document-source";
import type {
  EDIDiagnostic,
  EDIDocumentPreview,
  EDIPartnerDocumentProfile,
  EDITemplateElement,
  EDITemplateElementBaseSource,
  EDITemplateVersion,
  UpsertEDIPartnerDocumentProfileRequest,
} from "@/types/edi";
import { AlertTriangleIcon, CopyPlusIcon } from "lucide-react";
import { type ReactNode } from "react";
import { toast } from "sonner";
import { diagnosticKey } from "../utils/edi-designer-utils";
import { formatRawX12Display } from "../utils/edi-message-utils";
import type { EDIScriptPreset } from "../../edi-script-presets";

function ScriptPresetPicker({
  title,
  presets,
  disabled,
  onApply,
}: {
  title: string;
  presets: EDIScriptPreset[];
  disabled?: boolean;
  onApply: (preset: EDIScriptPreset) => void;
}) {
  if (presets.length === 0) return null;
  return (
    <div className="space-y-2 rounded-md border bg-muted/20 p-2">
      <div className="text-xs font-semibold">{title}</div>
      <div className="space-y-1">
        {presets.map((preset) => (
          <button
            key={preset.id}
            type="button"
            disabled={disabled}
            onClick={() => onApply(preset)}
            className="flex w-full items-start gap-2 rounded-sm px-2 py-1.5 text-left hover:bg-background disabled:cursor-not-allowed disabled:opacity-50"
          >
            <CopyPlusIcon className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" />
            <span className="min-w-0">
              <span className="block text-xs font-medium">{preset.label}</span>
              <span className="block text-xs leading-snug text-muted-foreground">
                {preset.description}
              </span>
            </span>
          </button>
        ))}
      </div>
    </div>
  );
}

function PreviewPane({ preview, isLoading }: { preview?: EDIDocumentPreview; isLoading: boolean }) {
  const previewContent = preview
    ? formatRawX12Display(preview.rawX12, preview.profile?.envelope)
    : "Preview output appears here.";

  return (
    <div className="grid min-h-0 flex-1 grid-cols-[minmax(0,1fr)_300px]">
      <pre className="min-h-0 overflow-auto bg-zinc-950 p-3 font-mono text-xs text-zinc-100">
        {isLoading ? "Rendering preview..." : previewContent}
      </pre>
      <DiagnosticsList diagnostics={preview?.diagnostics ?? []} />
    </div>
  );
}

function DiagnosticsList({
  diagnostics,
  onSelect,
}: {
  diagnostics: EDIDiagnostic[];
  onSelect?: (diagnostic: EDIDiagnostic) => void;
}) {
  const grouped = {
    Error: diagnostics.filter((diagnostic) => diagnostic.severity === "Error"),
    Warning: diagnostics.filter((diagnostic) => diagnostic.severity === "Warning"),
    Info: diagnostics.filter((diagnostic) => diagnostic.severity === "Info"),
  };
  return (
    <div className="min-h-0 overflow-auto p-3">
      {diagnostics.length === 0 ? (
        <div className="text-sm text-muted-foreground">No diagnostics.</div>
      ) : (
        (Object.keys(grouped) as Array<keyof typeof grouped>).map((severity) =>
          grouped[severity].length > 0 ? (
            <div key={severity} className="mb-3 space-y-2">
              <div className="text-xs font-semibold">{severity}</div>
              {grouped[severity].map((diagnostic) => (
                <button
                  key={diagnosticKey(diagnostic)}
                  type="button"
                  onClick={() => onSelect?.(diagnostic)}
                  className="block w-full rounded-md border p-2 text-left hover:bg-muted"
                >
                  <div className="flex items-center gap-2">
                    <Badge variant={diagnostic.severity === "Error" ? "inactive" : "warning"}>
                      {diagnostic.severity}
                    </Badge>
                    <span className="font-mono text-xs">
                      {diagnostic.segmentId ?? diagnostic.path}
                    </span>
                  </div>
                  <div className="mt-1 text-xs">{diagnostic.message}</div>
                  {diagnostic.suggestedFix ? (
                    <div className="mt-1 text-xs text-muted-foreground">
                      {diagnostic.suggestedFix}
                    </div>
                  ) : null}
                </button>
              ))}
            </div>
          ) : null,
        )
      )}
    </div>
  );
}

function PanelHeader({ icon, title }: { icon: ReactNode; title: string }) {
  return (
    <div className="flex h-11 items-center gap-2 border-b px-3">
      <span className="text-muted-foreground [&_svg]:size-4">{icon}</span>
      <span className="text-sm font-semibold">{title}</span>
    </div>
  );
}

function ReadOnlyBanner({ reason }: { reason: string }) {
  return (
    <div className="flex items-center gap-2 border-b bg-muted/50 px-3 py-2 text-xs text-muted-foreground">
      <AlertTriangleIcon className="size-4" />
      {reason}
    </div>
  );
}

function VersionStatusBadge({ version }: { version: EDITemplateVersion }) {
  const variant =
    version.status === "Active" ? "active" : version.status === "Draft" ? "warning" : "outline";
  return <Badge variant={variant}>{version.isActive ? "Active" : version.status}</Badge>;
}

function InputBlock({
  label,
  value,
  onChange,
  disabled,
  placeholder,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  placeholder?: string;
}) {
  return (
    <div className="space-y-1">
      <Label className="text-xs text-muted-foreground">{label}</Label>
      <Input
        value={value}
        disabled={disabled}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
      />
    </div>
  );
}

function TextareaBlock({
  label,
  value,
  onChange,
  disabled,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}) {
  return (
    <div className="space-y-1">
      <Label className="text-xs text-muted-foreground">{label}</Label>
      <Textarea
        value={value}
        disabled={disabled}
        onChange={(event) => onChange(event.target.value)}
        className="min-h-24 font-mono text-xs"
      />
    </div>
  );
}

function templateElementSourceLabel(element: EDITemplateElement) {
  if (element.source === "transform") {
    const base = element.baseSource
      ? templateBaseSourceLabel(element.baseSource)
      : "No base source";
    const steps =
      element.transformPipeline.length > 0
        ? element.transformPipeline.map((step) => step.operation).join(" -> ")
        : "No transforms";
    return `${base} / ${steps}`;
  }
  if (element.source === "starlark")
    return element.starlarkFunction ?? element.starlarkScript ?? "Starlark script";
  return (
    element.fieldPath ??
    element.runtimeKey ??
    element.mappingSourcePath ??
    element.partnerSettingPath ??
    element.repeatPath ??
    element.value ??
    element.default ??
    ""
  );
}

function templateBaseSourceLabel(source: EDITemplateElementBaseSource) {
  return (
    source.fieldPath ??
    source.runtimeKey ??
    source.mappingSourcePath ??
    source.partnerSettingPath ??
    source.repeatPath ??
    source.value ??
    source.default ??
    source.source
  );
}

function profileToDraft(
  profile: EDIPartnerDocumentProfile,
): UpsertEDIPartnerDocumentProfileRequest {
  return {
    ediPartnerId: profile.ediPartnerId,
    templateId: profile.templateId,
    templateVersionId: profile.templateVersionId ?? undefined,
    name: profile.name,
    status: profile.status,
    x12VersionOverride: profile.x12VersionOverride ?? undefined,
    functionalGroupId: profile.functionalGroupId,
    envelope: profile.envelope,
    acknowledgment: profile.acknowledgment,
    validationMode: profile.validationMode,
    partnerSettings: profile.partnerSettings,
    version: profile.version,
  };
}

function parseSettings(value: string): Record<string, unknown> {
  try {
    const parsed = JSON.parse(value) as unknown;
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
  } catch {
    toast.error("Partner settings must be valid JSON");
  }
  return {};
}

function parsePayload(value: string) {
  const result = parseEDIDocumentPayload(value);
  if (!result.ok) {
    toast.error("Payload must be valid JSON");
  }
  return result;
}

function formatArgumentValue(value: unknown) {
  if (value == null) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return JSON.stringify(value);
}

function useEditorTheme() {
  const { theme } = useTheme();
  return theme === "dark" ? darkTheme : lightTheme;
}

export {
  DiagnosticsList,
  InputBlock,
  PanelHeader,
  PreviewPane,
  ReadOnlyBanner,
  ScriptPresetPicker,
  TextareaBlock,
  VersionStatusBadge,
  formatArgumentValue,
  parsePayload,
  parseSettings,
  profileToDraft,
  templateElementSourceLabel,
  useEditorTheme,
};
