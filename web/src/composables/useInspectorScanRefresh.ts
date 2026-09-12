import { onScopeDispose, watch } from "vue";

// Check availability even while dirty; apply metadata only once edits/operations settle.
export function useInspectorScanRefresh(
  revision: () => number | undefined,
  blocked: () => boolean,
  refresh: (
    canApply: () => boolean,
    signal: AbortSignal,
    isCurrent: () => boolean,
  ) => Promise<void>,
) {
  let pending = 0;
  let checked = 0;
  let applied = 0;
  let running = false;
  let disposed = false;
  let controller: AbortController | undefined;
  const checkWaiters = new Set<(available: boolean) => void>();
  function settleChecks() {
    if (!disposed && checked !== pending) return;
    for (const resolve of checkWaiters) resolve(!disposed);
    checkWaiters.clear();
  }
  onScopeDispose(() => {
    disposed = true;
    controller?.abort();
    settleChecks();
  });

  async function drain() {
    if (
      disposed ||
      running ||
      applied === pending ||
      (blocked() && checked === pending)
    )
      return;
    running = true;
    const target = pending;
    controller = new AbortController();
    const signal = controller.signal;
    const timer = setTimeout(() => controller?.abort(), 15_000);
    const canApply = () => !disposed && !blocked() && pending === target;
    try {
      await refresh(canApply, signal, () => !disposed && pending === target);
      checked = target;
      if (canApply()) applied = target;
    } finally {
      clearTimeout(timer);
      running = false;
      settleChecks();
      if (
        !disposed &&
        (checked !== pending || (!blocked() && applied !== pending))
      )
        void drain();
    }
  }

  function retry() {
    pending++;
    void drain();
  }
  watch(revision, retry);
  watch(blocked, () => {
    void drain();
  });
  // Mutation continuations wait only for availability, never for draft replacement
  // (which is deliberately blocked by the mutation itself).
  function waitForCheck(): Promise<boolean> {
    if (disposed || checked === pending) return Promise.resolve(!disposed);
    return new Promise((resolve) => checkWaiters.add(resolve));
  }
  return { retry, waitForCheck };
}
