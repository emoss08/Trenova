import {
  AGENT_ACCENT_ORDER,
  AGENT_ICONS,
  AGENT_ICON_ORDER,
  type AgentAccentName,
  type AgentIconName,
} from "@/components/agent-identity/agent-identity";
import { ErrorMessage } from "@/components/fields/field-components";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type CSSProperties } from "react";
import { useController, useFormContext } from "react-hook-form";
import { Ic } from "../../kit/ic";
import { ACCENT_HUE, Tile } from "../../kit/marks";
import { Pop } from "../../kit/pop";
import type { AgentFormValues } from "../agent-form-schema";
import { useDraftField } from "./block";

/** A heading typed in place: no box until it is hovered or focused. */
const HEADING_INPUT =
  "placeholder:text-muted-foreground/60 hover:bg-field focus:bg-foreground/5 -ml-2 w-full rounded-md border-0 bg-transparent px-2 py-0.5 outline-none transition-colors";

/** The agent's tile, which opens its look, and its name and one-line description. */
export function IdentityBlock({ fresh }: { fresh: boolean }) {
  const t = useT();
  const { control } = useFormContext<AgentFormValues>();
  const {
    field: { value: name, onChange: setName, onBlur: onNameBlur, ref: nameRef, name: nameField },
    fieldState: nameState,
  } = useController({ control, name: "name" });
  const {
    field: {
      value: description,
      onChange: setDescription,
      onBlur: onDescriptionBlur,
      ref: descriptionRef,
      name: descriptionField,
    },
  } = useController({ control, name: "description" });
  const [icon, setIcon] = useDraftField("icon");
  const [accent, setAccent] = useDraftField("accent");
  const [looking, setLooking] = useState(false);

  return (
    <div id="ab-who" className={cn("hero2", fresh && "fresh")}>
      <div className="rel">
        <button
          type="button"
          className="hero2-t"
          title={t("Change icon and color")}
          aria-label={t("Change icon and color")}
          aria-expanded={looking}
          onClick={() => setLooking((open) => !open)}
        >
          <Tile agent={{ name, icon, accent }} s={64} />
          <span className="hero2-e">
            <Ic n="edit" s={10} />
          </span>
        </button>
        {looking && (
          <Pop className="look" label={t("Icon and color")} onClose={() => setLooking(false)}>
            <div className="eb-ics" role="radiogroup" aria-label={t("Icon")}>
              {AGENT_ICON_ORDER.map((key: AgentIconName) => {
                const Icon = AGENT_ICONS[key];
                return (
                  <button
                    key={key}
                    type="button"
                    role="radio"
                    aria-checked={icon === key}
                    aria-label={key}
                    className={icon === key ? "on" : undefined}
                    onClick={() => setIcon(key)}
                  >
                    <Icon size={14} strokeWidth={1.7} />
                  </button>
                );
              })}
            </div>
            <div className="eb-hs" role="radiogroup" aria-label={t("Color")}>
              {AGENT_ACCENT_ORDER.map((key: AgentAccentName) => (
                <button
                  key={key}
                  type="button"
                  role="radio"
                  aria-checked={accent === key}
                  aria-label={key}
                  className={accent === key ? "on" : undefined}
                  style={{ "--h": ACCENT_HUE[key] ?? 0 } as CSSProperties}
                  onClick={() => setAccent(key)}
                />
              ))}
            </div>
          </Pop>
        )}
      </div>
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <input
          ref={nameRef}
          name={nameField}
          value={name}
          maxLength={100}
          aria-label={t("Name")}
          aria-invalid={nameState.invalid || undefined}
          placeholder={t("Name your agent")}
          className={cn(HEADING_INPUT, "text-4xl leading-tight font-semibold tracking-tight")}
          onBlur={onNameBlur}
          onChange={(event) => setName(event.target.value)}
        />
        <input
          ref={descriptionRef}
          name={descriptionField}
          value={description}
          aria-label={t("Description")}
          placeholder={t("Say what it does in one line")}
          className={cn(HEADING_INPUT, "text-muted-foreground text-lg")}
          onBlur={onDescriptionBlur}
          onChange={(event) => setDescription(event.target.value)}
        />
        {nameState.error?.message && <ErrorMessage formError={nameState.error.message} />}
      </div>
    </div>
  );
}
