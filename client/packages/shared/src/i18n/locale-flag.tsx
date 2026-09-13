import { LOCALE_REGIONS, type Locale } from "@trenova/shared/i18n/generated/locales";
import { cn } from "@trenova/shared/lib/utils";

// Emoji flags were the obvious first choice and are unusable: Windows ships no flag emoji
// font, so every regional-indicator pair renders as two letters or as nothing at all. These
// are drawn instead, at a 4:3 viewBox, simplified to what survives being 16px wide - the US
// stars are a dot grid rather than fifty five-pointed stars, and Mexico's eagle is a mark
// rather than a crest. They are inline rather than files so the switcher needs no network,
// no asset pipeline, and nothing from a CDN the CSP would have to allow.

function UnitedStates() {
  return (
    <>
      <rect width="640" height="480" fill="#fff" />
      {[0, 2, 4, 6, 8, 10, 12].map((i) => (
        <rect key={i} y={(480 / 13) * i} width="640" height={480 / 13} fill="#b22234" />
      ))}
      <rect width="274" height={(480 / 13) * 7} fill="#3c3b6e" />
      {Array.from({ length: 5 }, (_, row) =>
        Array.from({ length: 6 }, (_, col) => (
          <circle key={`${row}-${col}`} cx={26 + col * 44} cy={26 + row * 47} r="12" fill="#fff" />
        )),
      )}
    </>
  );
}

function Mexico() {
  return (
    <>
      <rect width="213.3" height="480" fill="#006847" />
      <rect x="213.3" width="213.3" height="480" fill="#fff" />
      <rect x="426.6" width="213.4" height="480" fill="#ce1126" />
      <circle cx="320" cy="240" r="52" fill="none" stroke="#9b6a28" strokeWidth="14" />
      <path d="M320 200c26 14 34 40 22 62-14 24-44 24-58 4" fill="#7a4a1e" />
    </>
  );
}

function Taiwan() {
  return (
    <>
      <rect width="640" height="480" fill="#fe0000" />
      <rect width="320" height="240" fill="#000095" />
      {Array.from({ length: 12 }, (_, i) => (
        <rect
          key={i}
          x="150"
          y="30"
          width="20"
          height="180"
          fill="#fff"
          transform={`rotate(${i * 30} 160 120)`}
        />
      ))}
      <circle cx="160" cy="120" r="60" fill="#000095" />
      <circle cx="160" cy="120" r="50" fill="#fff" />
    </>
  );
}

function China() {
  const small = [
    { x: 160, y: 48 },
    { x: 200, y: 88 },
    { x: 200, y: 144 },
    { x: 160, y: 184 },
  ];
  return (
    <>
      <rect width="640" height="480" fill="#de2910" />
      <circle cx="80" cy="112" r="46" fill="#ffde00" />
      {small.map((s) => (
        <circle key={`${s.x}-${s.y}`} cx={s.x} cy={s.y} r="16" fill="#ffde00" />
      ))}
    </>
  );
}

const FLAGS: Record<string, () => React.JSX.Element> = {
  US: UnitedStates,
  MX: Mexico,
  TW: Taiwan,
  CN: China,
};

/**
 * LocaleFlag renders the flag of the region a translation is written for. A locale whose
 * region has no drawing falls back to the region code itself, so adding a fifth language to
 * locales.json still produces a usable switcher before anyone draws its flag.
 */
export function LocaleFlag({ locale, className }: { locale: Locale; className?: string }) {
  const region = LOCALE_REGIONS[locale];
  const Flag = FLAGS[region];

  if (!Flag) {
    return (
      <span
        aria-hidden="true"
        className={cn(
          "bg-muted text-muted-foreground inline-flex h-3.5 w-5 shrink-0 items-center",
          "justify-center rounded-[2px] text-[8px] font-semibold",
          className,
        )}
      >
        {region}
      </span>
    );
  }

  return (
    <svg
      viewBox="0 0 640 480"
      aria-hidden="true"
      focusable="false"
      className={cn("ring-border/60 h-3.5 w-5 shrink-0 rounded-[2px] ring-1", className)}
      preserveAspectRatio="xMidYMid slice"
    >
      <Flag />
    </svg>
  );
}
