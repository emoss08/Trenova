import type { AgentExtensionCatalogItem } from "@/types/agent-extension";
import { BrandLogo } from "@trenova/shared/components/brand-logo";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Ic } from "../kit/ic";
import { Magic } from "../kit/magic";
import { dailyUsageShare, extensionState } from "./extension-roster";

/** How long an extension counts as new after its release. */
const NEW_FOR_SECONDS = 30 * 86_400;

/** Whether an extension came out recently enough to mark as new. */
export function isNewExtension(item: AgentExtensionCatalogItem, now: number): boolean {
  return item.releasedAt > 0 && now - item.releasedAt < NEW_FOR_SECONDS;
}

/** On, waiting for a key, or new; nothing for an extension that is simply off. */
export function ExtensionStateTag({
  item,
  now,
}: {
  item: AgentExtensionCatalogItem;
  now: number;
}) {
  const t = useT();
  const state = extensionState(item);
  if (state === "on") {
    return (
      <span className="tg k">
        <i />
        {t("On")}
      </span>
    );
  }
  if (state === "needsSetup") {
    return <span className="tg w">{t("Needs a key")}</span>;
  }
  return isNewExtension(item, now) ? <span className="tg b">{t("New")}</span> : null;
}

/** The vendor's mark, drawn bare. */
export function ExtensionMark({ item, s }: { item: AgentExtensionCatalogItem; s: number }) {
  return <BrandLogo domain={item.brandDomain} name={item.vendor} size={s} className="pmk" />;
}

type ExtensionTileProps = {
  item: AgentExtensionCatalogItem;
  now: number;
  onOpen: () => void;
};

/** One extension in the marketplace: what it does, its tools, and its use today once on. */
export function ExtensionTile({ item, now, onOpen }: ExtensionTileProps) {
  const t = useT();
  const state = extensionState(item);
  const keyed = item.configSpec.some((field) => field.sensitive);

  return (
    <Magic
      className={cn("mkt-t", state === "on" && "on")}
      from="var(--brand)"
      to="var(--accent-sky)"
      label={item.name}
      onClick={onOpen}
    >
      <div className="mkt-h">
        <ExtensionMark item={item} s={40} />
        <div className="mkt-n">
          <b>{item.name}</b>
          <span>{t("{0} · {1}", item.vendor, item.categoryLabel)}</span>
        </div>
        <ExtensionStateTag item={item} now={now} />
      </div>
      <p>{item.summary}</p>
      <div className="mkt-f">
        <span className="mono">
          {item.tools.length === 1 ? t("1 tool") : t("{0} tools", item.tools.length)}
        </span>
        {!keyed && <span>{t("No key")}</span>}
        <span className="sp" />
        {state === "on" ? (
          <span className="mkt-u">
            <span className="ubar">
              <i style={{ width: `${dailyUsageShare(item) * 100}%` }} />
            </span>
            <span className="mono">{`${item.usage.requestsToday}/${item.dailyRequestLimit}`}</span>
          </span>
        ) : (
          <span className="mkt-cta">
            {state === "needsSetup" ? t("Finish setup") : t("Set up")}
            <Ic n="arrowR" s={11} />
          </span>
        )}
      </div>
    </Magic>
  );
}
