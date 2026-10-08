import type { AIRetrievalStatus } from "@/lib/graphql/ai-retrieval";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Ic } from "../kit/ic";
import { retrievalSentence } from "./retrieval-model";

type RetrievalHeroProps = {
  status: AIRetrievalStatus;
  /** The provider the Embedding task goes to, by name. */
  routedTo: string | null;
  canUpdate: boolean;
  busy: boolean;
  onPause: (paused: boolean) => void;
  onRoute: () => void;
  onShowFailed: () => void;
};

/** Nova's sentence on search by meaning, and the one control beside it. */
export function RetrievalHero({
  status,
  routedTo,
  canUpdate,
  busy,
  onPause,
  onRoute,
  onShowFailed,
}: RetrievalHeroProps) {
  const t = useT();
  const rt = useRichT();
  const sentence = retrievalSentence(status);
  const strong = (children: React.ReactNode) => <b>{children}</b>;
  const mono = (children: React.ReactNode) => <b className="mono">{children}</b>;

  return (
    <section className="hero rh">
      <div className="hero-s">
        <span className="who">
          <span className={cn("dm", sentence.kind === "indexing" && "spin")} />
          <b>{t("Nova")}</b>
          <span>{t("Retrieval")}</span>
        </span>
        <p className="say" key={sentence.kind}>
          {sentence.kind === "unrouted" &&
            rt(
              "Agents search by <b>keyword only</b>. Nothing handles the Embedding task, so <b>{0, plural, one {# item is} other {# items are}}</b> waiting to be searchable by meaning.",
              { b: strong },
              sentence.waiting,
            )}
          {sentence.kind === "paused" &&
            (sentence.budget
              ? rt(
                  "Indexing is <w>paused</w>: this month's budget is spent. Agents search by keyword until it resumes.",
                  { w: (children) => <b className="t-w">{children}</b> },
                )
              : rt("Indexing is <w>paused</w>. Agents search by keyword until it resumes.", {
                  w: (children) => <b className="t-w">{children}</b>,
                }))}
          {sentence.kind === "indexing" &&
            rt(
              "Indexing under <m>{0}</m>. <b>{1}</b> to go — searches use keywords for anything not done yet.",
              { m: mono, b: strong },
              sentence.model,
              formatNumber(sentence.waiting),
            )}
          {sentence.kind === "done" &&
            (sentence.failed > 0
              ? rt(
                  "Everything is searchable by meaning under <m>{0}</m>, except <f>{1, plural, one {# item that failed} other {# items that failed}}</f>.",
                  {
                    m: mono,
                    f: (children) => (
                      <span
                        role="link"
                        tabIndex={0}
                        className="ref d"
                        onClick={onShowFailed}
                        onKeyDown={(event) => {
                          if (event.key === "Enter" || event.key === " ") {
                            event.preventDefault();
                            onShowFailed();
                          }
                        }}
                      >
                        {children}
                      </span>
                    ),
                  },
                  sentence.model,
                  sentence.failed,
                )
              : rt(
                  "Everything is searchable by meaning under <m>{0}</m>.",
                  { m: mono },
                  sentence.model,
                ))}
        </p>
      </div>
      <div className="hc">
        {sentence.kind === "unrouted" ? (
          <>
            <button type="button" className="btn ink lg" onClick={onRoute}>
              {t("Route Embedding")}
            </button>
            <span>{t("Pick a provider on Providers")}</span>
          </>
        ) : (
          <>
            {canUpdate &&
              (status.settings.paused ? (
                <button
                  type="button"
                  className="btn ink lg"
                  disabled={busy}
                  onClick={() => onPause(false)}
                >
                  <Ic n="play" s={12} />
                  {t("Resume indexing")}
                </button>
              ) : (
                <button
                  type="button"
                  className="btn lg"
                  disabled={busy}
                  onClick={() => onPause(true)}
                >
                  <Ic n="pause" s={13} w={2.2} />
                  {t("Pause indexing")}
                </button>
              ))}
            {routedTo && <span>{t("Routed to {0}", routedTo)}</span>}
          </>
        )}
      </div>
    </section>
  );
}
