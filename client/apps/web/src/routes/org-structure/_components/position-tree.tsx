import { useT } from "@trenova/shared/i18n/use-t";
import type { HeadcountRow, JobPositionRow } from "@/lib/graphql/org-structure";
import {
  ancestorIds,
  buildPositionTree,
  canReportTo,
  flattenTree,
  matchesPositionSearch,
  pruneTree,
  type PositionNode,
} from "@/lib/org-chart";
import {
  DndContext,
  PointerSensor,
  useDraggable,
  useDroppable,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from "@dnd-kit/core";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { Input } from "@trenova/shared/components/ui/input";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { jobDepartmentLabel } from "@trenova/shared/lib/org-structure";
import { cn } from "@trenova/shared/lib/utils";
import {
  ChevronDownIcon,
  ChevronRightIcon,
  ChevronsDownUpIcon,
  ChevronsUpDownIcon,
  CornerDownRightIcon,
  GripVerticalIcon,
  MoreHorizontalIcon,
  PencilLineIcon,
  PlusIcon,
  SearchIcon,
  UsersIcon,
} from "lucide-react";
import { useCallback, useMemo, useState } from "react";

const INDENT_REM = 1.5;
const DEFAULT_OPEN_DEPTH = 1;
const TOP_LEVEL = "top-level";

type Tree = ReturnType<typeof buildPositionTree<JobPositionRow>>;

type PositionTreeProps = {
  positions: readonly JobPositionRow[];
  byPosition: readonly HeadcountRow[];
  canCreate: boolean;
  canUpdate: boolean;
  /** Which position a move is in flight for, so only that row shows busy. */
  moving: string | null;
  onAdd: (reportsToPositionId: string | null) => void;
  onEdit: (position: JobPositionRow) => void;
  onMove: (position: JobPositionRow, reportsToPositionId: string | null) => void;
  onOpenHolders: (position: JobPositionRow) => void;
};

/**
 * The org chart as a tree a person can work on, not just read. Branches fold
 * away so a deep chart fits on a screen; a position is hung somewhere else by
 * dragging it onto its new parent or, without a mouse, from its row menu; and
 * a new position is added straight under the one it will report to.
 */
export function PositionTree({
  positions,
  byPosition,
  canCreate,
  canUpdate,
  moving,
  onAdd,
  onEdit,
  onMove,
  onOpenHolders,
}: PositionTreeProps) {
  const t = useT();

  const [query, setQuery] = useState("");
  const tree = useMemo(() => buildPositionTree(positions, byPosition), [positions, byPosition]);
  const all = useMemo(() => flattenTree(tree.roots), [tree.roots]);
  const [collapsed, setCollapsed] = useState<Set<string>>(
    () =>
      new Set(
        all
          .filter((node) => node.depth >= DEFAULT_OPEN_DEPTH && node.children.length > 0)
          .map((node) => node.position.id),
      ),
  );

  // A search opens every branch on the way to a match; nothing is worse than
  // a hit the reader cannot see because its parent is folded.
  const searching = query.trim().length > 0;
  const forcedOpen = useMemo(() => {
    if (!searching) return new Set<string>();
    const open = new Set<string>();
    for (const position of positions) {
      if (matchesPositionSearch(position, query)) {
        for (const id of ancestorIds(positions, position.id)) open.add(id);
      }
    }
    return open;
  }, [positions, query, searching]);

  const isOpen = useCallback(
    (id: string) => forcedOpen.has(id) || !collapsed.has(id),
    [collapsed, forcedOpen],
  );

  const rows = useMemo(() => {
    const visible: PositionNode<JobPositionRow>[] = [];
    const walk = (node: PositionNode<JobPositionRow>) => {
      visible.push(node);
      if (node.children.length > 0 && isOpen(node.position.id)) {
        for (const child of node.children) walk(child);
      }
    };
    for (const root of pruneTree(tree.roots, query)) walk(root);
    return visible;
  }, [tree.roots, query, isOpen]);

  const toggle = (id: string) =>
    setCollapsed((current) => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  const expandAll = () => setCollapsed(new Set());
  const collapseAll = () =>
    setCollapsed(
      new Set(all.filter((node) => node.children.length > 0).map((node) => node.position.id)),
    );

  // The position on the move is held in state rather than read off the drag
  // event during render: every row needs it to decide whether it is a drop
  // target, and a ref read in render is invisible to React.
  const [dragging, setDragging] = useState<JobPositionRow | null>(null);
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }));
  const handleDragStart = (event: DragStartEvent) => {
    setDragging((event.active.data.current?.position as JobPositionRow | undefined) ?? null);
  };
  const handleDragEnd = (event: DragEndEvent) => {
    setDragging(null);
    const dragged = event.active.data.current?.position as JobPositionRow | undefined;
    const over = event.over?.id;
    if (!dragged || over === undefined) return;
    const parentId = over === TOP_LEVEL ? null : String(over).replace(/^node:/, "");
    if (parentId === (dragged.reportsToPositionId ?? null)) return;
    if (!canReportTo(tree.roots, dragged.id, parentId)) return;
    onMove(dragged, parentId);
  };

  return (
    <section aria-labelledby="org-chart-heading" className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <h3 id="org-chart-heading" className="sr-only">
            {t("Org chart")}
          </h3>
          <Input
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={t("Find a title, code or department")}
            aria-label={t("Find a position")}
            leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
            inputContainerClassName="w-72 max-w-full"
          />
          <Button size="xs" variant="ghost" onClick={expandAll} aria-label={t("Expand all")}>
            <ChevronsUpDownIcon className="size-3.5" />
            {t("Expand")}
          </Button>
          <Button size="xs" variant="ghost" onClick={collapseAll} aria-label={t("Collapse all")}>
            <ChevronsDownUpIcon className="size-3.5" />
            {t("Collapse")}
          </Button>
          {tree.unplaced > 0 ? (
            <span className="text-muted-foreground text-xs tabular-nums">
              {t("{0} {1} with no position", tree.unplaced, tree.unplaced === 1 ? "person" : "people")}
            </span>
          ) : null}
        </div>
        {canCreate ? (
          <Button size="sm" onClick={() => onAdd(null)}>
            <PlusIcon className="size-3.5" />
            {t("Add a position")}
          </Button>
        ) : null}
      </div>

      {rows.length === 0 ? (
        <div className="text-muted-foreground flex flex-col items-center gap-2 rounded-lg border border-dashed px-4 py-8 text-center text-sm">
          <p>{t("No position matches that.")}</p>
          <Button size="xs" variant="ghost" onClick={() => setQuery("")}>
            {t("Clear search")}
          </Button>
        </div>
      ) : (
        <DndContext
          sensors={sensors}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
          onDragCancel={() => setDragging(null)}
        >
          <div className="bg-card overflow-hidden rounded-lg border">
            <ul aria-label={t("Positions")}>
              {rows.map((node) => (
                <TreeRow
                  key={node.position.id}
                  node={node}
                  tree={tree}
                  dragging={dragging}
                  open={isOpen(node.position.id)}
                  canCreate={canCreate}
                  canUpdate={canUpdate}
                  busy={moving === node.position.id}
                  onToggle={() => toggle(node.position.id)}
                  onAdd={() => onAdd(node.position.id)}
                  onEdit={() => onEdit(node.position)}
                  onMove={(parentId) => onMove(node.position, parentId)}
                  onOpenHolders={() => onOpenHolders(node.position)}
                />
              ))}
            </ul>
            {canUpdate ? <TopLevelDropZone visible={dragging !== null} /> : null}
          </div>
        </DndContext>
      )}
      {canUpdate ? (
        <p className="text-muted-foreground text-xs">
          {t("Drag a position onto the one it should report to, or use its menu. Drop it on the bar at the bottom to make it top level.")}
        </p>
      ) : null}
    </section>
  );
}

