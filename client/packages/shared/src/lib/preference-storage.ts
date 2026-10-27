// preference-storage.ts is localStorage for what a page can do without, such as how a person
// sized its panels: a read or write the browser refuses (a private window, a full or blocked
// store) fails quietly and the page carries on with its defaults. Nothing about a record or a
// person goes here; it holds how the interface was arranged.

/** A `Storage`-shaped view of localStorage whose reads and writes never throw. */
export const preferenceStorage: Pick<Storage, "getItem" | "setItem"> = {
  getItem(key) {
    try {
      return typeof window === "undefined" ? null : window.localStorage.getItem(key);
    } catch {
      return null;
    }
  },
  setItem(key, value) {
    try {
      if (typeof window !== "undefined") window.localStorage.setItem(key, value);
    } catch {
      // Storage may be unavailable or full; the layout still works, it is just not remembered.
    }
  },
};
