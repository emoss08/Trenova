/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { SetStateAction, useCallback } from "react";
import { create } from "zustand";

const isNil = (value: any): value is undefined | null => value === null || value === undefined;
export type EqualityFn<T> = (left: T | null | undefined, right: T | null | undefined) => boolean

const isFunction = (fn: unknown): fn is Function => typeof fn === "function";

export const createGlobalStore = <State extends object>(initialState: State) => {
  const store = create<State>(() => structuredClone(initialState));

  const setter = <T extends keyof State>(key: T, value: SetStateAction<State[T]>, setAuth?: any) => {
    if (isFunction(value)) {
      store.setState((prevValue: any) => ({ [key]: value(prevValue[key]) } as unknown as Partial<State>));
    } else {
      store.setState({ [key]: value } as unknown as Partial<State>);
    }

    // Call setAuth function if it exists and the key is "auth"
    if (key === "auth" && setAuth) {
      setAuth(store.getState());
    }
  };

  const getState = () => store.getState();

  return {
    /** Works like React.useState. "Registers" the component as a listener on that key. */
    use<K extends keyof State>(
      key: K,
      defaultValue?: State[K],
      equalityFn?: EqualityFn<State[K]>
    ): [State[K], (value: SetStateAction<State[K]>) => void] {
      // If state isn't defined for a given defaultValue, set it.
      if (defaultValue !== undefined && !(key in store.getState())) {
        setter(key, defaultValue);
      }
      const result = store((state: any) => state[key], equalityFn);
      // eslint-disable-next-line react-hooks/rules-of-hooks
      const keySetter = useCallback((value: SetStateAction<State[K]>) => setter(key, value), [key]);
      return [result, keySetter] as any;
    },

    /** Listens on the entire state, causing a re-render when anything in the state changes. */
    useAll: () => store((state: any) => state),

    /** Deletes a `key` from state, causing a re-render for anything listening. */
    delete<K extends keyof State>(key: K) {
      store.setState((prevState: State) => {
        const { [key]: _, ...rest } = prevState;
        return rest as Partial<State>;
      }, true);
    },

    /** Retrieves the current `key` value. Does _not_ listen on state changes (meaning no re-renders). */
    get<K extends keyof State>(key: K) {
      return store.getState()[key];
    },

    /** Retrieves the entire state. Does _not_ listen on state changes (meaning no re-renders). */
    getAll: () => store.getState(),

    /** Returns `true` if `key` is in the state. */
    has<K extends keyof State>(key: K) {
      return key in store.getState();
    },

    /** Sets a `key`, triggering a re-render for all listeners. */
    set: setter,

    /** Sets the entire state, removing any keys that aren't present in `state`. */
    setAll: (state: State) => store.setState(state, true),

    /** Updates the keys in `state`, leaving any keys / values not in `state` unchanged. */
    update: (state: Partial<State>) => store.setState(state, false),

    /** Resets the entire state back to its initial state when the store was created. */
    reset: () => store.setState(structuredClone(initialState), true),

    /** Returns the current state */
    getState
  };
};

/**
 * Returns a wrapped `store` that can't be modified. Useful when you want to
 * control who is able to write to a store.
 */
export function createReadonlyStore<T extends ReturnType<typeof createGlobalStore>>(store: T) {
  type State = ReturnType<T["getAll"]>
  return {
    get: store.get,
    getAll: store.getAll,
    use: <K extends keyof State>(key: K, equalityFn?: EqualityFn<State[K]>) =>
      (store.use as any)(key, undefined, equalityFn)[0] as State[K] | undefined | null,
    useAll: store.useAll
  };
}

export function storeHasValues<T extends object>(store: T): boolean {
  return Object.values(store).some((value) => {
    if (typeof value === "object") {
      return Object.values(value).some((innerValue) => !isNil(innerValue) && innerValue !== "");
    }
    return !isNil(value) && value !== "";
  });
}
