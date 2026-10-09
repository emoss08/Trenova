import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryState } from "nuqs";
import { useMemo, useRef, useState } from "react";
import { EXTENSION_OPEN_PARAM, extensionOpenParser } from "../../ai-control-tabs";
import { Search, useSlashFocus } from "../kit/controls";
import { Ic } from "../kit/ic";
import { Seg, Switch } from "../kit/layout";
import { ReadSheet } from "../kit/read-sheet";
import { ExtensionDemo } from "./extension-demo";
import { ExtensionDetail, useExtensionControl } from "./extension-detail";
import {
  ALL_CATEGORIES,
  dailyUsageShare,
  extensionState,
  filterExtensions,
  sortExtensions,
  type ExtensionSort,
} from "./extension-roster";
import { ExtensionMark, ExtensionStateTag, ExtensionTile } from "./extension-tile";
import { Button } from "@trenova/shared/components/ui/button";

const SORT_LABELS: Record<ExtensionSort, string> = defineLabels({
  featured: "Featured",
  newest: "Newest",
  name: "Name",
});

/** The marketplace's shelves: everything, what is on, or one category. */
type Shelf = string;

/**
 * The extension marketplace: abilities an organization turns on for its agents only,
 * each on its own account with the vendor. The featured extension leads with how agents
 * use it; a tile opens the extension to set up and to choose which agents get it.
 */
