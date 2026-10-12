import { ChevronRightIcon } from "@trenova/shared/components/icons";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { cn } from "@trenova/shared/lib/utils";
import {
  useEffect,
  useId,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
  type MouseEvent as ReactMouseEvent,
  type PointerEvent as ReactPointerEvent,
} from "react";
import { createPortal } from "react-dom";
import { create } from "zustand";
import { createLongPress } from "./long-press";
import {
  entryKey,
  isActionable,
  isItem,
  normalizeEntries,
  type ContextMenuEntry,
  type ContextMenuItem,
  type ContextMenuSpec,
} from "./menu-spec";
import {
  ITEM_HEIGHT,
  VIRTUALIZE_AFTER,
  entryOffsets,
  placeMenu,
  placeSubmenu,
  scrollToReveal,
  visibleRange,
  type Placement,
  type Point,
  type Rect,
} from "./placement";

type OpenMenu = {
  spec: ContextMenuSpec;
  point: Point;
  /** Where focus goes back to when the menu closes. */
  returnFocus: HTMLElement | null;
  /** A new key per opening, so reopening on another row starts fresh. */
  key: number;
};

const useMenuStore = create<{ open: OpenMenu | null }>(() => ({ open: null }));
let openings = 0;

/**
 * Opens the one context menu at a point on screen, replacing any menu that
 * is open. Mount `ContextMenuHost` once for it to draw.
 */
export function openContextMenu(
  point: Point,
  spec: ContextMenuSpec,
  returnFocus: HTMLElement | null = null,
): void {
  openings += 1;
  useMenuStore.setState({ open: { spec, point, returnFocus, key: openings } });
}

export function closeContextMenu(): void {
  const open = useMenuStore.getState().open;
  if (open === null) {
    return;
  }
  useMenuStore.setState({ open: null });
  if (open.returnFocus !== null && open.returnFocus.isConnected) {
    open.returnFocus.focus({ preventScroll: true });
  }
}

export function useContextMenuOpen(): boolean {
  return useMenuStore((state) => state.open !== null);
}

/** What an element needs to open a menu: right-click, a long press, or Shift+F10 / the menu key. */
export type ContextMenuTriggerProps = {
  onContextMenu: (event: ReactMouseEvent<HTMLElement>) => void;
  onPointerDown: (event: ReactPointerEvent<HTMLElement>) => void;
  onPointerMove: (event: ReactPointerEvent<HTMLElement>) => void;
  onPointerUp: (event: ReactPointerEvent<HTMLElement>) => void;
  onPointerCancel: (event: ReactPointerEvent<HTMLElement>) => void;
  onClickCapture: (event: ReactMouseEvent<HTMLElement>) => void;
  onKeyDown: (event: ReactKeyboardEvent<HTMLElement>) => void;
};

/**
 * Props that open a menu from an element. `build` runs when the menu opens,
 * not when the element renders, so a row in a long list pays nothing for a
 * menu nobody opens, and the menu reads the state of that moment — the
 * selection, the message as it is now. Returning null leaves the browser's
 * own menu alone.
 */
export function useContextMenuTrigger(build: () => ContextMenuSpec | null): ContextMenuTriggerProps {
  const latestBuild = useLatestCallback(build);

  return useMemo<ContextMenuTriggerProps>(() => {
    const tracker = createLongPress<HTMLElement>({
      onLongPress: (point, target) => {
        const spec = latestBuild();
        if (spec !== null) {
          openContextMenu(point, spec, target ?? null);
        }
      },
    });

    return {
      onContextMenu: (event) => {
        if (tracker.justFired()) {
          event.preventDefault();
          return;
        }
        const spec = latestBuild();
        if (spec === null) {
          return;
        }
        event.preventDefault();
        openContextMenu({ x: event.clientX, y: event.clientY }, spec, event.currentTarget);
      },
      onPointerDown: (event) => tracker.down(event, event.currentTarget),
      onPointerMove: (event) => tracker.move(event),
      onPointerUp: () => tracker.up(),
      onPointerCancel: () => tracker.cancel(),
      onClickCapture: (event) => {
        if (tracker.consumeClick()) {
          event.preventDefault();
          event.stopPropagation();
        }
      },
      onKeyDown: (event) => {
        if (!((event.shiftKey && event.key === "F10") || event.key === "ContextMenu")) {
          return;
        }
        const spec = latestBuild();
        if (spec === null) {
          return;
        }
        event.preventDefault();
        const rect = event.currentTarget.getBoundingClientRect();
        openContextMenu({ x: rect.left + 8, y: rect.bottom }, spec, event.currentTarget);
      },
    };
  }, [latestBuild]);
}

