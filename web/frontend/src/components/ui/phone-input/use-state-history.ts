/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
