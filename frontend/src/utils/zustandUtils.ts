import create, { StoreApi, UseBoundStore } from 'zustand';


/**
 * @description A wrapper around zustand's create function that adds a few extra features to the store
 * @param state
 */
function createTyped<T>(state: (set: (state: T) => T, get: () => T) => T) {
  return create(state as any) as UseBoundStore<StoreApi<T>>;
}

export default createTyped;