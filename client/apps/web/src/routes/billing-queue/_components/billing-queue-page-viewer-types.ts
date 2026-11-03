/** A run of pages the viewer labels as one document, such as a packet's proof of delivery. */
export type ViewerPart = {
  key: string;
  name: string;
  firstPage: number;
  lastPage: number;
  /** Whether it can be taken out of the file, and moved among the other movable parts. */
  removable: boolean;
  movable: boolean;
};

/** Something on a page a person should look at before the file goes out. */
export type PageFlag = {
  title: string;
  detail: string;
  /** A short visible label, with the label a screen reader hears when it needs more. */
  action?: {
    label: string;
    ariaLabel?: string;
    /** Settles when the action is done; the button holds until it does. */
    onAction: () => Promise<void> | void;
    disabled?: boolean;
  };
};

/** How the parts of the file are changed from the rail, when they can be. */
export type PartEditing = {
  onRemove: (key: string) => Promise<void>;
  /** Moves the part to `index` among all parts (0 is the first), shifting the others over. */
  onMove: (key: string, index: number) => void;
  /** Turns a page a quarter turn clockwise in the file itself, when it can be turned. */
  onRotate: (page: number) => void;
  rotatable: (page: number) => boolean;
  /** A change is saving: the rail holds still until the file comes back. */
  busy: boolean;
};

/** The page range of a part as the rail and the column print it. */
export function partRange(part: Pick<ViewerPart, "firstPage" | "lastPage">) {
  return part.firstPage === part.lastPage
    ? String(part.firstPage)
    : `${part.firstPage}–${part.lastPage}`;
}

/** The page after `page` that carries a flag, wrapping to the first; null when none does. */
export function nextFlaggedPage(flags: ReadonlyMap<number, PageFlag>, page: number) {
  const pages = [...flags.keys()].sort((a, b) => a - b);
  return pages.find((candidate) => candidate > page) ?? pages[0] ?? null;
}
