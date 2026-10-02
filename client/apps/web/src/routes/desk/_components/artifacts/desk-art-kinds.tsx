import type { AssistantArtifact } from "@/types/assistant";
import type { ReactNode } from "react";

const ICONS = {
  table: (
    <>
      <rect x="3.5" y="5" width="17" height="14" rx="2" />
      <path d="M3.5 10h17M9 10v9" />
    </>
  ),
  record: (
    <>
      <rect x="3.5" y="5" width="17" height="14" rx="2" />
      <circle cx="9" cy="11" r="2" />
      <path d="M6 16c.6-1.4 1.7-2 3-2s2.4.6 3 2M14.5 10h3M14.5 13.5h3" />
    </>
  ),
  rate: (
    <>
      <path d="M6 3.5h12v17l-3-1.8-3 1.8-3-1.8-3 1.8z" />
      <path d="M9.5 9.5h5M9.5 13h5" />
    </>
  ),
  email: (
    <>
      <rect x="3.5" y="5.5" width="17" height="13" rx="2" />
      <path d="M4 7l8 6 8-6" />
    </>
  ),
  plan: (
    <>
      <path d="M9 6.5h11M9 12h11M9 17.5h11" />
      <path d="M3.8 6.5l1.2 1.2 2-2.2M3.8 12l1.2 1.2 2-2.2" />
      <circle cx="5.2" cy="17.5" r="1.3" />
    </>
  ),
  report: (
    <>
      <path d="M4 20V4M4 20h16" />
      <path d="M8 16v-4M12 16V8M16 16v-6" />
    </>
  ),
  diff: (
    <>
      <path d="M7 4v10M3.5 9.5L7 14l3.5-4.5" />
      <path d="M17 20V10M13.5 14.5L17 10l3.5 4.5" />
    </>
  ),
  doc: (
    <>
      <path d="M6 3.5h8l4 4v13H6z" />
      <path d="M14 3.5v4h4M9 12h6M9 15.5h6" />
    </>
  ),
  view: <path d="M4 5h16l-6 7.5V19l-4-2v-4.5z" />,
  decision: (
    <>
      <path d="M12 3.5l7 2.5v5.5c0 4.4-3 7.7-7 9-4-1.3-7-4.6-7-9V6z" />
      <path d="M9 12l2 2 4-4" />
    </>
  ),
  pin: <path d="M9 4h6l-1 5 3 3H7l3-3zM12 12v8" />,
  dl: <path d="M12 4v11M7.5 10.5L12 15l4.5-4.5M5 19.5h14" />,
  ext: (
    <path d="M14 4.5h5.5V10M19.5 4.5L11 13M17 14v4.5a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V8a1 1 0 0 1 1-1h4.5" />
  ),
  x: <path d="M6.5 6.5l11 11M17.5 6.5l-11 11" />,
  up: <path d="M6 15l6-6 6 6" />,
  down: <path d="M6 9l6 6 6-6" />,
  search: (
    <>
      <circle cx="11" cy="11" r="6" />
      <path d="M20 20l-4.5-4.5" />
    </>
  ),
  check: <path d="M5 12.5l4.5 4.5L19 7" />,
  truck: (
    <>
      <path d="M3.5 6.5h10v9h-10zM13.5 9.5h4l3 3v3h-7" />
      <circle cx="7" cy="17" r="1.6" />
      <circle cx="17" cy="17" r="1.6" />
    </>
  ),
  warn: (
    <>
      <path d="M12 4l9 15.5H3z" />
      <path d="M12 10v4M12 17v.01" />
    </>
  ),
  copy: (
    <>
      <rect x="8" y="8" width="11" height="11" rx="2" />
      <path d="M5 15V6a1 1 0 0 1 1-1h9" />
    </>
  ),
} satisfies Record<string, ReactNode>;

export type ArtIconName = keyof typeof ICONS;

/** The workspace's own line icons, drawn on one 24-unit grid. */
export function ArtIcon({
  name,
  size = 14,
  stroke = 1.8,
}: {
  name: ArtIconName;
  size?: number;
  stroke?: number;
}) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={stroke}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
    >
      {ICONS[name]}
    </svg>
  );
}

/** How the workspace groups the kinds it draws: one mark and one colour per family. */
export type DeskArtKind =
  | "table"
  | "record"
  | "rate"
  | "email"
  | "plan"
  | "report"
  | "diff"
  | "doc"
  | "view"
  | "decision";

const KIND_LABELS: Record<DeskArtKind, string> = {
  table: "Table",
  record: "Record",
  rate: "Rate explanation",
  email: "Email draft",
  plan: "Plan",
  report: "Report",
  diff: "Run diff",
  doc: "Document",
  view: "View",
  decision: "Decision",
};

export function deskArtKind(artifact: Pick<AssistantArtifact, "kind" | "payload">): DeskArtKind {
  switch (artifact.kind) {
    case "table_view":
      return "path" in artifact.payload ? "view" : "table";
    case "report_preview":
    case "report_run":
      return "report";
    case "entity_card":
      return "record";
    case "rate_explanation":
      return "rate";
    case "email_draft":
    case "inbound_message":
      return "email";
    case "plan":
      return "plan";
    case "run_diff":
      return "diff";
    case "navigation":
    case "dashboard_ref":
      return "view";
    case "decision_request":
      return "decision";
    case "document":
    case "briefing":
    case "draft_edit":
      return "doc";
    default:
      return "doc";
  }
}

export function deskArtKindLabel(kind: DeskArtKind): string {
  return KIND_LABELS[kind];
}

/** The kinds in the order the browser's filters list them. */
export const DESK_ART_KINDS = Object.keys(KIND_LABELS) as DeskArtKind[];

export function DeskArtKindIcon({ kind, size = 15 }: { kind: DeskArtKind; size?: number }) {
  return <ArtIcon name={kind} size={size} stroke={size <= 11 ? 2 : 1.8} />;
}