type TreeRowProps = {
  node: PositionNode<JobPositionRow>;
  tree: Tree;
  /** The position being dragged, if any; decides whether this row can take it. */
  dragging: JobPositionRow | null;
  open: boolean;
  canCreate: boolean;
  canUpdate: boolean;
  busy: boolean;
  onToggle: () => void;
  onAdd: () => void;
  onEdit: () => void;
  onMove: (reportsToPositionId: string | null) => void;
  onOpenHolders: () => void;
};

function TreeRow({
  node,
  tree,
  dragging,
  open,
  canCreate,
  canUpdate,
  busy,
  onToggle,
  onAdd,
  onEdit,
  onMove,
  onOpenHolders,
}: TreeRowProps) {
  const t = useT();

  const { position } = node;
  const archived = position.status !== "Active";
  const hasChildren = node.children.length > 0;

  const {
    attributes: dragAttributes,
    listeners: dragListeners,
    setNodeRef: setDragRef,
    isDragging,
  } = useDraggable({
    id: `drag:${position.id}`,
    disabled: !canUpdate || busy,
    data: { position },
  });
  const acceptable = dragging
    ? dragging.id !== position.id && canReportTo(tree.roots, dragging.id, position.id)
    : false;
  const { setNodeRef: setDropRef, isOver } = useDroppable({
    id: `node:${position.id}`,
    disabled: !canUpdate || !acceptable,
    data: { position },
  });

  // Parents the row could be moved under, for the keyboard route. Its own
  // branch is left out for the same reason a drop onto it is refused.
  const moveTargets = useMemo(
    () =>
      flattenTree(tree.roots).filter(
        (candidate) =>
          candidate.position.id !== position.id &&
          candidate.position.id !== (position.reportsToPositionId ?? null) &&
          canReportTo(tree.roots, position.id, candidate.position.id),
      ),
    [tree.roots, position],
  );

  return (
    <li
      ref={setDropRef}
      data-depth={node.depth}
      data-open={hasChildren ? open : undefined}
      aria-label={t(position.title)}
      className={cn(
        "group/row relative grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 border-b py-1.5 pr-2 transition-colors last:border-b-0",
        isDragging && "opacity-40",
        isOver && acceptable && "bg-accent",
        archived && "opacity-70",
      )}
      style={{ paddingLeft: `${0.5 + node.depth * INDENT_REM}rem` }}
    >
      {node.depth > 0 ? (
        <span
          aria-hidden
          className="bg-border absolute top-0 bottom-0 w-px"
          style={{ left: `${0.5 + (node.depth - 1) * INDENT_REM + 0.6}rem` }}
        />
      ) : null}
      <div className="flex min-w-0 items-center gap-1.5">
        {hasChildren ? (
          <button
            type="button"
            onClick={onToggle}
            aria-expanded={open}
            aria-label={`${open ? "Collapse" : "Expand"} ${position.title}`}
            className="text-muted-foreground hover:text-foreground grid size-5 shrink-0 place-items-center rounded-md"
          >
            {open ? (
              <ChevronDownIcon className="size-3.5" />
            ) : (
              <ChevronRightIcon className="size-3.5" />
            )}
          </button>
        ) : (
          <span aria-hidden className="grid size-5 shrink-0 place-items-center">
            {node.depth > 0 ? (
              <CornerDownRightIcon className="text-muted-foreground/60 size-3" />
            ) : null}
          </span>
        )}
        {canUpdate ? (
          <button
            type="button"
            ref={setDragRef}
            {...dragAttributes}
            {...dragListeners}
            aria-label={`Drag ${position.title}`}
            disabled={busy}
            className="text-muted-foreground/50 hover:text-foreground grid size-5 shrink-0 cursor-grab place-items-center rounded-md opacity-0 transition-opacity group-hover/row:opacity-100 focus-visible:opacity-100 active:cursor-grabbing disabled:cursor-default"
          >
            <GripVerticalIcon className="size-3.5" />
          </button>
        ) : null}
        <div className="flex min-w-0 flex-col leading-tight">
          <span className="flex min-w-0 flex-wrap items-center gap-2">
            <span className="truncate text-sm font-medium">{t(position.title)}</span>
            <span className="text-muted-foreground text-xs tabular-nums">{position.code}</span>
            {position.isDrivingPosition ? <Badge variant="info">{t("Driving")}</Badge> : null}
            {position.flsaExempt ? <Badge variant="secondary">{t("Exempt")}</Badge> : null}
            {archived ? <Badge variant="inactive">{t("Archived")}</Badge> : null}
          </span>
          <span className="text-muted-foreground text-xs">
            {jobDepartmentLabel(position.department)}
            {hasChildren
              ? t("· {0} reporting position{1}", node.children.length, node.children.length === 1 ? "" : "s")
              : ""}
            {node.terminated > 0 ? t("· {0} left", node.terminated) : ""}
          </span>
        </div>
      </div>

      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={onOpenHolders}
          className="hover:bg-accent rounded-md px-1.5 py-0.5 text-right text-xs tabular-nums transition-colors"
          aria-label={`${position.title} headcount`}
          title={t("Who holds it")}
        >
          <span
            className={cn("font-mono font-medium", node.people === 0 && "text-muted-foreground")}
          >
            {node.people}
          </span>
          {hasChildren ? (
            <span className="text-muted-foreground"> {t("· {0} below", node.rolledUp - node.people)}</span>
          ) : null}
        </button>
        {canCreate ? (
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  size="icon-xs"
                  variant="ghost"
                  onClick={onAdd}
                  aria-label={`Add a position under ${position.title}`}
                  className="opacity-0 transition-opacity group-hover/row:opacity-100 focus-visible:opacity-100"
                />
              }
            >
              <PlusIcon className="size-3.5" />
            </TooltipTrigger>
            <TooltipContent>{t("Add a position reporting to this one")}</TooltipContent>
          </Tooltip>
        ) : null}
        {canUpdate ? (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  size="icon-xs"
                  variant="ghost"
                  isLoading={busy}
                  aria-label={`More for ${position.title}`}
                />
              }
            >
              <MoreHorizontalIcon className="size-3.5" />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-64">
              <DropdownMenuItem
                title={t("Who holds it")}
                startContent={<UsersIcon className="size-3.5" />}
                onClick={onOpenHolders}
              />
              <DropdownMenuItem
                title={t("Edit")}
                startContent={<PencilLineIcon className="size-3.5" />}
                onClick={onEdit}
              />
              <DropdownMenuSeparator />
              <DropdownMenuGroup>
                <DropdownMenuLabel>{t("Move under")}</DropdownMenuLabel>
                {position.reportsToPositionId ? (
                  <DropdownMenuItem title={t("Nothing (top level)")} onClick={() => onMove(null)} />
                ) : null}
                {moveTargets.length === 0 && !position.reportsToPositionId ? (
                  <DropdownMenuItem title={t("Nowhere else to put it")} disabled />
                ) : null}
                {moveTargets.map((target) => (
                  <DropdownMenuItem
                    key={target.position.id}
                    title={t(target.position.title)}
                    titleClassProps="truncate"
                    onClick={() => onMove(target.position.id)}
                    style={{ paddingLeft: `${0.5 + target.depth * 0.75}rem` }}
                  />
                ))}
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : null}
      </div>
    </li>
  );
}

function TopLevelDropZone({ visible }: { visible: boolean }) {
  const t = useT();

  const { setNodeRef, isOver } = useDroppable({ id: TOP_LEVEL });
  return (
    <div
      ref={setNodeRef}
      aria-label={t("Make top level")}
      className={cn(
        "text-muted-foreground border-t px-3 py-1.5 text-center text-xs transition-colors",
        visible ? "border-dashed" : "hidden",
        isOver && "bg-accent text-foreground",
      )}
    >
      {t("Drop here to make it top level")}
    </div>
  );
}
