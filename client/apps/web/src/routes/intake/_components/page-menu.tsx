import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  ArrowRightToLineIcon,
  EyeIcon,
  FilePlus02Icon,
  FlipBackwardIcon,
  RefreshCcw01Icon,
  RefreshCw01Icon,
  Scissors01Icon,
} from "@trenova/shared/components/icons";
import { LOOSE, type LayoutGroup } from "./page-layout";
import type { PageMenu as PageMenuHandle, PageMenuPayload } from "./page-thumbnail";

export type PageMenuActions = {
  preview: (pageId: string) => void;
  rotate: (pageId: string, quarterTurns: number) => void;
  /** Starts a new document after this page. */
  splitAfter: (groupKey: string, pageId: string) => void;
  /** Takes the page out of its document. */
  leaveOut: (pageId: string) => void;
  /** Makes a document of one page set aside. */
  newDocument: (pageId: string) => void;
  moveTo: (pageId: string, target: string) => void;
};

/**
 * The menu of whichever page's button was pressed. A stack shares one, so a
 * thousand pages mount one menu, and where a page can move is worked out when
 * the menu opens rather than for every page on every change.
 */
export function PageMenu({
  handle,
  groups,
  canEdit,
  actions,
}: {
  handle: PageMenuHandle;
  groups: readonly LayoutGroup[];
  canEdit: boolean;
  actions: PageMenuActions;
}) {
  return (
    <DropdownMenu<PageMenuPayload> handle={handle}>
      {({ payload }) =>
        payload === undefined ? null : (
          <PageMenuContent payload={payload} groups={groups} canEdit={canEdit} actions={actions} />
        )
      }
    </DropdownMenu>
  );
}

function PageMenuContent({
  payload,
  groups,
  canEdit,
  actions,
}: {
  payload: PageMenuPayload;
  groups: readonly LayoutGroup[];
  canEdit: boolean;
  actions: PageMenuActions;
}) {
  const t = useT();
  const { pageId, groupKey, last } = payload;
  const loose = groupKey === LOOSE;
  const targets = groups.flatMap((group, index) =>
    group.key === groupKey ? [] : [{ key: group.key, label: t("Document {0}", index + 1) }],
  );

  return (
    <DropdownMenuContent align="end" className="w-60">
      <DropdownMenuItem
        title={t("Preview")}
        startContent={<EyeIcon className="size-3.5" />}
        onClick={() => actions.preview(pageId)}
      />
      {canEdit && (
        <>
          <DropdownMenuItem
            title={t("Rotate right")}
            startContent={<RefreshCw01Icon className="size-3.5" />}
            onClick={() => actions.rotate(pageId, 1)}
          />
          <DropdownMenuItem
            title={t("Rotate left")}
            startContent={<RefreshCcw01Icon className="size-3.5" />}
            onClick={() => actions.rotate(pageId, -1)}
          />
          <DropdownMenuSeparator />
          {!loose && !last && (
            <DropdownMenuItem
              title={t("Start a new document after this page")}
              startContent={<Scissors01Icon className="size-3.5" />}
              onClick={() => actions.splitAfter(groupKey, pageId)}
            />
          )}
          {loose && (
            <DropdownMenuItem
              title={t("Make it a document of its own")}
              startContent={<FilePlus02Icon className="size-3.5" />}
              onClick={() => actions.newDocument(pageId)}
            />
          )}
          {targets.length > 0 && (
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                <ArrowRightToLineIcon className="size-3.5" />
                {t("Move to")}
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent className="max-h-72 w-52">
                {targets.map((target) => (
                  <DropdownMenuItem
                    key={target.key}
                    title={target.label}
                    onClick={() => actions.moveTo(pageId, target.key)}
                  />
                ))}
              </DropdownMenuSubContent>
            </DropdownMenuSub>
          )}
          {!loose && (
            <DropdownMenuItem
              title={t("Set aside")}
              description={t("Keeps the page out of every document")}
              startContent={<FlipBackwardIcon className="size-3.5" />}
              onClick={() => actions.leaveOut(pageId)}
            />
          )}
        </>
      )}
    </DropdownMenuContent>
  );
}