function viewport() {
  return { width: window.innerWidth, height: window.innerHeight };
}

function firstActionable(entries: readonly ContextMenuEntry[], from = 0, step: 1 | -1 = 1): number {
  const count = entries.length;
  if (count === 0) {
    return -1;
  }
  for (let offset = 0; offset < count; offset += 1) {
    const index = (((from + offset * step) % count) + count) % count;
    if (isActionable(entries[index])) {
      return index;
    }
  }

  return -1;
}

const TYPEAHEAD_RESET_MS = 600;
const SUBMENU_HOVER_MS = 90;

/**
 * Draws the one context menu every surface opens. Mounted once, anywhere
 * under the app; it renders nothing until a menu is opened.
 *
 * Like every surface that floats from a trigger it is inverted (the `dark`
 * scope resolves the dark tokens in both themes), fully opaque, and lifted
 * with `ui-lift-float`. It closes on a choice, Escape, Tab, a press outside,
 * a scroll behind it, a resize or the window losing focus.
 */
export function ContextMenuHost() {
  const open = useMenuStore((state) => state.open);
  if (open === null || typeof document === "undefined") {
    return null;
  }

  return createPortal(<MenuLayer key={open.key} menu={open} />, document.body);
}

type SubmenuState = { index: number; anchor: Rect };

