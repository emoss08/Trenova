import { parseAsString, useQueryState } from "nuqs";

/** The query parameter a settings page keeps its open tab in, so a tab can be linked to. */
export const SETTINGS_TAB_PARAM = "tab";

/** What a tab is, apart from what it shows: enough for the page's loading state to draw it. */
export type SettingsTabSpec<TField extends string = string> = {
  value: string;
  label: string;
  /**
   * The form's top-level fields this tab holds. An error on one marks the tab, and a save
   * that is refused opens the tab that holds the problem.
   */
  fields: readonly TField[];
};

/**
 * The open tab of a settings page, kept in the address. A value the page does not offer
 * (a tab the organization does not have) opens the first tab, which keeps a clean address.
 */
export function useSettingsTab(
  tabs: readonly Pick<SettingsTabSpec, "value">[],
): readonly [string, (value: string) => void] {
  const [param, setParam] = useQueryState(SETTINGS_TAB_PARAM, parseAsString);
  const first = tabs[0]?.value ?? "";
  const open = tabs.some((tab) => tab.value === param) ? (param as string) : first;
  const select = (value: string) => void setParam(value === first ? null : value);

  return [open, select] as const;
}

/**
 * The tab to show after a save was refused: the open one while it holds a problem, since
 * the person is already looking at it, and otherwise the first that does. Null when no tab
 * holds one, where the form's own alert says what went wrong.
 */
export function tabHoldingError<TField extends string>(
  tabs: readonly SettingsTabSpec<TField>[],
  open: string,
  hasError: (field: TField) => boolean,
): string | null {
  const holds = (tab: SettingsTabSpec<TField>) => tab.fields.some(hasError);
  const current = tabs.find((tab) => tab.value === open);
  if (current && holds(current)) return current.value;

  return tabs.find(holds)?.value ?? null;
}
