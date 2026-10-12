import { describeToolCall } from "@/components/assistant/tool-presentation";
import { ErrorMessage } from "@/components/fields/field-components";
import type { InstructionFinding } from "@/lib/graphql/agent-builder";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { fieldInvalidClass } from "@trenova/shared/lib/variants/field";
import { useRef, useState, type ReactNode } from "react";
import { useController, useFormContext } from "react-hook-form";
import { Ic } from "../../kit/ic";
import type { AgentFormValues } from "../agent-form-schema";
import { Blk, useDraftField } from "./block";
import { Button } from "@trenova/shared/components/ui/button";

/** The variables an instruction may name, filled in when the agent runs. */
export const INSTRUCTION_VARIABLES = [
  "{{organization}}",
  "{{user.name}}",
  "{{user.role}}",
  "{{today}}",
] as const;

/** Roughly how many characters a model's token holds. */
const CHARS_PER_TOKEN = 4;

/** The box and type the instructions are typed in, shared by the highlight drawn under them. */
const INSTRUCTION_TEXT =
  "m-0 px-4.5 pt-4 pb-4.5 text-sm leading-[1.75] whitespace-pre-wrap wrap-break-word [grid-area:1/1]";

type InstructionsBlockProps = {
  fresh: boolean;
  findings: readonly InstructionFinding[];
  onGiveTool: (tool: string) => void;
  /** Absent when no provider can rewrite instructions. */
  onTighten?: () => void;
  tightening: boolean;
};

/** How the agent thinks and talks, what its tools cannot do, and the lines it never crosses. */
export function InstructionsBlock({
  fresh,
  findings,
  onGiveTool,
  onTighten,
  tightening,
}: InstructionsBlockProps) {
  const t = useT();
  const { control } = useFormContext<AgentFormValues>();
  const {
    field: { value: instructions, onChange: setInstructions, onBlur, name },
    fieldState,
  } = useController({ control, name: "instructions" });
  const area = useRef<HTMLTextAreaElement>(null);
  const variableLabels: Record<(typeof INSTRUCTION_VARIABLES)[number], string> = {
    "{{organization}}": t("Organization"),
    "{{user.name}}": t("Person asking"),
    "{{user.role}}": t("Their role"),
    "{{today}}": t("Today's date"),
  };

  const insert = (variable: string) => {
    const element = area.current;
    const at = element?.selectionStart ?? instructions.length;
    const to = element?.selectionEnd ?? instructions.length;
    setInstructions(instructions.slice(0, at) + variable + instructions.slice(to));
    window.requestAnimationFrame(() => {
      if (!element) return;
      element.focus();
      element.selectionStart = element.selectionEnd = at + variable.length;
    });
  };

  return (
    <Blk
      id="instr"
      fresh={fresh}
      title={t("Instructions")}
      note={t("How it should think and talk. Write it the way you'd brief a new hire.")}
    >
      <div
        className={cn(
          "ui-field ui-container-focus-ring overflow-hidden rounded-lg",
          fieldState.invalid && fieldInvalidClass,
        )}
      >
        <div className="ins-tb">
          {INSTRUCTION_VARIABLES.map((variable) => (
            <button
              key={variable}
              type="button"
              className="ins-v"
              title={t("Insert {0}", variable)}
              onClick={() => insert(variable)}
            >
              <Ic n="plus" s={10} w={2.4} />
              {variableLabels[variable]}
            </button>
          ))}
          <span className="sp" />
          <span className="mono ins-n">
            {t("~{0} tokens", Math.ceil(instructions.length / CHARS_PER_TOKEN).toLocaleString())}
          </span>
          {onTighten && (
            <button
              type="button"
              className="ins-ai"
              disabled={tightening || !instructions.trim()}
              onClick={onTighten}
            >
              {tightening ? <span className="border-border-strong border-t-foreground inline-block size-3 animate-spin rounded-full border-[1.5px]" /> : <Ic n="sparkle" s={12} />}
              {tightening ? t("Tightening…") : t("Tighten")}
            </button>
          )}
        </div>
        <div
          className={cn(
            "grid transition-[opacity,filter] duration-200",
            tightening && "opacity-50 blur-[0.4px]",
          )}
        >
          <div
            aria-hidden
            className={cn(
              "text-foreground pointer-events-none overflow-hidden",
              "[&_mark]:bg-brand-subtle [&_mark]:text-brand-subtle-foreground [&_mark]:rounded-sm",
              INSTRUCTION_TEXT,
            )}
          >
            {highlight(instructions)}
            {"\n"}
          </div>
          <textarea
            ref={area}
            name={name}
            value={instructions}
            spellCheck={false}
            aria-label={t("Instructions")}
            aria-invalid={fieldState.invalid || undefined}
            placeholder={t(
              "You are the detention desk for {{organization}}.\nWhen a truck has waited more than two hours…",
            )}
            className={cn(
              "caret-foreground placeholder:text-muted-foreground min-h-45 w-full resize-none overflow-hidden border-0 bg-transparent text-transparent outline-none",
              "selection:bg-brand/30 selection:text-transparent",
              INSTRUCTION_TEXT,
            )}
            onBlur={onBlur}
            onChange={(event) => setInstructions(event.target.value)}
          />
        </div>
      </div>
      {fieldState.error?.message && <ErrorMessage formError={fieldState.error.message} />}
      {findings.map((finding) => (
        <div key={`${finding.start}:${finding.resource}:${finding.operation}`} className="lint">
          <Ic n="warn" s={13} />
          <span>
            {t(
              "Asks it to {0} {1}, but it holds no tool that can.",
              finding.operation,
              finding.resourceLabel.toLowerCase(),
            )}
          </span>
          {finding.tools[0] && (
            <Button type="button" variant="outline" size="sm" onClick={() => onGiveTool(finding.tools[0]!)}>
              <Ic n="plus" s={11} />
              {t("Give it {0}", describeToolCall(finding.tools[0], null).title)}
            </Button>
          )}
        </div>
      ))}
      <NeverList />
    </Blk>
  );
}

