import { ErrorMessage } from "@/components/fields/field-components";
import {
  ALL_AREAS,
  MAX_PINNED_TOOLS,
  allowTool,
  areaLabel,
  isSwitchedOff,
  withAreas,
} from "@/lib/agent-scope";
import type { ToolCatalogEntry } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useController, useFormContext } from "react-hook-form";
import { Callout } from "../../edit/callout";
import { Ic } from "../../kit/ic";
import { Switch } from "../../kit/layout";
import type { AgentFormValues } from "../agent-form-schema";
import { areasOf } from "../tool-catalog";
import { useDraftField } from "./block";
import { capabilityTitle } from "./tool-picker-model";

type ScopePanelProps = {
  /** The tools an administrator may give or take: the catalog less those every agent holds. */
  selectable: readonly ToolCatalogEntry[];
  catalog: readonly ToolCatalogEntry[];
  onChoose: () => void;
};

/**
 * What the agent may do: everything the person it works for may, or only the areas and
 * capabilities chosen here; what is switched off whatever else allows it; and the
 * capabilities it keeps at hand on every turn.
 */
export function ScopePanel({ selectable, catalog, onChoose }: ScopePanelProps) {
  const t = useT();
  const { control } = useFormContext<AgentFormValues>();
  const {
    field: { value: scope, onChange: setScope },
    fieldState,
  } = useController({ control, name: "scope" });
  const [limited, setLimited] = useDraftField("limited");
  const [pinned, setPinned] = useDraftField("pinnedTools");
  const areas = areasOf(selectable);
  const title = (name: string) => capabilityTitle(name, catalog, t);
  const switchedOffAreas = scope.denyAreas.filter((area) => area !== ALL_AREAS);
  const off = scope.denyCapabilities.length + switchedOffAreas.length;

  const setEverything = (everything: boolean) => {
    setLimited(!everything);
    if (everything) {
      setScope({ ...scope, areas: [], capabilities: [] });
    }
  };
  const toggleArea = (area: string) => {
    setScope(
      withAreas(
        scope,
        scope.areas.includes(area)
          ? scope.areas.filter((entry) => entry !== area)
          : [...scope.areas, area],
      ),
    );
  };
  const turnBackOn = (name: string) => setScope(allowTool(scope, name, limited));
  const allowArea = (area: string) =>
    setScope({ ...scope, denyAreas: scope.denyAreas.filter((entry) => entry !== area) });

  return (
    <div className="scp">
      {isSwitchedOff(scope) && (
        <div className="scp-w">
          <Callout
            tone="w"
            action={
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => allowArea(ALL_AREAS)}
              >
                {t("Switch it back on")}
              </Button>
            }
          >
            {t("Every area is switched off, so this agent can do nothing.")}
          </Callout>
        </div>
      )}
      <span className="bf-k">{t("May do")}</span>
      <div className="scp-e">
        <Switch
          on={!limited}
          label={t("Everything the person it works for may do")}
          onChange={setEverything}
        />
        <span>
          <b>{t("Everything the person it works for may do")}</b>
          <em>
            {limited
              ? t("Only the areas and capabilities chosen below.")
              : t("It never does more than that person may, and never past its ceiling.")}
          </em>
        </span>
      </div>
      {limited && (
        <>
          <span className="bf-k">{t("Areas")}</span>
          <div className="bf-c" role="group" aria-label={t("Areas")}>
            {areas.map((area) => {
              const on = scope.areas.includes(area);
              return (
                <button
                  key={area}
                  type="button"
                  className={cn("tkb", on && "on")}
                  aria-pressed={on}
                  onClick={() => toggleArea(area)}
                >
                  <Ic n={on ? "check" : "plus"} s={11} w={2.2} />
                  {areaLabel(area)}
                </button>
              );
            })}
          </div>
          <span className="bf-k">{t("Also allowed")}</span>
          <div className="bf-c">
            {scope.capabilities.map((name) => (
              <span key={name} className="bfc">
                {title(name)}
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  className="text-muted-foreground hover:text-foreground"
                  aria-label={t("Remove {0}", title(name))}
                  onClick={() =>
                    setScope({
                      ...scope,
                      capabilities: scope.capabilities.filter((entry) => entry !== name),
                    })
                  }
                >
                  <Ic n="x" s={10} />
                </Button>
              </span>
            ))}
            <Button type="button" variant="outline" size="sm" onClick={onChoose}>
              <Ic n="plus" s={12} />
              {t("Choose capabilities")}
            </Button>
          </div>
          {fieldState.error?.message && (
            <div className="scp-w">
              <ErrorMessage formError={fieldState.error.message} />
            </div>
          )}
        </>
      )}
      {off > 0 && (
        <>
          <span className="bf-k">{t("Switched off")}</span>
          <div className="bf-c">
            {switchedOffAreas.map((area) => (
              <span key={`area:${area}`} className="bfc">
                {areaLabel(area)}
                <em>{t("area")}</em>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  className="text-muted-foreground hover:text-foreground"
                  aria-label={t("Turn {0} back on", areaLabel(area))}
                  onClick={() => allowArea(area)}
                >
                  <Ic n="undo" s={10} />
                </Button>
              </span>
            ))}
            {scope.denyCapabilities.map((name) => (
              <span key={name} className="bfc">
                {title(name)}
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  className="text-muted-foreground hover:text-foreground"
                  aria-label={t("Turn {0} back on", title(name))}
                  onClick={() => turnBackOn(name)}
                >
                  <Ic n="undo" s={10} />
                </Button>
              </span>
            ))}
          </div>
        </>
      )}
      <span className="bf-k">{t("Kept at hand")}</span>
      <div className="bf-c">
        {pinned.length === 0 ? (
          <span className="scp-h">
            {t(
              "Pin up to {0} tools from the list below to have them on every turn.",
              MAX_PINNED_TOOLS,
            )}
          </span>
        ) : (
          <>
            {pinned.map((name) => (
              <span key={name} className="bfc">
                {title(name)}
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  className="text-muted-foreground hover:text-foreground"
                  aria-label={t("Unpin {0}", title(name))}
                  onClick={() => setPinned(pinned.filter((entry) => entry !== name))}
                >
                  <Ic n="x" s={10} />
                </Button>
              </span>
            ))}
            <span className="scp-h mono">{t("{0} of {1}", pinned.length, MAX_PINNED_TOOLS)}</span>
          </>
        )}
      </div>
    </div>
  );
}