function MenuLayer({ menu }: { menu: OpenMenu }) {
  const baseId = useId();
  const rootEntries = useMemo(() => normalizeEntries(menu.spec.entries), [menu.spec.entries]);
  const [rootActive, setRootActive] = useState(-1);
  const [submenu, setSubmenu] = useState<SubmenuState | null>(null);
  const [subActive, setSubActive] = useState(-1);
  const [level, setLevel] = useState<0 | 1>(0);
  const layerRef = useRef<HTMLDivElement>(null);
  const rootRef = useRef<HTMLDivElement>(null);
  const subRef = useRef<HTMLDivElement>(null);
  const hoverTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const typeahead = useRef({ text: "", at: 0 });

  const subEntries = useMemo(() => {
    if (submenu === null) {
      return [];
    }
    const owner = rootEntries[submenu.index];
    return owner !== undefined && isItem(owner) ? normalizeEntries(owner.submenu ?? []) : [];
  }, [rootEntries, submenu]);

  const clearHover = () => {
    if (hoverTimer.current !== null) {
      clearTimeout(hoverTimer.current);
      hoverTimer.current = null;
    }
  };

  useEffect(() => {
    rootRef.current?.focus({ preventScroll: true });
  }, []);

  useEffect(() => {
    if (level === 1) {
      subRef.current?.focus({ preventScroll: true });
    } else {
      rootRef.current?.focus({ preventScroll: true });
    }
  }, [level, submenu]);

  useEffect(() => {
    const outside = (event: Event) => {
      const target = event.target;
      if (target instanceof Node && layerRef.current?.contains(target)) {
        return;
      }
      closeContextMenu();
    };
    const close = () => closeContextMenu();
    document.addEventListener("pointerdown", outside, true);
    document.addEventListener("scroll", outside, true);
    window.addEventListener("resize", close);
    window.addEventListener("blur", close);

    return () => {
      clearHover();
      document.removeEventListener("pointerdown", outside, true);
      document.removeEventListener("scroll", outside, true);
      window.removeEventListener("resize", close);
      window.removeEventListener("blur", close);
    };
  }, []);

  const openSubmenuAt = (index: number, focus: boolean) => {
    const element = document.getElementById(`${baseId}-0-${index}`);
    if (element === null) {
      return;
    }
    const rect = element.getBoundingClientRect();
    setSubmenu({
      index,
      anchor: { top: rect.top, left: rect.left, right: rect.right, bottom: rect.bottom },
    });
    const entries = normalizeEntries((rootEntries[index] as ContextMenuItem).submenu ?? []);
    setSubActive(focus ? firstActionable(entries) : -1);
    if (focus) {
      setLevel(1);
    }
  };

  const choose = (entry: ContextMenuEntry, index: number, fromLevel: 0 | 1) => {
    if (!isActionable(entry)) {
      return;
    }
    if (entry.submenu !== undefined) {
      if (fromLevel === 0) {
        openSubmenuAt(index, true);
      }
      return;
    }
    closeContextMenu();
    entry.onSelect?.();
  };

  const onKeyDown = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    const entries = level === 0 ? rootEntries : subEntries;
    const active = level === 0 ? rootActive : subActive;
    const setActive = level === 0 ? setRootActive : setSubActive;

    switch (event.key) {
      case "ArrowDown":
        event.preventDefault();
        setActive(firstActionable(entries, active + 1, 1));
        return;
      case "ArrowUp":
        event.preventDefault();
        setActive(firstActionable(entries, active === -1 ? entries.length - 1 : active - 1, -1));
        return;
      case "Home":
        event.preventDefault();
        setActive(firstActionable(entries, 0, 1));
        return;
      case "End":
        event.preventDefault();
        setActive(firstActionable(entries, entries.length - 1, -1));
        return;
      case "ArrowRight":
        if (level === 0 && active !== -1) {
          const entry = entries[active];
          if (isActionable(entry) && entry.submenu !== undefined) {
            event.preventDefault();
            openSubmenuAt(active, true);
          }
        }
        return;
      case "ArrowLeft":
        if (level === 1) {
          event.preventDefault();
          setLevel(0);
          setSubmenu(null);
        }
        return;
      case "Escape":
        event.preventDefault();
        event.stopPropagation();
        if (level === 1) {
          setLevel(0);
          setSubmenu(null);
        } else {
          closeContextMenu();
        }
        return;
      case "Tab":
        event.preventDefault();
        closeContextMenu();
        return;
      case "Enter":
      case " ":
        event.preventDefault();
        if (active !== -1) {
          choose(entries[active], active, level);
        }
        return;
    }

    if (event.key.length === 1 && !event.metaKey && !event.ctrlKey && !event.altKey) {
      const now = Date.now();
      const text =
        now - typeahead.current.at > TYPEAHEAD_RESET_MS
          ? event.key.toLowerCase()
          : typeahead.current.text + event.key.toLowerCase();
      typeahead.current = { text, at: now };
      const count = entries.length;
      const start = text.length > 1 ? Math.max(active, 0) : active + 1;
      for (let offset = 0; offset < count; offset += 1) {
        const index = (start + offset + count) % count;
        const entry = entries[index];
        if (isActionable(entry) && entry.label.toLowerCase().startsWith(text)) {
          setActive(index);
          break;
        }
      }
    }
  };

  const onRootHover = (index: number) => {
    setRootActive(index);
    if (level === 1) {
      setLevel(0);
    }
    clearHover();
    const entry = rootEntries[index];
    hoverTimer.current = setTimeout(() => {
      if (isActionable(entry) && entry.submenu !== undefined) {
        if (submenu?.index !== index) {
          openSubmenuAt(index, false);
        }
      } else {
        setSubmenu(null);
      }
    }, SUBMENU_HOVER_MS);
  };

  const label = menu.spec.title ?? menu.spec.label;

  return (
    <div ref={layerRef} className="dark" onKeyDown={onKeyDown}>
      <MenuSurface
        refObject={rootRef}
        id={`${baseId}-0`}
        level={0}
        label={label}
        title={menu.spec.title}
        subtitle={menu.spec.subtitle}
        entries={rootEntries}
        active={rootActive}
        expanded={submenu?.index ?? -1}
        place={(size) => placeMenu(menu.point, size, viewport())}
        onHover={onRootHover}
        onChoose={(entry, index) => choose(entry, index, 0)}
      />
      {submenu !== null && subEntries.length > 0 && (
        <MenuSurface
          key={submenu.index}
          refObject={subRef}
          id={`${baseId}-1`}
          level={1}
          label={(rootEntries[submenu.index] as ContextMenuItem).label}
          entries={subEntries}
          active={subActive}
          expanded={-1}
          place={(size) => placeSubmenu(submenu.anchor, size, viewport())}
          onEnter={clearHover}
          onHover={(index) => {
            clearHover();
            setSubActive(index);
            setLevel(1);
          }}
          onChoose={(entry, index) => choose(entry, index, 1)}
        />
      )}
    </div>
  );
}

