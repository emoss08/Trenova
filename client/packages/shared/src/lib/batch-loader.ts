/** Answers many keys in one request; a key missing from the answer resolves to undefined. */
export type BatchFetch<K, V> = (keys: readonly K[]) => Promise<ReadonlyMap<K, V>>;

export type BatchLoader<K, V> = {
  load: (key: K) => Promise<V | undefined>;
};

type Pending<V> = {
  resolve: (value: V | undefined) => void;
  reject: (error: unknown) => void;
};

/**
 * Collects the keys asked for within one tick and fetches them together, the way a
 * server dataloader does. Rows rendering their own cells ask one key each; the table
 * makes one request for all of them. A key asked for twice in a batch is fetched once.
 */
export function createBatchLoader<K, V>(
  fetch: BatchFetch<K, V>,
  options: { maxBatchSize?: number } = {},
): BatchLoader<K, V> {
  const maxBatchSize = options.maxBatchSize ?? 100;
  let queue = new Map<K, Pending<V>[]>();
  let scheduled = false;

  const dispatch = (batch: Map<K, Pending<V>[]>) => {
    fetch([...batch.keys()]).then(
      (answers) => {
        for (const [key, waiters] of batch) {
          const value = answers.get(key);
          for (const waiter of waiters) waiter.resolve(value);
        }
      },
      (error: unknown) => {
        for (const waiters of batch.values()) {
          for (const waiter of waiters) waiter.reject(error);
        }
      },
    );
  };

  const flush = () => {
    scheduled = false;
    const keys = [...queue.entries()];
    queue = new Map();
    for (let start = 0; start < keys.length; start += maxBatchSize) {
      dispatch(new Map(keys.slice(start, start + maxBatchSize)));
    }
  };

  return {
    load: (key) =>
      new Promise<V | undefined>((resolve, reject) => {
        const waiters = queue.get(key);
        if (waiters) waiters.push({ resolve, reject });
        else queue.set(key, [{ resolve, reject }]);
        if (!scheduled) {
          scheduled = true;
          setTimeout(flush, 0);
        }
      }),
  };
}
