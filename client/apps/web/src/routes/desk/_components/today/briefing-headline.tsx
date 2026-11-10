import type { Briefing } from "@/lib/graphql/briefing";
import type { ReactNode } from "react";
import { Link } from "react-router";

type HeadlineLink = Briefing["headlineLinks"][number];

/**
 * The morning's headline with the phrases the server linked: each opens the
 * section it is about. The server keeps only phrases that appear in the
 * headline, in order and apart, so a phrase not found from where the last one
 * ended is left as plain text rather than guessed at.
 */
export function BriefingHeadline({
  headline,
  links,
}: {
  headline: string;
  links: readonly HeadlineLink[];
}) {
  const parts: ReactNode[] = [];
  let cursor = 0;

  for (const link of links) {
    const start = headline.indexOf(link.text, cursor);
    if (start < 0 || link.text === "") {
      continue;
    }
    if (start > cursor) {
      parts.push(headline.slice(cursor, start));
    }
    parts.push(
      <Link key={start} to={link.path} className="dk-dy-lnk ui-focus-ring">
        {link.text}
      </Link>,
    );
    cursor = start + link.text.length;
  }
  if (cursor < headline.length) {
    parts.push(headline.slice(cursor));
  }

  return <>{parts}</>;
}
