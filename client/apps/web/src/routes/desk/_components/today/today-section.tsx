import type { ReactNode } from "react";
import { Link } from "react-router";

/** One block of Today, the prototype's `.sec`: a quiet head, how many, and where the rest are. */
export function TodaySection({
  id,
  title,
  count,
  more,
  aside,
  children,
}: {
  id: string;
  title: string;
  /** How many the block counts, already written out ("18", "18+"); hidden at none. */
  count?: string;
  more?: { to: string; label: string; shortcut?: string };
  /** A quiet note on the right of the head, when there is no link to the rest. */
  aside?: string;
  children: ReactNode;
}) {
  return (
    <section className="dk-dy-sec" id={id} aria-labelledby={`${id}-h`}>
      <div className="dk-dy-sh">
        <h2 id={`${id}-h`}>
          {title}
          {count !== undefined && <b>{count}</b>}
        </h2>
        <span className="dk-dy-sp" />
        {more ? (
          <Link to={more.to} className="dk-dy-more ui-focus-ring">
            {more.label}
            {more.shortcut && <span className="dk-dy-kbd">{more.shortcut}</span>}
          </Link>
        ) : (
          aside && <span className="dk-dy-more">{aside}</span>
        )}
      </div>
      {children}
    </section>
  );
}

/** A block's rows while they are read: the rows' own height, in shimmer. */
export function TodayRowsSkeleton({ rows }: { rows: number }) {
  return (
    <div className="dk-dy-sk" aria-hidden>
      {Array.from({ length: rows }, (_, index) => (
        <i key={index} className="dk-sk" />
      ))}
    </div>
  );
}