export default function ExtensionsTab() {
  const t = useT();
  const searchRef = useRef<HTMLInputElement>(null);
  useSlashFocus(searchRef);
  const catalogQuery = useQuery(queries.agentExtension.catalog());
  const agentsQuery = useQuery(queries.assistant.agents(false));
  const { allowed: canUpdate } = usePermission(Resource.AgentExtension, Operation.Update);
  const { allowed: canUpdateAgents } = usePermission(Resource.AgentDefinition, Operation.Update);

  const [query, setQuery] = useState("");
  const [shelf, setShelf] = useState<Shelf>("all");
  const [sort, setSort] = useState<ExtensionSort>("featured");
  const [openType, setOpenType] = useQueryState(EXTENSION_OPEN_PARAM, extensionOpenParser);
  const [now] = useState(() => Math.floor(Date.now() / 1000));

  const items = useMemo(() => catalogQuery.data?.items ?? [], [catalogQuery.data?.items]);
  const categories = useMemo(
    () =>
      (catalogQuery.data?.categories ?? [])
        .map((category) => ({
          ...category,
          count: items.filter((item) => item.category === category.value).length,
        }))
        .filter((category) => category.count > 0),
    [catalogQuery.data?.categories, items],
  );
  const deskAgents = useMemo(
    () => (agentsQuery.data ?? []).filter((agent) => agent.triggerMode === "Chat"),
    [agentsQuery.data],
  );
  const visible = useMemo(
    () =>
      sortExtensions(
        filterExtensions(items, {
          query,
          category: shelf === "all" || shelf === "on" ? ALL_CATEGORIES : shelf,
          status: shelf === "on" ? "on" : "all",
        }),
        sort,
      ),
    [items, query, shelf, sort],
  );
  const onCount = items.filter((item) => extensionState(item) === "on").length;
  const featured = items.find((item) => item.featured) ?? null;
  const open = items.find((item) => item.type === openType) ?? null;
  const control = useExtensionControl(open);
  const demoAgent =
    deskAgents.find((agent) => agent.enabled && agent.toolNames.length > 0)?.name ??
    t("Dispatch desk");
  const shelfTitle =
    shelf === "all"
      ? query
        ? t("Results")
        : t("All extensions")
      : shelf === "on"
        ? t("On for your agents")
        : (categories.find((category) => category.value === shelf)?.label ?? shelf);

  return (
    <div className="tabp">
      <div className="mkt-top">
        <div>
          <h2>{t("Give your agents new abilities")}</h2>
          <p>
            {t(
              "Each extension runs under your organization's own account with the vendor, and only inside AI features.",
            )}
          </p>
        </div>
        <Search
          size="lg"
          value={query}
          onChange={setQuery}
          placeholder={t("Search extensions, vendors or tools")}
          inputRef={searchRef}
        />
      </div>
      {catalogQuery.isError && (
        <div className="bnr d">
          <Ic n="alert" s={14} />
          <span>{t("The extensions could not be loaded. Reload the page to try again.")}</span>
        </div>
      )}
      {featured && shelf === "all" && !query && (
        <section className={cn("spot", extensionState(featured) === "on" && "on")}>
          <div className="spot-m">
            <span className="spot-k">{t("Featured")}</span>
            <div className="mkt-h">
              <ExtensionMark item={featured} s={48} />
              <div className="mkt-n">
                <b>{featured.name}</b>
                <span>{t("By {0} · {1}", featured.vendor, featured.categoryLabel)}</span>
              </div>
              <ExtensionStateTag item={featured} now={now} />
            </div>
            <p className="spot-p">{featured.summary}</p>
            <ul className="caps two">
              {featured.capabilities.map((capability) => (
                <li key={capability}>
                  <Ic n="check" s={12} w={2.2} />
                  {capability}
                </li>
              ))}
            </ul>
            <div className="spot-a">
              {extensionState(featured) === "on" ? (
                <>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => void setOpenType(featured.type)}
                  >
                    <Ic n="gear" s={13} />
                    {t("Manage")}
                  </Button>
                  <span className="mkt-u">
                    <span className="ubar">
                      <i style={{ width: `${dailyUsageShare(featured) * 100}%` }} />
                    </span>
                    <span className="mono">
                      {t(
                        "{0} of {1} today",
                        featured.usage.requestsToday.toLocaleString(),
                        featured.dailyRequestLimit.toLocaleString(),
                      )}
                    </span>
                  </span>
                </>
              ) : (
                <>
                  <Button
                    type="button"
                    variant="default" size="lg"
                    onClick={() => void setOpenType(featured.type)}
                  >
                    {t("Set up {0}", featured.name)}
                    <Ic n="arrowR" s={12} />
                  </Button>
                  <span className="spot-s">
                    {featured.configSpec.some((field) => field.sensitive)
                      ? t("About a minute · needs a {0} key", featured.vendor)
                      : t("About a minute · no key needed")}
                  </span>
                </>
              )}
            </div>
          </div>
          <ExtensionDemo agentName={demoAgent} />
        </section>
      )}
      <div className="mkt">
        <nav className="mkt-c" aria-label={t("Categories")}>
          <ShelfButton
            label={t("All extensions")}
            count={items.length}
            on={shelf === "all"}
            onClick={() => setShelf("all")}
          />
          <ShelfButton
            label={t("On")}
            count={onCount}
            on={shelf === "on"}
            onClick={() => setShelf("on")}
          />
          {categories.length > 0 && <span className="mkt-ch">{t("Categories")}</span>}
          {categories.map((category) => (
            <ShelfButton
              key={category.value}
              label={category.label}
              count={category.count}
              on={shelf === category.value}
              onClick={() => setShelf(category.value)}
            />
          ))}
        </nav>
        <div className="mkt-b">
          <div className="mkt-bh">
            <b>{shelfTitle}</b>
            <em className="mono">{visible.length}</em>
            <span className="sp" />
            <Seg
              className="sm"
              label={t("Sort by")}
              v={sort}
              opts={(Object.keys(SORT_LABELS) as ExtensionSort[]).map(
                (value) => [value, SORT_LABELS[value]] as const,
              )}
              onChange={setSort}
            />
          </div>
          {catalogQuery.isLoading ? (
            <div className="mkt-g" aria-busy>
              <div className="mkt-t sk ui-shimmer" />
              <div className="mkt-t sk ui-shimmer" />
            </div>
          ) : visible.length > 0 ? (
            <div className="mkt-g">
              {visible.map((item) => (
                <ExtensionTile
                  key={item.type}
                  item={item}
                  now={now}
                  onOpen={() => void setOpenType(item.type)}
                />
              ))}
            </div>
          ) : (
            <div className="dtx-e">
              <b>{shelf === "on" ? t("Nothing is on yet") : t("No extensions match")}</b>
              <span>
                {shelf === "on"
                  ? t("Set one up and it shows here.")
                  : t("Try another word or category.")}
              </span>
            </div>
          )}
          <div className="mkt-more">
            <Ic n="sparkle" s={13} />
            <span>{t("More extensions are on the way — they arrive with Trenova updates.")}</span>
          </div>
        </div>
      </div>
      <ReadSheet
        open={open !== null}
        onClose={() => void setOpenType(null)}
        label={open?.name ?? t("Extension")}
        head={
          open && (
            <>
              <ExtensionMark item={open} s={36} />
              <div className="sh-t">
                <b>{open.name}</b>
                <span>{t("By {0} · {1}", open.vendor, open.categoryLabel)}</span>
              </div>
              {extensionState(open) === "on" && canUpdate && (
                <Switch
                  on
                  label={t("Turn off")}
                  disabled={control.saving}
                  onChange={() => control.save({ enabled: false }, t("{0} is off", open.name))}
                />
              )}
            </>
          )
        }
      >
        {open && (
          <>
            <p className="sh-d">{open.summary}</p>
            <ExtensionDetail
              key={open.type}
              item={open}
              control={control}
              agents={deskAgents}
              canUpdate={canUpdate}
              canUpdateAgents={canUpdateAgents}
            />
          </>
        )}
      </ReadSheet>
    </div>
  );
}

function ShelfButton({
  label,
  count,
  on,
  onClick,
}: {
  label: string;
  count: number;
  on: boolean;
  onClick: () => void;
}) {
  return (
    <button type="button" className={cn(on && "on")} aria-pressed={on} onClick={onClick}>
      <span>{label}</span>
      <em className="mono">{count}</em>
    </button>
  );
}