function highlight(text: string): ReactNode[] {
  return text
    .split(/(\{\{[^}]+\}\})/g)
    .map((part, index) => (part.startsWith("{{") ? <mark key={index}>{part}</mark> : part));
}

const GUARDRAIL_ROW =
  "group grid min-h-10 grid-cols-[22px_minmax(0,1fr)_auto] items-center gap-2 border-b border-border-subtle pr-2 pl-3.5 last:border-b-0 bg-field";

/** A line typed straight into its row: the row is the box, so the input draws none. */
const GUARDRAIL_INPUT =
  "text-foreground placeholder:text-muted-foreground h-9.5 min-w-0 border-0 bg-transparent text-sm outline-none";

/** Hard lines the agent won't cross, whatever it is asked. */
function NeverList() {
  const t = useT();
  const [guardrails, setGuardrails] = useDraftField("guardrails");
  const [draft, setDraft] = useState("");

  const add = () => {
    const line = draft.trim();
    if (!line) return;
    setGuardrails([...guardrails, line]);
    setDraft("");
  };

  return (
    <div className="nv2">
      <div className="nv2-h">
        <Ic n="ban" s={13} />
        <b>{t("Never")}</b>
        <span>{t("Hard lines it won't cross, whatever it's asked.")}</span>
      </div>
      {guardrails.map((line, index) => (
        <div key={index} className={GUARDRAIL_ROW}>
          <span className="nv2-n mono">{index + 1}</span>
          <input
            value={line}
            aria-label={t("Line {0}", index + 1)}
            className={GUARDRAIL_INPUT}
            onChange={(event) =>
              setGuardrails(
                guardrails.map((entry, at) => (at === index ? event.target.value : entry)),
              )
            }
          />
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            className="text-muted-foreground hover:text-foreground opacity-0 group-focus-within:opacity-100 group-hover:opacity-100"
            title={t("Remove")}
            aria-label={t("Remove line {0}", index + 1)}
            onClick={() => setGuardrails(guardrails.filter((_, at) => at !== index))}
          >
            <Ic n="x" s={11} />
          </Button>
        </div>
      ))}
      <div className={GUARDRAIL_ROW}>
        <span className="nv2-n">
          <Ic n="plus" s={11} />
        </span>
        <input
          value={draft}
          className={GUARDRAIL_INPUT}
          aria-label={t("Add a line")}
          placeholder={t("Add a line, like “Email a customer outside business hours”")}
          onChange={(event) => setDraft(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              add();
            }
          }}
        />
        {draft.trim() && <span className="kbd">↵</span>}
      </div>
    </div>
  );
}
