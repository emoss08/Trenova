/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { useCallback, useEffect } from "react";
import { useLocation, useNavigate } from "react-router";
import { create } from "zustand";

interface UrlFragmentStore {
  fragment: string;
  setFragment: (fragment: string) => void;
  clearFragment: () => void;
}

const useUrlFragmentStore = create<UrlFragmentStore>((set) => ({
  fragment: "",
  setFragment: (fragment) => set({ fragment }),
  clearFragment: () => set({ fragment: "" }),
}));

export function useUrlFragment() {
  const location = useLocation();
  const navigate = useNavigate();
  const {
    fragment,
    setFragment: setStoreFragment,
    clearFragment: clearStoreFragment,
  } = useUrlFragmentStore();

  // Sync store with URL on mount and location changes
  useEffect(() => {
    const currentFragment = location.hash.slice(1);
    setStoreFragment(currentFragment);
  }, [location.hash, setStoreFragment]);

  const setFragment = useCallback(
    (newFragment: string) => {
      const cleanFragment = newFragment.startsWith("#")
        ? newFragment.slice(1)
        : newFragment;
      navigate(`${location.pathname}${location.search}#${cleanFragment}`, {
        replace: true,
      });
      setStoreFragment(cleanFragment);
    },
    [location.pathname, location.search, navigate, setStoreFragment],
  );

  const clearFragment = useCallback(() => {
    navigate(`${location.pathname}${location.search}`, {
      replace: true,
    });
    clearStoreFragment();
  }, [location.pathname, location.search, navigate, clearStoreFragment]);

  const getFragment = useCallback(() => {
    return location.hash.slice(1);
  }, [location.hash]);

  const hasFragment = useCallback(
    (fragmentToCheck?: string) => {
      const currentFragment = location.hash.slice(1);
      if (fragmentToCheck) {
        return currentFragment === fragmentToCheck;
      }
      return !!currentFragment;
    },
    [location.hash],
  );

  return {
    fragment: fragment || location.hash.slice(1),
    setFragment,
    clearFragment,
    getFragment,
    hasFragment,
  };
}
