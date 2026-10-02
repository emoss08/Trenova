import type { AssistantProviderOption } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  Fragment,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent,
} from "react";
import { DeskIcon } from "../desk-icons";
import { useOutsideDismiss } from "../use-outside-dismiss";
import {
  DeskBrandMark,
  GeminiGradient,
  REEL_VENDORS,
  VENDOR_NAMES,
} from "./desk-brand-mark";

/** The marks the Auto chip's reel turns through, starting and ending on Auto. */
const CHIP_REEL = ["auto", "anthropic", "openai", "gemini", "groq", "auto"] as const;

/** A repeatable 0–1 number for column and cell, so the rain falls the same way every time. */
function seeded(column: number, salt: number): number {
  const x = Math.sin(column * 12.9898 + salt * 78.233) * 43758.5453;
  return x - Math.floor(x);
}

const RAIN = Array.from({ length: 18 }, (_, column) => ({
  cells: Array.from(
    { length: 5 },
    (_, cell) => REEL_VENDORS[Math.floor(seeded(column, cell + 3) * REEL_VENDORS.length)],
  ),
  duration: 1.6 + seeded(column, 1) * 1.8,
  delay: -seeded(column, 2) * 3,
  opacity: 0.35 + seeded(column, 9) * 0.5,
}));

/** Vendor marks falling behind the Auto row while it is pointed at: every model it can reach. */
function LogoRain() {
  return (
    <span className="dk-ai-rain" aria-hidden>
      {RAIN.map((column, index) => (
        <span
          key={index}
          className="dk-ai-col"
          style={{
            left: index * 18 + 4,
            animationDuration: `${column.duration}s`,
            animationDelay: `${column.delay}s`,
            opacity: column.opacity,
          }}
        >
          {[...column.cells, ...column.cells].map((vendor, cell) => (
            <span
              key={cell}
              className={cn("dk-ai-drop", `dk-p-${vendor}`, cell % 5 === 4 && "dk-head")}
            >
              <DeskBrandMark vendor={vendor} size={8} />
            </span>
          ))}
        </span>
      ))}
    </span>
  );
}

/** The one word that says why a person might pick this endpoint, or why they cannot. */
function endpointTag(
  option: AssistantProviderOption,
  index: number,
  t: ReturnType<typeof useT>,
): string {
  if (option.unavailable) {
    return t("Unavailable");
  }
  if (option.reasoning === "High") {
    return t("Deep reasoning");
  }
  if (option.reasoning === "Medium" || option.reasoning === "Low") {
    return t("Reasoning");
  }
  if (index === 0) {
    return t("Primary");
  }
  return t("Balanced");
}

type Row =
  | { auto: true }
  | { auto: false; option: AssistantProviderOption; order: number };

/**
 * Which model answers, as a chip in the composer.
 *
 * Auto is first and the default: the organization's own provider order
 * decides, and it is the only choice that survives a provider going down. On
 * the chip, Auto's mark turns through the vendors it can reach; in the list,
 * the Auto row rains their marks while it is pointed at. Endpoints are listed
 * under their vendor in the organization's order, and one that failed its
 * last check is shown but cannot be chosen.
 */
