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

// Credits: Acorn1010 - https://gist.github.com/acorn1010/9f4621d3dfc33052ffd84f6c2a06d4d6.

import { SetStateAction, useCallback } from 'react';
import { create } from 'zustand';

export type EqualityFn<T> = (left: T | null | undefined, right: T | null | undefined) => boolean;
// eslint-disable-next-line @typescript-eslint/ban-types
const isFunction = (fn: unknown): fn is Function => typeof fn === 'function';

/**
 * Create a global state
 *
 * It returns a set of functions
 * - `use`: Works like React.useState. "Registers" the component as a listener on that key
 * - `get`: retrieves a key without a re-render
 * - `set`: sets a key. Causes re-renders on any listeners
 * - `getAll`: retrieves the entire state (all keys) as an object without a re-render
 * - `reset`: resets the state back to its initial value
 *
 * @example
 * import { createStore } from 'create-store';
 *
 * const store = createStore({ count: 0 });
 *
 * const Component = () => {
 *   const [count, setCount] = store.use("count");
 *   ...
 * };
 */
export const createGlobalStore = <State extends object>(initialState: State) => {
  // NOTE: Not using structuredClone because browser support only goes about 2 years back.
  const store = create<State>(() => deepClone(initialState));

  const setter = <T extends keyof State>(key: T, value: SetStateAction<State[T]>) => {
    if (typeof value === 'function') {
      store.setState(
        (prevValue) =>
          ({
            [key]: (value as Function)(prevValue[key])
          }) as unknown as Partial<State>
      );
    } else {
      store.setState({ [key]: value } as unknown as Partial<State>);
    }
  };
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
      const result = store((state) => state[key], equalityFn);
      // eslint-disable-next-line react-hooks/rules-of-hooks
      const keySetter = useCallback((value: SetStateAction<State[K]>) => setter(key, value), [key]);
      return [result, keySetter];
    },

    /** Listens on the entire state, causing a re-render when anything in the state changes. */
    useAll: () => store((state) => state),

    /** Deletes a `key` from state, causing a re-render for anything listening. */
    delete<K extends keyof State>(key: K) {
      store.setState((prevState) => {
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
    reset: () => store.setState(deepClone(initialState), true)
  };
};

/**
 * Returns a wrapped `store` that can't be modified. Useful when you want to
 * control who is able to write to a store.
 */
export function createReadonlyStore<T extends ReturnType<typeof createGlobalStore>>(store: T) {
  type State = ReturnType<T['getAll']>;
  return {
    get: store.get,
    getAll: store.getAll,
    use: <K extends keyof State>(key: K, equalityFn?: EqualityFn<State[K]>) =>
      (store.use as any)(key, undefined, equalityFn)[0] as State[K] | undefined | null,
    useAll: store.useAll
  };
}

/**
 * Deeply copies objects. Borrowed from just-clone, but with some nicer types.
 * See: https://github.com/angus-c/just/blob/master/packages/collection-clone/index.cjs
 */
function deepClone<T>(obj: T): T {
  let result = obj;
  const type = {}.toString.call(obj).slice(8, -1);
  if (type === 'Set') {
    return new Set([...(obj as Set<any>)].map((value) => deepClone(value))) as any;
  }
  if (type === 'Map') {
    return new Map([...(obj as Set<any>)].map((kv) => [deepClone(kv[0]), deepClone(kv[1])])) as any;
  }
  if (type === 'Date') {
    return new Date((obj as Date).getTime()) as any;
  }
  if (type === 'RegExp') {
    return RegExp((obj as RegExp).source as string, getRegExpFlags(obj as RegExp)) as any;
  }
  if (type === 'Array' || type === 'Object') {
    result = Array.isArray(obj) ? [] : ({} as any);
    for (const key in obj) {
      // include prototype properties
      result[key] = deepClone(obj[key]);
    }
  }
  // primitives and non-supported objects (e.g. functions) land here
  return result;
}

// Define a custom persisting middleware
function getRegExpFlags(regExp: RegExp): string {
  if ((typeof regExp.source as any).flags === 'string') {
    return (regExp.source as any).flags;
  }
  const flags = [];
  regExp.global && flags.push('g');
  regExp.ignoreCase && flags.push('i');
  regExp.multiline && flags.push('m');
  regExp.sticky && flags.push('y');
  regExp.unicode && flags.push('u');
  return flags.join('');
}