type MenuSurfaceProps = {
  refObject: React.RefObject<HTMLDivElement | null>;
  id: string;
  level: 0 | 1;
  label?: string;
  title?: string;
  subtitle?: string;
  entries: readonly ContextMenuEntry[];
  active: number;
  /** The row whose submenu is open. */
  expanded: number;
  place: (size: { width: number; height: number }) => Placement;
  onHover: (index: number) => void;
  onEnter?: () => void;
  onChoose: (entry: ContextMenuEntry, index: number) => void;
};

function MenuSurface({
  refObject,
  id,
  level,
  label,
  title,
  subtitle,
  entries,
  active,
  expanded,
  place,
  onHover,
  onEnter,
  onChoose,
}: MenuSurfaceProps) {
  const [placement, setPlacement] = useState<Placement | null>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const [scrollTop, setScrollTop] = useState(0);
  const virtual = entries.length > VIRTUALIZE_AFTER;
  const offsets = useMemo(() => entryOffsets(entries), [entries]);
  const placeLatest = useLatestCallback(place);

  useLayoutEffect(() => {
    const element = refObject.current;
    if (element === null) {
      return;
    }
    const rect = element.getBoundingClientRect();
    const list = listRef.current;
    const chrome = list === null ? 0 : rect.height - list.clientHeight;
    const contentHeight = virtual ? chrome + offsets[offsets.length - 1] : element.scrollHeight;
    setPlacement(placeLatest({ width: rect.width, height: contentHeight }));
  }, [refObject, virtual, offsets, placeLatest]);

  useLayoutEffect(() => {
    const list = listRef.current;
    if (list === null || active === -1) {
      return;
    }
    if (virtual) {
      const next = scrollToReveal(offsets, active, list.scrollTop, list.clientHeight);
      if (next !== list.scrollTop) {
        list.scrollTop = next;
        setScrollTop(next);
      }
      return;
    }
    document.getElementById(`${id}-${active}`)?.scrollIntoView({ block: "nearest" });
  }, [active, id, offsets, virtual]);

  const listHeight =
    placement === null
      ? undefined
      : Math.max(ITEM_HEIGHT, placement.maxHeight - (title !== undefined ? 48 : 8));
  const range = virtual
    ? visibleRange(offsets, scrollTop, listHeight ?? offsets[offsets.length - 1])
    : { start: 0, end: entries.length };

  const renderEntry = (entry: ContextMenuEntry, index: number) => {
    if (entry.type === "separator") {
      return <div key={entryKey(entry, index)} role="separator" className="bg-border-subtle mx-1 my-1 h-px" />;
    }
    if (entry.type === "header") {
      return (
        <div
          key={entryKey(entry, index)}
          role="presentation"
          className="text-foreground-subtle flex h-6.5 items-end px-2 pb-1 text-xs font-medium"
        >
          {entry.label}
        </div>
      );
    }

    return (
      <MenuRow
        key={entryKey(entry, index)}
        id={`${id}-${index}`}
        item={entry}
        active={index === active}
        expanded={index === expanded}
        onHover={() => onHover(index)}
        onChoose={() => onChoose(entry, index)}
      />
    );
  };

  return (
    <div
      ref={refObject}
      id={id}
      role="menu"
      tabIndex={-1}
      aria-label={label}
      aria-activedescendant={active === -1 ? undefined : `${id}-${active}`}
      data-level={level}
      onPointerEnter={onEnter}
      onContextMenu={(event) => event.preventDefault()}
      style={{
        left: placement?.x ?? 0,
        top: placement?.y ?? 0,
        visibility: placement === null ? "hidden" : "visible",
      }}
      className={cn(
        "bg-popover text-popover-foreground ui-lift-float animate-pop-in fixed z-[60] flex max-w-80 min-w-52 flex-col rounded-lg p-1 text-sm ring-1 ring-foreground/10 outline-none select-none",
      )}
    >
      {title !== undefined && (
        <div className="border-border-subtle mb-1 flex min-w-0 flex-col gap-0.5 border-b px-2 pt-1 pb-2">
          <span className="truncate font-medium">{title}</span>
          {subtitle !== undefined && subtitle !== "" && (
            <span className="text-foreground-subtle truncate text-xs">{subtitle}</span>
          )}
        </div>
      )}
      <div
        ref={listRef}
        onScroll={virtual ? (event) => setScrollTop(event.currentTarget.scrollTop) : undefined}
        className="scrollbar-overlay min-h-0 overflow-y-auto overscroll-contain"
        style={{ maxHeight: listHeight }}
      >
        {virtual ? (
          <div className="relative" style={{ height: offsets[offsets.length - 1] }}>
            {entries.slice(range.start, range.end).map((entry, offset) => {
              const index = range.start + offset;
              return (
                <div
                  key={entryKey(entry, index)}
                  className="absolute inset-x-0"
                  style={{ transform: `translateY(${offsets[index]}px)` }}
                >
                  {renderEntry(entry, index)}
                </div>
              );
            })}
          </div>
        ) : (
          entries.map(renderEntry)
        )}
      </div>
    </div>
  );
}

