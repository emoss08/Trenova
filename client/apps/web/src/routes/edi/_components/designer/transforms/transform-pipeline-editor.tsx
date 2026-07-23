import { Button } from "@trenova/shared/components/ui/button";
import type {
  EDITemplateElement,
  EDITemplateElementBaseSource,
  EDITemplateTransformStep,
} from "@trenova/shared/types/edi";
import { ShuffleIcon, Trash2Icon } from "lucide-react";
import {
  createTransformStep,
  getTransformOperationDefinition,
  transformOperationDefinitions,
} from "../utils/edi-designer-utils";
import {
  ControlledSelectField,
  PathInsertField,
  PathReferenceField,
} from "../components/designer-fields";
import {
  InputBlock,
  TextareaBlock,
  formatArgumentValue,
} from "../components/designer-shared";
import { transformBaseSourceOptions } from "../utils/edi-designer-options";

const transformOperationOptions = transformOperationDefinitions.map((definition) => ({
  value: definition.operation,
  label: definition.label,
}));

export function TransformPipelineEditor({
  element,
  disabled,
  onChange,
}: {
  element: EDITemplateElement;
  disabled: boolean;
  onChange: (patch: Partial<EDITemplateElement>) => void;
}) {
  const baseSource = element.baseSource ?? { source: "fieldPath" as const, fieldPath: "" };
  const updateBase = (patch: Partial<EDITemplateElementBaseSource>) =>
    onChange({ baseSource: { ...baseSource, ...patch } });
  const updatePipeline = (transformPipeline: EDITemplateTransformStep[]) =>
    onChange({ transformPipeline });
  return (
    <div className="space-y-3 rounded-md border p-2">
      <div className="flex items-center gap-2 text-xs font-semibold">
        <ShuffleIcon className="size-4" />
        Transform Pipeline
      </div>
      <ControlledSelectField
        label="Base Source"
        value={baseSource.source}
        onValueChange={(source) =>
          updateBase({ source: source as EDITemplateElementBaseSource["source"] })
        }
        disabled={disabled}
        options={transformBaseSourceOptions}
      />
      <BaseSourceValueEditor
        source={baseSource}
        disabled={disabled}
        onChange={updateBase}
      />
      <div className="space-y-2">
        {element.transformPipeline.map((step, index) => (
          <TransformStepEditor
            key={`${step.operation}-${index}`}
            step={step}
            index={index}
            disabled={disabled}
            onMove={(direction) => {
              const next = [...element.transformPipeline];
              const target = index + direction;
              if (target < 0 || target >= next.length) return;
              [next[index], next[target]] = [next[target], next[index]];
              updatePipeline(next);
            }}
            onRemove={() =>
              updatePipeline(
                element.transformPipeline.filter((_, itemIndex) => itemIndex !== index),
              )
            }
            onChange={(updated) =>
              updatePipeline(
                element.transformPipeline.map((item, itemIndex) =>
                  itemIndex === index ? updated : item,
                ),
              )
            }
          />
        ))}
      </div>
      <ControlledSelectField
        label="Add Operation"
        value=""
        onValueChange={(operation) => {
          if (!operation) return;
          updatePipeline([...element.transformPipeline, createTransformStep(operation)]);
        }}
        disabled={disabled}
        placeholder="Select operation"
        options={transformOperationOptions}
      />
    </div>
  );
}

