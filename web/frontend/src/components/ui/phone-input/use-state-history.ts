/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { useCallback, useMemo, useState } from "react";

export interface UseStateHistoryHandlers<T> {
  set: (value: T) => void;
  back: (steps?: number) => void;
  forward: (steps?: number) => void;
}

export interface StateHistory<T> {
  history: T[];
  current: number;
}

export function useStateHistory<T>(
  initialValue: T,
): [T, UseStateHistoryHandlers<T>, StateHistory<T>] {
  const [state, setState] = useState<StateHistory<T>>({
    history: [initialValue],
    current: 0,
  });

  const set = useCallback(
    (val: T) =>
      setState((currentState) => {
        const nextState = [
          ...currentState.history.slice(0, currentState.current + 1),
          val,
        ];
        return {
          history: nextState,
          current: nextState.length - 1,
        };
      }),
    [],
  );

  const back = useCallback(
    (steps = 1) =>
      setState((currentState) => ({
        history: currentState.history,
        current: Math.max(0, currentState.current - steps),
      })),
    [],
  );

  const forward = useCallback(
    (steps = 1) =>
      setState((currentState) => ({
        history: currentState.history,
        current: Math.min(
          currentState.history.length - 1,
          currentState.current + steps,
        ),
      })),
    [],
  );

  const handlers = useMemo(
    () => ({ set, forward, back }),
    [set, forward, back],
  );

  return [state.history[state.current], handlers, state];
}
