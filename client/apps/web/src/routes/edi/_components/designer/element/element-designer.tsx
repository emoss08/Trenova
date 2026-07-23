import { Badge } from "@trenova/shared/components/ui/badge";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Switch } from "@trenova/shared/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { cn } from "@trenova/shared/lib/utils";
import type { EDITemplateElement, EDITemplateSegment } from "@trenova/shared/types/edi";
import { useState } from "react";
import {
  buildConditionString,
  diagnosticsForElement,
  parseConditionString,
  type ConditionDraft,
} from "../utils/edi-designer-utils";
import {
  getEDIScriptPresetsByCategory,
  insertScriptPresetCode,
  type EDIScriptPreset,
} from "../../edi-script-presets";
import { TransformPipelineEditor } from "../transforms/transform-pipeline-editor";
import { ControlledSelectField, PathReferenceField } from "../components/designer-fields";
import {
  InputBlock,
  ScriptPresetPicker,
  TextareaBlock,
  templateElementSourceLabel,
} from "../components/designer-shared";
import {
  useSelectedTemplateDesignerData,
  useSelectedTemplateDesignerSegmentElement,
  useTemplateDesignerUrlActions,
} from "@/hooks/use-template-designer-state";
import { useTemplateDesignerStore } from "@/stores/template-designer-store";
import {
  conditionModeOptions,
  conditionOperatorOptions,
  mappingEntityTypeOptions,
  templateElementSourceOptions,
} from "../utils/edi-designer-options";
import { isTemplateVersionEditable } from "../utils/edi-designer-utils";

