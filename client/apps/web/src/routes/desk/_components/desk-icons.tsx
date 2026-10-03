import type { ReactNode } from "react";

const PATHS = {
  plus: <path d="M12 5v14M5 12h14" />,
  search: (
    <>
      <circle cx="11" cy="11" r="6.5" />
      <path d="M20 20l-4-4" />
    </>
  ),
  home: <path d="M4 11l8-6.5 8 6.5V20H4z M10 20v-5h4v5" />,
  radar: (
    <>
      <circle cx="12" cy="12" r="8" />
      <circle cx="12" cy="12" r="4" />
      <path d="M12 12l5-5" />
    </>
  ),
  inbox: <path d="M4 13l2.5-7h11L20 13v6H4z M4 13h4.5l1 2h5l1-2H20" />,
  pin: <path d="M9 4h6l-1 5 3 3v1H7v-1l3-3zM12 13v7" />,
  chevR: <path d="M9 6l6 6-6 6" />,
  chevL: <path d="M15 6l-6 6 6 6" />,
  x: <path d="M6 6l12 12M18 6L6 18" />,
  link: (
    <path d="M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1 1M14 10a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1-1" />
  ),
  ext: <path d="M14 4h6v6M20 4l-9 9M18 14v5H5V6h5" />,
  table: (
    <>
      <rect x="4" y="5" width="16" height="14" rx="2" />
      <path d="M4 10h16M10 10v9" />
    </>
  ),
  menu: <path d="M4 7h16M4 12h16M4 17h10" />,
  panel: (
    <>
      <rect x="3.5" y="4.5" width="17" height="15" rx="2" />
      <path d="M14 4.5v15" />
    </>
  ),
  layout: (
    <>
      <rect x="3.5" y="4.5" width="17" height="15" rx="2" />
      <path d="M3.5 9.5h17M9 9.5v10" />
    </>
  ),
  rail: (
    <>
      <rect x="3.5" y="4.5" width="17" height="15" rx="2" />
      <path d="M9 4.5v15" />
    </>
  ),
  check: <path d="M5 12.5l4.5 4.5L19 7" />,
  up: <path d="M12 19V5M6 11l6-6 6 6" />,
  stop: <rect x="7" y="7" width="10" height="10" rx="2" />,
  mic: (
    <>
      <rect x="9" y="4" width="6" height="11" rx="3" />
      <path d="M6 11a6 6 0 0 0 12 0M12 17v3" />
    </>
  ),
  replay: <path d="M4 12a8 8 0 1 0 2.5-5.8M4 4v4h4" />,
  download: <path d="M12 4v11M7 10.5l5 5 5-5M5 19.5h14" />,
  more: (
    <>
      <circle cx="6" cy="12" r="1" />
      <circle cx="12" cy="12" r="1" />
      <circle cx="18" cy="12" r="1" />
    </>
  ),
  receipt: <path d="M7 3.5h10v17l-2.5-1.5-2.5 1.5-2.5-1.5L7 20.5zM10 8h4M10 11.5h4" />,
  truck: (
    <path d="M3 7h11v9H3zM14 10h3.5l2.5 3v3h-6M7 18.5a1.5 1.5 0 1 0 0-.1M17 18.5a1.5 1.5 0 1 0 0-.1" />
  ),
  shield: <path d="M12 3l7 3v5.5c0 4.2-2.9 7.7-7 8.5-4.1-.8-7-4.3-7-8.5V6z" />,
  undo: <path d="M9 14l-5-5 5-5M4 9h10a6 6 0 0 1 0 12h-3" />,
  trash: <path d="M4.5 7h15M10 7V5h4v2M6.5 7l1 12h9l1-12M10 11v5M14 11v5" />,
  enter: <path d="M19 5v7a3 3 0 0 1-3 3H6M10 11l-4 4 4 4" />,
  chat: (
    <path d="M5 6.5A2.5 2.5 0 0 1 7.5 4h9A2.5 2.5 0 0 1 19 6.5v7a2.5 2.5 0 0 1-2.5 2.5H11l-4 3.5V16h0A2.5 2.5 0 0 1 5 13.5z" />
  ),
  gear: <path d="M4 7h9M17 7h3M4 17h3M11 17h9M15 5v4M9 15v4" />,
  eye: (
    <>
      <path d="M2.5 12S6 5.5 12 5.5 21.5 12 21.5 12 18 18.5 12 18.5 2.5 12 2.5 12z" />
      <circle cx="12" cy="12" r="2.8" />
    </>
  ),
  bookmark: <path d="M7 4h10v16l-5-3.5L7 20z" />,
  copy: (
    <>
      <rect x="8" y="8" width="11" height="11" rx="2" />
      <path d="M5 15V6a1 1 0 0 1 1-1h9" />
    </>
  ),
  speaker: <path d="M5 10v4h3l4 3.5v-11L8 10zM15.5 9.5a3.5 3.5 0 0 1 0 5M18 7a7 7 0 0 1 0 10" />,
  alert: (
    <>
      <circle cx="12" cy="12" r="9" />
      <path d="M12 7.5v5.5M12 16.5v.01" />
    </>
  ),
  info: (
    <>
      <circle cx="12" cy="12" r="9" />
      <path d="M12 11v5.5M12 7.5v.01" />
    </>
  ),
  headset: (
    <>
      <path d="M4.5 14v-2a7.5 7.5 0 0 1 15 0v2" />
      <rect x="4" y="13.5" width="4" height="5.5" rx="1.5" />
      <rect x="16" y="13.5" width="4" height="5.5" rx="1.5" />
    </>
  ),
  compass: (
    <>
      <circle cx="12" cy="12" r="8.5" />
      <path d="M15.5 8.5l-2 5-5 2 2-5z" />
    </>
  ),
  route: (
    <>
      <circle cx="6" cy="18" r="2" />
      <circle cx="18" cy="6" r="2" />
      <path d="M8 18h7.5a3 3 0 0 0 0-6h-7a3 3 0 0 1 0-6H16" />
    </>
  ),
  scanner: (
    <>
      <path d="M4 15h16v4H4zM6 15V9l4-4h8v10" />
      <path d="M8 18h.01" />
    </>
  ),
  file: (
    <>
      <path d="M6 3.5h8l4 4v13H6z" />
      <path d="M14 3.5v4h4M9 13h6M9 16.5h4" />
    </>
  ),
  lock: (
    <>
      <rect x="5" y="11" width="14" height="9" rx="2" />
      <path d="M8 11V8a4 4 0 0 1 8 0v3" />
    </>
  ),
} satisfies Record<string, ReactNode>;

export type DeskIconName = keyof typeof PATHS;

export function DeskIcon({
  name,
  size = 14,
  stroke = 1.7,
}: {
  name: DeskIconName;
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
      {PATHS[name]}
    </svg>
  );
}
