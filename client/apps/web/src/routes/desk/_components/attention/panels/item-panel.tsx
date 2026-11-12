import type { WatchtowerItem, WatchtowerPanel } from "@/lib/graphql/watchtower";
import { ArrowRightIcon } from "@trenova/shared/components/icons";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import { Link } from "react-router";
import { AskPanel } from "./ask-panel";
import { BooksPanel } from "./books-panel";
import { ChangePanel } from "./change-panel";
import { ClockPanel } from "./clock-panel";
import { ContactPanel } from "./contact-panel";
import { CoveragePanel } from "./coverage-panel";
import { ExpiryPanel } from "./expiry-panel";
import { FactsPanel } from "./facts-panel";
import { FilePanel } from "./file-panel";
import { FindingPanel } from "./finding-panel";
import { MessagePanel } from "./message-panel";
import { MoneyPanel } from "./money-panel";
import { PanelActionContext } from "./panel-parts";
import { PaperPanel } from "./paper-panel";
import { RunPanel } from "./run-panel";
import { SightingPanel } from "./sighting-panel";
import { StopPanel } from "./stop-panel";

/**
 * Where a kind of thing is fixed, in the words of the place: not every item
 * needs a case, and most are one visit to the page that holds them. A message
 * has none here, since Triage is offered from the bar under the item.
 */
function whereLabel(t: TranslateFn, panel: WatchtowerPanel | null | undefined): string | null {
  switch (panel?.kind) {
    case "Stop":
    case "Sighting":
    case "Contact":
    case "Paper":
      return t("Open the shipment");
    case "Clock":
      return t("Open it in detention");
    case "Expiry":
      return t("Open their record");
    case "Coverage":
      return t("Cover it in dispatch");
    case "Run":
    case "Ask":
      return t("See what the agent did");
    case "Books":
      return t("Open accounting");
    case "Money":
      return t("Open it in billing");
    case "Change":
      return t("See the change");
    case "File":
      return t("Open the file");
    case "Finding":
      return t("Open the insight");
    case "Message":
      return null;
    default:
      return t("Open the record");
  }
}

/**
 * What is happening on an item, drawn by the view made for its kind: a stop
 * on its line of time, a detention clock as a meter, missing paperwork as a
 * checklist a file is added to, a message as mail. The source says which
 * (`panel.kind`) and hands over what that view needs; an item with no panel
 * is read from its facts in the same frame.
 *
 * A new kind of view is a struct in `watchtower.Panel`, a type in the schema
 * and one component here.
 */
export function ItemPanel({
  item,
  now,
  timezone,
}: {
  item: WatchtowerItem;
  now: number;
  timezone: string;
}) {
  const t = useT();
  const label = whereLabel(t, item.panel);
  const action: ReactNode =
    label !== null && item.path !== "" ? (
      <Link
        to={item.path}
        className="flex items-center justify-between gap-2 rounded-b-[8px] border-t ui-focus-ring border-dsk-b-sub px-3.5 py-2 text-dsk-sm text-dsk-subtle hover:bg-dsk-hover hover:text-dsk-fg"
      >
        {label}
        <ArrowRightIcon aria-hidden className="size-3.5" />
      </Link>
    ) : null;

  return (
    <PanelActionContext.Provider value={action}>
      <PanelOf item={item} now={now} timezone={timezone} />
    </PanelActionContext.Provider>
  );
}

function PanelOf({ item, now, timezone }: { item: WatchtowerItem; now: number; timezone: string }) {
  const panel = item.panel;
  const time = { now, timezone };

  if (panel?.stop) return <StopPanel stop={panel.stop} {...time} />;
  if (panel?.clock) return <ClockPanel clock={panel.clock} {...time} />;
  if (panel?.paper) return <PaperPanel item={item} paper={panel.paper} {...time} />;
  if (panel?.message) return <MessagePanel message={panel.message} {...time} />;
  if (panel?.expiry) return <ExpiryPanel expiry={panel.expiry} {...time} />;
  if (panel?.coverage) return <CoveragePanel coverage={panel.coverage} {...time} />;
  if (panel?.contact) return <ContactPanel contact={panel.contact} {...time} />;
  if (panel?.run) return <RunPanel run={panel.run} {...time} />;
  if (panel?.books) return <BooksPanel books={panel.books} {...time} />;
  if (panel?.money) return <MoneyPanel money={panel.money} {...time} />;
  if (panel?.change) return <ChangePanel change={panel.change} {...time} />;
  if (panel?.ask) return <AskPanel ask={panel.ask} />;
  if (panel?.finding) return <FindingPanel finding={panel.finding} />;
  if (panel?.file) return <FilePanel file={panel.file} {...time} />;
  if (panel?.sighting) return <SightingPanel sighting={panel.sighting} {...time} />;

  return <FactsPanel item={item} {...time} />;
}