function BaseSourceValueEditor({
  source,
  disabled,
  onChange,
}: {
  source: EDITemplateElementBaseSource;
  disabled: boolean;
  onChange: (patch: Partial<EDITemplateElementBaseSource>) => void;
}) {
  if (source.source === "partnerSetting") {
    return (
      <PathReferenceField
        label="Base Partner Setting"
        value={source.partnerSettingPath ?? ""}
        onChange={(partnerSettingPath) => onChange({ partnerSettingPath })}
        disabled={disabled}
        partner
      />
    );
  }
  if (source.source === "fieldPath" || source.source === "repeat" || source.source === "mapping") {
    return (
      <PathReferenceField
        label="Base Path"
        value={source.fieldPath ?? source.repeatPath ?? source.mappingSourcePath ?? ""}
        onChange={(value) => {
          if (source.source === "repeat") onChange({ repeatPath: value });
          else if (source.source === "mapping") onChange({ mappingSourcePath: value });
          else onChange({ fieldPath: value });
        }}
        disabled={disabled}
        sourceOnlyRepeated={source.source === "repeat"}
      />
    );
  }
  if (source.source === "runtime") {
    return (
      <InputBlock
        label="Base Runtime Key"
        value={source.runtimeKey ?? ""}
        onChange={(runtimeKey) => onChange({ runtimeKey })}
        disabled={disabled}
      />
    );
  }
  return (
    <InputBlock
      label="Base Value"
      value={source.value ?? ""}
      onChange={(value) => onChange({ value })}
      disabled={disabled}
    />
  );
}

function TransformStepEditor({
  step,
  index,
  disabled,
  onChange,
  onMove,
  onRemove,
}: {
  step: EDITemplateTransformStep;
  index: number;
  disabled: boolean;
  onChange: (step: EDITemplateTransformStep) => void;
  onMove: (direction: -1 | 1) => void;
  onRemove: () => void;
}) {
  const definition = getTransformOperationDefinition(step.operation);
  const setArg = (key: string, value: unknown) =>
    onChange({ ...step, arguments: { ...step.arguments, [key]: value } });
  return (
    <div className="space-y-2 rounded-md border bg-muted/20 p-2">
      <div className="flex items-center justify-between gap-2">
        <div>
          <div className="text-xs font-semibold">
            {index + 1}. {definition?.label ?? step.operation}
          </div>
          <div className="text-xs text-muted-foreground">{definition?.description}</div>
        </div>
        <div className="flex items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            disabled={disabled}
            onClick={() => onMove(-1)}
          >
            Up
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            disabled={disabled}
            onClick={() => onMove(1)}
          >
            Down
          </Button>
          <Button type="button" variant="ghost" size="icon" disabled={disabled} onClick={onRemove}>
            <Trash2Icon className="size-4" />
          </Button>
        </div>
      </div>
      {(definition?.arguments ?? []).map((argument) => {
        const raw = step.arguments[argument.key];
        const value =
          argument.kind === "json"
            ? JSON.stringify(raw ?? {}, null, 2)
            : Array.isArray(raw)
              ? raw.join(", ")
              : formatArgumentValue(raw);
        const onValueChange = (nextValue: string) => {
          if (argument.kind === "number") setArg(argument.key, Number(nextValue) || 0);
          else if (argument.kind === "boolean") setArg(argument.key, nextValue === "true");
          else if (argument.kind === "json") {
            try {
              setArg(argument.key, JSON.parse(nextValue) as unknown);
            } catch {
              setArg(argument.key, nextValue);
            }
          } else if (argument.kind === "path-list") {
            setArg(
              argument.key,
              nextValue
                .split(",")
                .map((item) => item.trim())
                .filter(Boolean),
            );
          } else {
            setArg(argument.key, nextValue);
          }
        };
        return (
          <div key={argument.key} className="space-y-1">
            {argument.kind === "path-list" ? (
              <PathInsertField
                label={argument.label}
                value={value}
                placeholder={argument.placeholder}
                disabled={disabled}
                onChange={onValueChange}
              />
            ) : argument.kind === "json" ? (
              <TextareaBlock
                label={argument.label}
                value={value}
                onChange={onValueChange}
                disabled={disabled}
              />
            ) : (
              <InputBlock
                label={argument.label}
                value={value}
                onChange={onValueChange}
                disabled={disabled}
                placeholder={argument.placeholder}
              />
            )}
          </div>
        );
      })}
    </div>
  );
}
