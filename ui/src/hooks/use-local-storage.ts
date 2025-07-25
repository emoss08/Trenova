/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

"use client";

import { useEffect, useState, useCallback } from "react";

function getItemFromLocalStorage(key: string) {
  const item = window?.localStorage.getItem(key);
  if (item) return JSON.parse(item);

  return null;
}

export function useLocalStorage<T>(
  key: string,
  initialValue: T
): [T, React.Dispatch<React.SetStateAction<T>>] {
  const [storedValue, setStoredValue] = useState(initialValue);

  useEffect(() => {
    // initialize
    if (typeof window !== "undefined") {
      const stored = getItemFromLocalStorage(key);
      if (stored !== null) setStoredValue(stored);
    }
  }, [key]);

  const setValue: React.Dispatch<React.SetStateAction<T>> = useCallback(
    (value) => {
      if (value instanceof Function) {
        setStoredValue((prev: T) => {
          const newValue = value(prev);
          // Save to localStorage
          window.localStorage.setItem(key, JSON.stringify(newValue));
          return newValue;
        });
      } else {
        setStoredValue(value);
        // Save to localStorage
        window.localStorage.setItem(key, JSON.stringify(value));
      }
      return setStoredValue;
    },
    [key, setStoredValue]
  );

  return [storedValue, setValue];
}