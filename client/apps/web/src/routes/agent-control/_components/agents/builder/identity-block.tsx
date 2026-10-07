import {
  AGENT_ACCENT_ORDER,
  AGENT_ICONS,
  AGENT_ICON_ORDER,
  type AgentAccentName,
  type AgentIconName,
} from "@/components/agent-identity/agent-identity";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type CSSProperties } from "react";
import { Ic } from "../../kit/ic";
import { ACCENT_HUE, Tile } from "../../kit/marks";
import { Pop } from "../../kit/pop";
import { useDraftField } from "./block";

/** The agent's tile, which opens its look, and its name and one-line description. */
export function IdentityBlock({ fresh }: { fresh: boolean }) {
  const t = useT();
  const [name, setName] = useDraftField("name");
  const [description, setDescription] = useDraftField("description");
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
      <div className="hero2-f">
        <input
          className="hero2-n"
          value={name}
          maxLength={100}
          aria-label={t("Name")}
          placeholder={t("Name your agent")}
          onChange={(event) => setName(event.target.value)}
        />
        <input
          className="hero2-d"
          value={description}
          aria-label={t("Description")}
          placeholder={t("Say what it does in one line")}
          onChange={(event) => setDescription(event.target.value)}
        />
      </div>
    </div>
  );
}
