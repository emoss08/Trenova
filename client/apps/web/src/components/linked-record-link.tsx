import { recordPath, type RecordEntityType } from "@/config/record-links";
import { LockedReference } from "@trenova/shared/components/withheld-value";
import { useCanReadKind } from "@trenova/shared/hooks/use-can-read-kind";
import type { ReferenceKind } from "@trenova/shared/types/generated/reference-views";
import type { ReactNode } from "react";
import { Link, type LinkProps } from "react-router";

/** A kind of record that has its own page and a reference view. */
export type LinkedRecordKind = RecordEntityType & ReferenceKind;

type LinkedRecordLinkProps = Omit<LinkProps, "to" | "children"> & {
  /** The linked record's kind; its read decides whether the link is drawn. */
  kind: LinkedRecordKind;
  id: string;
  /** What names the record: its reference view's name, code or number. */
  children: ReactNode;
  /** Query parameters added to the record's address (a tab to open on). */
  extra?: Readonly<Record<string, string>>;
};

/**
 * A record shown inside another one, linking to its own page. The page needs the kind's read,
 * so without it the name is drawn as plain text with a lock and a tooltip saying why, instead
 * of a link that opens on a refusal. See docs/engineering/reference-views.md.
 */
export function LinkedRecordLink({
  kind,
  id,
  children,
  extra,
  className,
  ...props
}: LinkedRecordLinkProps) {
  const canRead = useCanReadKind(kind);

  if (!canRead) {
    return (
      <LockedReference kind={kind} className={className}>
        {children}
      </LockedReference>
    );
  }

  return (
    <Link to={recordPath(kind, id, extra)} className={className} {...props}>
      {children}
    </Link>
  );
}