export function ElementDesigner() {
  const { selectedVersion } = useSelectedTemplateDesignerData();
  const { selectedSegment: segment, selectedElement: element } =
    useSelectedTemplateDesignerSegmentElement();
  const metadataDraft = useTemplateDesignerStore((state) => state.metadataDraft);
  const diagnostics = useTemplateDesignerStore((state) => state.diagnostics);
  const updateMetadata = useTemplateDesignerStore((state) => state.updateMetadata);
  const updateSegment = useTemplateDesignerStore((state) => state.updateSegment);
  const updateElement = useTemplateDesignerStore((state) => state.updateElement);
  const { patchTemplateUrlState } = useTemplateDesignerUrlActions();
  const isEditable = isTemplateVersionEditable(selectedVersion);

  if (!selectedVersion || !segment) {
    return (
      <div className="flex h-full items-center justify-center p-4 text-sm text-muted-foreground">
        Select a template version and segment to edit.
      </div>
    );
  }

  return (
    <div className="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)] overflow-hidden">
      <div className="sticky top-0 z-10 grid grid-cols-4 gap-2 border-b bg-background p-3 max-xl:grid-cols-2 max-sm:grid-cols-1">
        <InputBlock
          label="X12 Version"
          value={metadataDraft.x12Version}
          onChange={(value) => {
            if (!isEditable) return;
            updateMetadata({ x12Version: value });
          }}
          disabled={!isEditable}
        />
        <InputBlock
          label="Functional Group"
          value={metadataDraft.functionalGroupId}
          onChange={(value) => {
            if (!isEditable) return;
            updateMetadata({ functionalGroupId: value });
          }}
          disabled={!isEditable}
        />
        <InputBlock
          label="Notes"
          value={metadataDraft.versionNotes}
          onChange={(value) => {
            if (!isEditable) return;
            updateMetadata({ versionNotes: value });
          }}
          disabled={!isEditable}
        />
        <InputBlock
          label="Segment Condition"
          value={segment.condition ?? ""}
          onChange={(condition) => {
            if (!isEditable) return;
            updateSegment(segment.id, (current) => ({ ...current, condition }));
          }}
          disabled={!isEditable}
        />
      </div>
      <div className="grid min-h-0 grid-cols-[minmax(0,1fr)_360px] overflow-hidden max-lg:grid-cols-1">
        <ScrollArea className="min-h-0" viewportClassName="min-h-0">
          <div className="p-3">
            <div className="mb-3 flex flex-wrap items-center gap-2">
              <Badge variant={segment.required ? "active" : "outline"}>{segment.segmentId}</Badge>
              <Badge variant="outline">{segment.required ? "Required" : "Optional"}</Badge>
              <div>
                <div className="text-sm font-semibold">{segment.name}</div>
                <div className="text-xs text-muted-foreground">
                  Sequence {segment.sequence}
                  {segment.repeatPath ? ` / repeats ${segment.repeatPath}` : ""}
                </div>
              </div>
            </div>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-14">Pos</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Source</TableHead>
                  <TableHead>Path / Value</TableHead>
                  <TableHead className="w-20">Issues</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {segment.elements.map((item) => {
                  const itemDiagnostics = diagnosticsForElement(diagnostics, segment, item);
                  return (
                    <TableRow
                      key={`${segment.id}-${item.position}`}
                      onClick={() => patchTemplateUrlState({ elementPosition: item.position })}
                      className={cn(
                        "cursor-pointer",
                        element?.position === item.position && "bg-muted",
                      )}
                    >
                      <TableCell className="font-mono">{item.position}</TableCell>
                      <TableCell>{item.name}</TableCell>
                      <TableCell>
                        <Badge variant={item.validation.required ? "warning" : "outline"}>
                          {item.source}
                        </Badge>
                      </TableCell>
                      <TableCell className="font-mono text-xs">
                        {templateElementSourceLabel(item)}
                      </TableCell>
                      <TableCell>
                        {itemDiagnostics.length > 0 ? (
                          <Badge
                            variant={
                              itemDiagnostics.some((diagnostic) => diagnostic.severity === "Error")
                                ? "inactive"
                                : "warning"
                            }
                          >
                            {itemDiagnostics.length}
                          </Badge>
                        ) : null}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>
        </ScrollArea>
        <ElementInspector
          segment={segment}
          element={element}
          isEditable={isEditable}
          onChange={(position, updater) => updateElement(segment.id, position, updater)}
        />
      </div>
    </div>
  );
}

function ElementInspector({
  segment,
  element,
  isEditable,
  onChange,
}: {
  segment: EDITemplateSegment;
  element?: EDITemplateElement;
  isEditable: boolean;
  onChange: (
    position: number,
    updater: (element: EDITemplateElement) => EDITemplateElement,
  ) => void;
}) {
  if (!element) {
    return (
      <div className="flex items-center justify-center border-l p-4 text-sm text-muted-foreground">
        Select an element.
      </div>
    );
  }
  const update = (patch: Partial<EDITemplateElement>) =>
    onChange(element.position, (current) => ({ ...current, ...patch }));

  return (
    <ScrollArea
      className="min-h-0 border-l max-lg:border-t max-lg:border-l-0"
      viewportClassName="min-h-0"
    >
      <div className="space-y-3 p-3">
        <div>
          <div className="text-sm font-semibold">
            {segment.segmentId}
            {element.position.toString().padStart(2, "0")} {element.name}
          </div>
          <div className="text-xs text-muted-foreground">Element source and validation rules</div>
        </div>
        <ControlledSelectField
          label="Source"
          value={element.source}
          onValueChange={(source) => update({ source: source as EDITemplateElement["source"] })}
          disabled={!isEditable}
          options={templateElementSourceOptions}
        />
        <SourceEditor element={element} isEditable={isEditable} onChange={update} />
        <ConditionEditor
          key={`${segment.id}-${element.position}`}
          condition={element.condition ?? ""}
          disabled={!isEditable}
          onChange={(condition) => update({ condition })}
        />
        <div className="grid grid-cols-2 gap-2">
          <InputBlock
            label="Default"
            value={element.default ?? ""}
            onChange={(value) => update({ default: value })}
            disabled={!isEditable}
          />
          <InputBlock
            label="Max Length"
            value={String(element.validation.maxLength || "")}
            onChange={(value) =>
              update({
                validation: { ...element.validation, maxLength: Number(value) || 0 },
              })
            }
            disabled={!isEditable}
          />
        </div>
        <div className="flex items-center justify-between rounded-md border p-2">
          <div>
            <div className="text-xs font-medium">Required</div>
            <div className="text-xs text-muted-foreground">Backend validation rule</div>
          </div>
          <Switch
            checked={element.validation.required}
            disabled={!isEditable}
            onCheckedChange={(required) =>
              update({ validation: { ...element.validation, required } })
            }
          />
        </div>
        <TextareaBlock
          label="Implementation Guide Note"
          value={element.implementationGuideNote ?? ""}
          onChange={(value) => update({ implementationGuideNote: value })}
          disabled={!isEditable}
        />
      </div>
    </ScrollArea>
  );
}

function SourceEditor({
  element,
  isEditable,
  onChange,
}: {
  element: EDITemplateElement;
  isEditable: boolean;
  onChange: (patch: Partial<EDITemplateElement>) => void;
}) {
  if (element.source === "constant") {
    return (
      <InputBlock
        label="Value"
        value={element.value ?? ""}
        onChange={(value) => onChange({ value })}
        disabled={!isEditable}
      />
    );
  }
  if (element.source === "fieldPath") {
    return (
      <PathReferenceField
        label="Field Path"
        value={element.fieldPath ?? ""}
        onChange={(fieldPath) => onChange({ fieldPath })}
        disabled={!isEditable}
      />
    );
  }
  if (element.source === "partnerSetting") {
    return (
      <PathReferenceField
        label="Partner Setting"
        value={element.partnerSettingPath ?? ""}
        onChange={(partnerSettingPath) => onChange({ partnerSettingPath })}
        disabled={!isEditable}
        partner
      />
    );
  }
  if (element.source === "runtime") {
    return (
      <InputBlock
        label="Runtime Key"
        value={element.runtimeKey ?? ""}
        onChange={(runtimeKey) => onChange({ runtimeKey })}
        disabled={!isEditable}
      />
    );
  }
  if (element.source === "repeat") {
    return (
      <PathReferenceField
        label="Repeat Path"
        value={element.repeatPath ?? ""}
        onChange={(repeatPath) => onChange({ repeatPath })}
        disabled={!isEditable}
        sourceOnlyRepeated
      />
    );
  }
  if (element.source === "mapping") {
    return (
      <div className="space-y-2">
        <ControlledSelectField
          label="Mapping Entity"
          value={element.mappingEntityType ?? ""}
          onValueChange={(mappingEntityType) =>
            onChange({
              mappingEntityType: mappingEntityType as EDITemplateElement["mappingEntityType"],
            })
          }
          disabled={!isEditable}
          options={mappingEntityTypeOptions}
        />
        <PathReferenceField
          label="Mapping Source Path"
          value={element.mappingSourcePath ?? ""}
          onChange={(mappingSourcePath) => onChange({ mappingSourcePath })}
          disabled={!isEditable}
        />
      </div>
    );
  }
  if (element.source === "transform") {
    return <TransformPipelineEditor element={element} disabled={!isEditable} onChange={onChange} />;
  }
  const starlarkPresets = [
    ...getEDIScriptPresetsByCategory("elementValue"),
    ...getEDIScriptPresetsByCategory("repeatItem"),
  ];
  const applyStarlarkPreset = (preset: EDIScriptPreset) => {
    const patch: Partial<EDITemplateElement> = {
      starlarkScript: insertScriptPresetCode(element.starlarkScript ?? "", preset),
    };
    if (preset.recommendedFunctionName && !element.starlarkFunction?.trim()) {
      patch.starlarkFunction = preset.recommendedFunctionName;
    }
    onChange(patch);
  };

  return (
    <div className="space-y-2">
      <InputBlock
        label="Function Name"
        value={element.starlarkFunction ?? ""}
        onChange={(starlarkFunction) => onChange({ starlarkFunction })}
        disabled={!isEditable}
      />
      <TextareaBlock
        label="Inline Script"
        value={element.starlarkScript ?? ""}
        onChange={(starlarkScript) => onChange({ starlarkScript })}
        disabled={!isEditable}
      />
      <ScriptPresetPicker
        title="Presets"
        presets={starlarkPresets}
        disabled={!isEditable}
        onApply={applyStarlarkPreset}
      />
    </div>
  );
}
function ConditionEditor({
  condition,
  disabled,
  onChange,
}: {
  condition: string;
  disabled: boolean;
  onChange: (condition: string) => void;
}) {
  const [draft, setDraft] = useState<ConditionDraft>(() => parseConditionString(condition));

  const apply = (next: ConditionDraft) => {
    setDraft(next);
    onChange(buildConditionString(next));
  };
  const applyPreset = (preset: EDIScriptPreset) => {
    const next = parseConditionString(preset.code);
    if (draft.mode === "inlineStarlark" && next.mode === "inlineStarlark") {
      apply({
        mode: "inlineStarlark",
        script: insertScriptPresetCode(draft.script, { code: next.script }),
      });
      return;
    }
    apply(next);
  };

  return (
    <div className="space-y-2 rounded-md border p-2">
      <div className="text-xs font-semibold">Condition</div>
      <ScriptPresetPicker
        title="Presets"
        presets={getEDIScriptPresetsByCategory("condition")}
        disabled={disabled}
        onApply={applyPreset}
      />
      <ControlledSelectField
        label="Mode"
        value={draft.mode}
        disabled={disabled}
        onValueChange={(mode) => {
          if (mode === "none") apply({ mode: "none" });
          if (mode === "truthy") apply({ mode: "truthy", path: "" });
          if (mode === "falsey") apply({ mode: "falsey", path: "" });
          if (mode === "comparison")
            apply({ mode: "comparison", path: "", operator: "==", value: "" });
          if (mode === "starlarkFunction") apply({ mode: "starlarkFunction", functionName: "" });
          if (mode === "inlineStarlark") apply({ mode: "inlineStarlark", script: "" });
        }}
        options={conditionModeOptions}
      />
      {draft.mode === "truthy" || draft.mode === "falsey" ? (
        <InputBlock
          label="Path"
          value={draft.path}
          disabled={disabled}
          onChange={(path) => apply({ ...draft, path })}
        />
      ) : null}
      {draft.mode === "comparison" ? (
        <div className="grid grid-cols-[1fr_76px_1fr] gap-2">
          <InputBlock
            label="Path"
            value={draft.path}
            disabled={disabled}
            onChange={(path) => apply({ ...draft, path })}
          />
          <ControlledSelectField
            label="Op"
            value={draft.operator}
            disabled={disabled}
            onValueChange={(operator) => apply({ ...draft, operator: operator as "==" | "!=" })}
            options={conditionOperatorOptions}
          />
          <InputBlock
            label="Value"
            value={draft.value}
            disabled={disabled}
            onChange={(value) => apply({ ...draft, value })}
          />
        </div>
      ) : null}
      {draft.mode === "starlarkFunction" ? (
        <InputBlock
          label="Function"
          value={draft.functionName}
          disabled={disabled}
          onChange={(functionName) => apply({ ...draft, functionName })}
        />
      ) : null}
      {draft.mode === "inlineStarlark" ? (
        <TextareaBlock
          label="Script"
          value={draft.script}
          disabled={disabled}
          onChange={(script) => apply({ ...draft, script })}
        />
      ) : null}
    </div>
  );
}
