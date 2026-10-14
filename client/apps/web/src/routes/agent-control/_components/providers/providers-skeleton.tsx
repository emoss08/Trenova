import { useT } from "@trenova/shared/i18n/use-t";
import type { CSSProperties } from "react";
import {
  HeroSkeleton,
  SecHeadSkeleton,
  SkBlock,
  SkIconButton,
  SkSearch,
  SkSwitch,
  SkText,
} from "../kit/tab-skeleton";

/** The chain, top to bottom: name, model, what each is first for, and its week. */
const CHAIN = [
  { name: 84, model: 132, role: 70 },
  { name: 104, model: 116, role: 70 },
  { name: 72, model: 148, role: 48 },
] as const;
/** A week of calls beside each provider, oldest first. */
const WEEK_HEIGHTS = [7, 11, 5, 13, 9, 16, 12] as const;
/** Every task the routing matrix lists, one row each. */
const ROUTING_ROWS = [
  { task: 104, goesTo: 84, backup: true },
  { task: 118, goesTo: 84, backup: true },
  { task: 132, goesTo: 104, backup: false },
  { task: 140, goesTo: 84, backup: true },
  { task: 128, goesTo: 104, backup: false },
  { task: 124, goesTo: 84, backup: true },
  { task: 92, goesTo: 72, backup: false },
  { task: 112, goesTo: 84, backup: true },
  { task: 96, goesTo: 104, backup: false },
  { task: 134, goesTo: 84, backup: true },
  { task: 126, goesTo: 72, backup: false },
  { task: 116, goesTo: 84, backup: true },
  { task: 70, goesTo: 104, backup: false },
  { task: 56, goesTo: 84, backup: true },
] as const;
const NEW_PROVIDER_WIDTH = 134;

/**
 * Below Nova's sentence while the providers are read: the toolbar, the chain of providers
 * in priority order, and the routing matrix of tasks against them.
 */
export function ProvidersBodySkeleton() {
  const t = useT();
  return (
    <div aria-busy="true" aria-label={t("Loading providers")}>
      <div className="tb" aria-hidden>
        <SkSearch />
        <span className="sp" />
        <span className="tb-ct mono">
          <SkText w={54} />
        </span>
        <SkBlock w={NEW_PROVIDER_WIDTH} h={32} />
      </div>
      <section className="sec chain-s" aria-hidden>
        <SecHeadSkeleton title={70} note={150} end={{ note: 96 }} />
        <ol className="pls">
          {CHAIN.map((line, index) => (
            <li key={index} className="pl tsk-still">
              <span className="pl-n mono">
                <SkBlock w={11} h={11} r={2} />
                <SkText w={14} />
              </span>
              <span className="pc-mk">
                <SkBlock w={26} h={26} />
              </span>
              <span className="pl-t">
                <b>
                  <SkText w={line.name} />
                </b>
                <span className="mono">
                  <SkText w={line.model} />
                </span>
              </span>
              <span className="pl-tg">
                <span className="pl-f">
                  <SkText w={line.role} />
                </span>
              </span>
              <span className="pl-st">
                <span className="pc-bars sm">
                  {WEEK_HEIGHTS.map((height, day) => (
                    <SkBlock key={day} w={4} h={height} r={1} />
                  ))}
                </span>
                <span className="mono pl-ok">
                  <SkText w={38} />
                </span>
                <span className="mono pl-ms">
                  <SkText w={32} />
                </span>
              </span>
              <span className="pc-c">
                <SkIconButton />
                <SkSwitch />
              </span>
            </li>
          ))}
          <li className="pl-add tsk-still">
            <SkBlock w={13} h={13} r={3} />
            <span>
              <SkText w={92} />
            </span>
          </li>
        </ol>
      </section>
      <section className="sec" aria-hidden>
        <SecHeadSkeleton title={52} note={84} />
        <p className="lead">
          <span className="tsk-ln">
            <SkText w={620} />
          </span>
        </p>
        <div className="mx-w">
          <div className="mx" style={{ "--cols": CHAIN.length } as CSSProperties}>
            <div className="mx-h">
              <span>
                <SkText w={28} />
              </span>
              {CHAIN.map((line, index) => (
                <span key={index} className="mx-ph">
                  <SkText w={6} />
                  <SkBlock w={16} h={16} r={4} />
                  <SkText w={line.name - 20} />
                </span>
              ))}
              <span>
                <SkText w={48} />
              </span>
            </div>
            {ROUTING_ROWS.map((row, index) => (
              <div key={index} className="mx-r tsk-still">
                <span className="mx-t">
                  <SkText w={row.task} />
                </span>
                {CHAIN.map((_, column) => (
                  <span key={column} className="mx-c">
                    <SkBlock w={12} h={12} r="full" />
                  </span>
                ))}
                <span className="mx-g">
                  <b>
                    <SkText w={row.goesTo} />
                  </b>
                  {row.backup && (
                    <em>
                      <SkText w={row.goesTo - 10} />
                    </em>
                  )}
                </span>
              </div>
            ))}
          </div>
        </div>
      </section>
    </div>
  );
}

/**
 * The Providers tab while its code and first reads arrive: Nova's sentence, then the body.
 * It draws the tab with providers saved; with none, the tab opens on its own empty state.
 */
export function ProvidersTabSkeleton() {
  const t = useT();
  return (
    <div className="tabp" aria-busy="true" aria-label={t("Loading providers")}>
      <HeroSkeleton />
      <ProvidersBodySkeleton />
    </div>
  );
}
