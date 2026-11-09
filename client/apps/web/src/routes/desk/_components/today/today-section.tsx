import type { ReactNode } from "react";
import { Link } from "react-router";

/** One block of Today under the box: a quiet head, how many, and where the rest are. */
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
  count?: number;
  more?: { to: string; label: string; shortcut?: string };
  /** A quiet note on the right of the head, when there is no link to the rest. */
  aside?: string;
  children: ReactNode;
}) {
  return (
    <section className="dk-w-sec" id={id} aria-labelledby={`${id}-h`}>
      <div className="dk-w-sh">
        <h2 id={`${id}-h`}>
          {title}
          {count !== undefined && count > 0 && <b>{count}</b>}
        </h2>
        <span className="dk-w-sp" />
        {more ? (
          <Link to={more.to} className="dk-w-more ui-focus-ring">
            {more.label}
            {more.shortcut && <span className="dk-w-kbd">{more.shortcut}</span>}
          </Link>
        ) : (
          aside && <span className="dk-w-more">{aside}</span>
        )}
      </div>
      {children}
    </section>
  );
}