function MenuRow({
  id,
  item,
  active,
  expanded,
  onHover,
  onChoose,
}: {
  id: string;
  item: ContextMenuItem;
  active: boolean;
  expanded: boolean;
  onHover: () => void;
  onChoose: () => void;
}) {
  const Icon = item.icon;
  const disabled = item.disabled === true;

  return (
    <div
      id={id}
      role="menuitem"
      aria-disabled={disabled || undefined}
      aria-haspopup={item.submenu !== undefined ? "menu" : undefined}
      aria-expanded={item.submenu !== undefined ? expanded : undefined}
      data-active={active || expanded || undefined}
      data-variant={item.destructive === true ? "destructive" : undefined}
      onPointerMove={disabled ? undefined : onHover}
      onClick={(event) => {
        event.stopPropagation();
        onChoose();
      }}
      className={cn(
        "flex h-7 cursor-default items-center gap-2 rounded-md px-2",
        "data-active:bg-accent data-active:text-accent-foreground",
        "data-[variant=destructive]:text-destructive data-[variant=destructive]:data-active:bg-danger-subtle",
        disabled && "opacity-50",
      )}
    >
      <span
        aria-hidden
        className="text-foreground-muted flex size-4 shrink-0 items-center justify-center [&_svg]:size-3.5"
      >
        {item.leading ?? (Icon === undefined ? null : <Icon />)}
      </span>
      <span className="flex min-w-0 flex-1 items-baseline gap-2">
        <span className="truncate">{item.label}</span>
        {item.hint !== undefined && item.hint !== "" && (
          <span className="text-foreground-subtle truncate text-xs">{item.hint}</span>
        )}
      </span>
      {item.shortcut !== undefined && <Kbd className="ml-2">{item.shortcut}</Kbd>}
      {item.submenu !== undefined && (
        <ChevronRightIcon aria-hidden className="text-foreground-subtle size-3.5 shrink-0" />
      )}
    </div>
  );
}

/** Whether a context menu is open right now, for a key handler deciding whether a key is its own. */
export function isContextMenuOpen(): boolean {
  return useMenuStore.getState().open !== null;
}
