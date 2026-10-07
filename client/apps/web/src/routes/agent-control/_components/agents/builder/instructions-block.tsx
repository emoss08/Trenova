import { describeToolCall } from "@/components/assistant/tool-presentation";
import type { InstructionFinding } from "@/lib/graphql/agent-builder";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { Ic } from "../../kit/ic";
import { Blk, useDraftField } from "./block";

/** The variables an instruction may name, filled in when the agent runs. */
export const INSTRUCTION_VARIABLES = [
  "{{organization}}",
  "{{user.name}}",
  "{{user.role}}",
  "{{today}}",
] as const;

/** Roughly how many characters a model's token holds. */
const CHARS_PER_TOKEN = 4;

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
  const [instructions, setInstructions] = useDraftField("instructions");
  const area = useRef<HTMLTextAreaElement>(null);
  const variableLabels: Record<(typeof INSTRUCTION_VARIABLES)[number], string> = {
    "{{organization}}": t("Organization"),
    "{{user.name}}": t("Person asking"),
    "{{user.role}}": t("Their role"),
    "{{today}}": t("Today's date"),
  };

  useLayoutEffect(() => {
    const element = area.current;
    if (element) {
      element.style.height = "auto";
      element.style.height = `${element.scrollHeight}px`;
    }
  }, [instructions]);

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
      <div className={cn("ins", tightening && "busy")}>
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
              {tightening ? <i className="spn" /> : <Ic n="sparkle" s={12} />}
              {tightening ? t("Tightening…") : t("Tighten")}
            </button>
          )}
        </div>
        <div className="ins-ed">
          <div className="ins-bk" aria-hidden>
            {highlight(instructions)}
            {"\n"}
          </div>
          <textarea
            ref={area}
            value={instructions}
            spellCheck={false}
            aria-label={t("Instructions")}
            placeholder={t(
              "You are the detention desk for {{organization}}.\nWhen a truck has waited more than two hours…",
            )}
            onChange={(event) => setInstructions(event.target.value)}
          />
        </div>
      </div>
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
            <button type="button" className="btn sm" onClick={() => onGiveTool(finding.tools[0]!)}>
              <Ic n="plus" s={11} />
              {t("Give it {0}", describeToolCall(finding.tools[0], null).title)}
            </button>
          )}
        </div>
      ))}
      <NeverList />
    </Blk>
  );
}

function highlight(text: string): ReactNode[] {
  return text.split(/(\{\{[^}]+\}\})/g).map((part, index) =>
    part.startsWith("{{") ? <mark key={index}>{part}</mark> : part,
  );
}

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
        <div key={index} className="nv2-r">
          <span className="nv2-n mono">{index + 1}</span>
          <input
            value={line}
            aria-label={t("Line {0}", index + 1)}
            onChange={(event) =>
              setGuardrails(guardrails.map((entry, at) => (at === index ? event.target.value : entry)))
            }
          />
          <button
            type="button"
            className="ib xs"
            title={t("Remove")}
            aria-label={t("Remove line {0}", index + 1)}
            onClick={() => setGuardrails(guardrails.filter((_, at) => at !== index))}
          >
            <Ic n="x" s={11} />
          </button>
        </div>
      ))}
      <div className="nv2-r add">
        <span className="nv2-n">
          <Ic n="plus" s={11} />
        </span>
        <input
          value={draft}
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
