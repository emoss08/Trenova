import { useCallback, useState, type Dispatch, type SetStateAction } from "react";

function getItemFromLocalStorage<T>(key: string, initialValue: T): T {
  if (typeof window === "undefined") {
    return initialValue;
  }

  try {
    const item = window.localStorage.getItem(key);
    if (item === null) {
      return initialValue;
    }

    return JSON.parse(item) as T;
  } catch {
    try {
      window.localStorage.removeItem(key);
    } catch {
      // Ignore storage cleanup failures and keep rendering with the fallback.
    }

    return initialValue;
  }
}

function setItemInLocalStorage<T>(key: string, value: T) {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // Storage may be unavailable or full; keep the in-memory state update.
  }
}

export function useLocalStorage<T>(
  key: string,
  initialValue: T,
): [T, Dispatch<SetStateAction<T>>] {
  const [storedValue, setStoredValue] = useState<T>(() =>
    getItemFromLocalStorage(key, initialValue),
  );

  const setValue: Dispatch<SetStateAction<T>> = useCallback(
    (value) => {
      if (value instanceof Function) {
        setStoredValue((prev: T) => {
          const newValue = value(prev);
          setItemInLocalStorage(key, newValue);
          return newValue;
        });
      } else {
        setStoredValue(value);
        setItemInLocalStorage(key, value);
      }
    },
    [key],
  );

  return [storedValue, setValue];
}
