import type { CSSProperties } from "react";

const FIGURES = 5;
const CHIP_WIDTHS = [132, 118, 124, 96, 140, 120];
const BOARD_CARDS = 3;
const BAR_HEIGHTS = [88, 82, 74, 66, 58, 50];

/**
 * The tab below its scope while the analytics are read: the five figures, the comparing
 * chips and the first row of the leaderboard, each where the real one lands. It draws from
 * the page's own stylesheet so it also stands in while the tab's code is still loading.
 */
export function AnalyticsBodySkeleton() {
  return (
    <div className="an-sk" aria-busy="true">
      <div className="figs" style={{ "--n": FIGURES } as CSSProperties}>
        {Array.from({ length: FIGURES }, (_, index) => (
          <div key={index} className="fg">
            <span className="an-sk-line ui-shimmer" style={{ width: 64, height: 11 }} />
            <span className="an-sk-line ui-shimmer" style={{ width: 96, height: 26 }} />
            <span className="an-sk-line ui-shimmer" style={{ width: "70%", height: 11 }} />
          </div>
        ))}
      </div>
      <div className="an-sk-row">
        {CHIP_WIDTHS.map((width, index) => (
          <span key={index} className="an-sk-chip ui-shimmer" style={{ width }} />
        ))}
      </div>
      <span className="an-sk-line ui-shimmer" style={{ width: 96, height: 13 }} />
      <div className="an-sk-cards">
        {Array.from({ length: BOARD_CARDS }, (_, index) => (
          <div key={index} className="an-sk-card">
            <span className="an-sk-line ui-shimmer" style={{ width: "45%", height: 14 }} />
            <span className="an-sk-line ui-shimmer" style={{ width: "75%", height: 11 }} />
            <span className="an-sk-line ui-shimmer" style={{ width: "90%", height: 13 }} />
            <div className="an-sk-bars">
              {BAR_HEIGHTS.map((height) => (
                <span
                  key={height}
                  className="an-sk-bar ui-shimmer"
                  style={{ height: `${height}%` }}
                />
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

/** The whole tab while its code is read: Nova's sentence, the scope, and the body. */
export function AnalyticsTabSkeleton() {
  return (
    <div className="an-sk an-sk-tab" aria-busy="true">
      <div className="an-sk-hero">
        <span className="an-sk-line ui-shimmer" style={{ width: 120, height: 12 }} />
        <span className="an-sk-line ui-shimmer" style={{ width: "88%", height: 24 }} />
        <span className="an-sk-line ui-shimmer" style={{ width: "62%", height: 24 }} />
      </div>
      <div className="an-sk-row">
        <span className="an-sk-ctl ui-shimmer" style={{ width: 188 }} />
        <span className="an-sk-ctl ui-shimmer" style={{ width: 176 }} />
        <span className="an-sk-ctl ui-shimmer" style={{ width: 150 }} />
        <span className="an-sk-ctl ui-shimmer" style={{ width: 136 }} />
      </div>
      <AnalyticsBodySkeleton />
    </div>
  );
}