export function DeskModelPicker({
  options,
  value,
  onChange,
  hasReplies,
  disabled = false,
}: {
  options: readonly AssistantProviderOption[];
  /** Empty means Auto. */
  value: string;
  onChange: (providerId: string) => void;
  /** Switching mid-conversation re-reads it, which the footer says. */
  hasReplies: boolean;
  disabled?: boolean;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [highlighted, setHighlighted] = useState(0);
  const [spin, setSpin] = useState(0);
  const rootRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const close = useCallback(() => setOpen(false), []);
  useOutsideDismiss(rootRef, open, close);

  const selected = options.find((option) => option.id === value) ?? null;

  const rows = useMemo<Row[]>(() => {
    const term = query.trim().toLowerCase();
    const showAuto = term === "" || "auto".includes(term);
    const matched = options
      .map((option, order) => ({ option, order }))
      .filter(
        ({ option }) =>
          term === "" ||
          `${option.model} ${option.name} ${VENDOR_NAMES[option.vendor ?? ""] ?? ""}`
            .toLowerCase()
            .includes(term),
      )
      .map<Row>(({ option, order }) => ({ auto: false, option, order }));
    return showAuto ? [{ auto: true }, ...matched] : matched;
  }, [options, query]);

  useEffect(() => {
    if (!open) {
      return;
    }
    const timer = window.setTimeout(() => inputRef.current?.focus(), 30);
    return () => window.clearTimeout(timer);
  }, [open]);

  const toggle = () => {
    if (!open) {
      setQuery("");
      setHighlighted(0);
    }
    setOpen(!open);
  };

  const pick = (row: Row | undefined) => {
    if (!row || (!row.auto && row.option.unavailable)) {
      return;
    }
    const id = row.auto ? "" : row.option.id;
    if (id !== value) {
      setSpin((count) => count + 1);
    }
    onChange(id);
    setOpen(false);
  };

  const onKeyDown = (event: KeyboardEvent) => {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setHighlighted((index) => Math.min(rows.length - 1, index + 1));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setHighlighted((index) => Math.max(0, index - 1));
    } else if (event.key === "Enter") {
      event.preventDefault();
      pick(rows[highlighted]);
    }
  };

  if (options.length === 0) {
    return null;
  }

  return (
    <div className="dk-mp2" ref={rootRef}>
      <GeminiGradient />
      <button
        type="button"
        className={cn("dk-mp2-b", open && "dk-on", !selected && "dk-auto")}
        disabled={disabled}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={t("Choose which model answers")}
        onClick={toggle}
      >
        {selected ? (
          <span className="dk-mp2-bi" key={spin}>
            <DeskBrandMark vendor={selected.vendor ?? ""} size={13} />
          </span>
        ) : (
          <span className="dk-mp2-bi dk-tr-reel" key={spin}>
            <span className="dk-tr-strip">
              {CHIP_REEL.map((vendor, index) => (
                <span key={index} className={cn("dk-tr-cell", `dk-p-${vendor}`)}>
                  <DeskBrandMark vendor={vendor} size={vendor === "auto" ? 13 : 11} />
                </span>
              ))}
            </span>
          </span>
        )}
        <span className="dk-mp2-bl">
          {selected ? selected.model : <span className="dk-tr-word">{t("Auto")}</span>}
        </span>
        {!selected && <span className="dk-tr-ring" aria-hidden />}
        <svg
          className="dk-mp2-cv"
          width="10"
          height="10"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2.4"
          strokeLinecap="round"
          aria-hidden
        >
          <path d="M6 9l6 6 6-6" />
        </svg>
      </button>
      {open && (
        <div className="dk-mp2-pop" onKeyDown={onKeyDown}>
          <div className="dk-mp2-s">
            <DeskIcon name="search" size={13} />
            <input
              ref={inputRef}
              value={query}
              onChange={(event) => {
                setQuery(event.target.value);
                setHighlighted(0);
              }}
              placeholder={t("Search models…")}
              aria-label={t("Search models")}
            />
          </div>
          <div className="dk-mp2-l" role="listbox">
            {rows.map((row, index) => {
              if (row.auto) {
                return (
                  <button
                    key="auto"
                    type="button"
                    role="option"
                    aria-selected={value === ""}
                    className={cn(
                      "dk-mp2-auto",
                      highlighted === index && "dk-hi",
                      value === "" && "dk-sel",
                    )}
                    onMouseMove={() => highlighted !== index && setHighlighted(index)}
                    onClick={() => pick(row)}
                  >
                    <LogoRain />
                    <span className="dk-mp2-ai">
                      <span className="dk-ai-reel">
                        {REEL_VENDORS.map((vendor) => (
                          <span key={vendor} className={cn("dk-ai-cell", `dk-p-${vendor}`)}>
                            <DeskBrandMark vendor={vendor} size={14} />
                          </span>
                        ))}
                        <span className="dk-ai-cell dk-ai-home">
                          <DeskBrandMark vendor="auto" size={17} />
                        </span>
                      </span>
                      <span className="dk-ai-orbit">
                        {REEL_VENDORS.slice(0, 3).map((vendor, k) => (
                          <span
                            key={vendor}
                            className={cn("dk-ai-sat", `dk-p-${vendor}`)}
                            style={{ "--dk-k": k } as CSSProperties}
                          >
                            <DeskBrandMark vendor={vendor} size={8} />
                          </span>
                        ))}
                      </span>
                    </span>
                    <span className="dk-mp2-tx">
                      <b>
                        <span className="dk-ai-word">{t("Auto")}</span>
                        <em>{t("Recommended")}</em>
                      </b>
                    </span>
                    {value === "" && (
                      <span className="dk-mp2-ck">
                        <DeskIcon name="check" size={13} stroke={2.4} />
                      </span>
                    )}
                  </button>
                );
              }
              const { option, order } = row;
              const vendor = option.vendor ?? "";
              const previous = rows[index - 1];
              const heading =
                !previous || previous.auto || (previous.option.vendor ?? "") !== vendor;
              return (
                <Fragment key={option.id}>
                  {heading && (
                    <div className="dk-mp2-gh">
                      <DeskBrandMark vendor={vendor} size={10} />
                      {VENDOR_NAMES[vendor] ?? t("Self-hosted")}
                    </div>
                  )}
                  <button
                    type="button"
                    role="option"
                    aria-selected={value === option.id}
                    aria-disabled={option.unavailable}
                    className={cn(
                      "dk-mp2-r",
                      highlighted === index && "dk-hi",
                      value === option.id && "dk-sel",
                      option.unavailable && "dk-down",
                    )}
                    style={{ animationDelay: `${Math.min(index, 8) * 22}ms` }}
                    title={
                      option.unavailable ? t("This endpoint failed its last health check") : undefined
                    }
                    onMouseMove={() => highlighted !== index && setHighlighted(index)}
                    onClick={() => pick(row)}
                  >
                    <span className={cn("dk-mp2-ri", `dk-p-${vendor}`)}>
                      <DeskBrandMark vendor={vendor} size={13} />
                    </span>
                    <span className="dk-mp2-tx">
                      <b>{option.model}</b>
                      <span>{option.name}</span>
                    </span>
                    <span className={cn("dk-mp2-tag", option.unavailable && "dk-bad")}>
                      {endpointTag(option, order, t)}
                    </span>
                    {value === option.id && (
                      <span className="dk-mp2-ck">
                        <DeskIcon name="check" size={13} stroke={2.4} />
                      </span>
                    )}
                  </button>
                </Fragment>
              );
            })}
            {rows.length === 0 && (
              <div className="dk-mp2-empty">{t("No models match “{0}”", query)}</div>
            )}
          </div>
          <div className="dk-mp2-f">
            {hasReplies ? (
              <>
                <DeskIcon name="info" size={12} />
                {t("Switching re-reads this conversation before the next reply.")}
              </>
            ) : (
              t("Your organization's admins choose which models are available.")
            )}
          </div>
        </div>
      )}
    </div>
  );
}
