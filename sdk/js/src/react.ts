import {
  createContext,
  createElement,
  useContext,
  useEffect,
  useRef,
  useState,
} from 'react';
import type { ReactNode } from 'react';
import type { CuttlegateConfig } from './types.js';
import type { EvaluationResult } from './client.js';
import { createClient } from './client.js';
import { connectStream } from './streaming.js';
import type { CachedClient } from './cached-client.js';

/** Props for the CuttlegateProvider component. */
export interface CuttlegateProviderProps {
  config: CuttlegateConfig;
  children: ReactNode;
}

interface FlagState {
  flags: Map<string, EvaluationResult>;
  loading: boolean;
}

const CuttlegateContext = createContext<FlagState | null>(null);

/**
 * Provider component that establishes a Cuttlegate SDK connection.
 *
 * Calls `evaluate()` on mount to get initial flag state, then opens an SSE
 * stream via `connectStream()` to receive real-time updates. Closes the
 * connection on unmount.
 */
export function CuttlegateProvider({
  config,
  children,
}: CuttlegateProviderProps): ReactNode {
  const [state, setState] = useState<FlagState>({
    flags: new Map(),
    loading: true,
  });

  // Use a ref to hold the config so the effect doesn't re-run on every render
  // if the consumer passes an inline object literal.
  const configRef = useRef(config);
  configRef.current = config;

  useEffect(() => {
    const cfg = configRef.current;
    const client = createClient(cfg);
    let closed = false;

    client.evaluate({ user_id: '', attributes: {} }).then(
      (results) => {
        if (closed) return;
        const flags = new Map<string, EvaluationResult>();
        for (const r of results) {
          flags.set(r.key, r);
        }
        setState({ flags, loading: false });
      },
      () => {
        if (closed) return;
        setState((prev) => ({ ...prev, loading: false }));
      },
    );

    const conn = connectStream(cfg, {
      onFlagChange: (event) => {
        if (closed) return;
        setState((prev) => {
          const next = new Map(prev.flags);
          const existing = next.get(event.flagKey);
          next.set(event.flagKey, {
            key: event.flagKey,
            enabled: event.enabled,
            value: existing?.value ?? null,
            reason: existing?.reason ?? 'default',
            evaluatedAt: event.occurredAt,
          });
          return { flags: next, loading: prev.loading };
        });
      },
      onConnected: (reconnect) => {
        if (closed || !reconnect) return;
        // Re-fetch current flag state on reconnect to close the missed-events gap.
        client.evaluate({ user_id: '', attributes: {} }).then(
          (results) => {
            if (closed) return;
            const flags = new Map<string, EvaluationResult>();
            for (const r of results) {
              flags.set(r.key, r);
            }
            setState((prev) => ({ flags, loading: prev.loading }));
          },
          () => {
            // Silently ignore — stream will still deliver future updates.
          },
        );
      },
    });

    return () => {
      closed = true;
      conn.close();
    };
  }, []);

  return createElement(CuttlegateContext.Provider, { value: state }, children);
}

function useFlagState(): FlagState {
  const ctx = useContext(CuttlegateContext);
  if (ctx === null) {
    throw new Error(
      'useFlag/useFlagVariant must be used within a CuttlegateProvider',
    );
  }
  return ctx;
}

/** Returns whether a flag is enabled. Reactively updates on SSE events. */
export function useFlag(key: string): { enabled: boolean; loading: boolean } {
  const { flags, loading } = useFlagState();
  const flag = flags.get(key);
  if (loading) {
    return { enabled: false, loading: true };
  }
  return { enabled: flag?.enabled ?? false, loading: false };
}

/** Returns the active variant key for a flag. Reactively updates on SSE events. */
export function useFlagVariant(
  key: string,
): { value: string | null; loading: boolean } {
  const { flags, loading } = useFlagState();
  const flag = flags.get(key);
  if (loading) {
    return { value: null, loading: true };
  }
  return { value: flag?.value ?? null, loading: false };
}

/**
 * Hook for consuming a single flag from a `CachedClient` instance.
 *
 * Returns `{ enabled: false, loading: true }` until `client.ready` resolves,
 * then `{ enabled: <cached value>, loading: false }`. Reactively re-renders
 * when an SSE event updates the flag in the cache via `client.subscribe`.
 * Returns `{ enabled: false, loading: false }` if `client.ready` rejects.
 *
 * The `client` reference must be stable (module-level singleton or useMemo).
 * An unstable reference causes the effect to re-run on every render.
 */
export function useCachedFlag(
  client: CachedClient,
  key: string,
): { enabled: boolean; loading: boolean } {
  const [state, setState] = useState<{ enabled: boolean; loading: boolean }>({
    enabled: false,
    loading: true,
  });

  useEffect(() => {
    let cancelled = false;

    // Resolve the initial value once ready, then subscribe to live updates.
    client.ready.then(
      () => {
        if (cancelled) return;
        // Read the current cached value now that hydration is complete.
        void client.evaluateFlag(key, { user_id: '', attributes: {} }).then((result) => {
          if (cancelled) return;
          setState({ enabled: result.enabled, loading: false });
        });
      },
      () => {
        // ready rejected — surface as enabled:false, loading:false.
        if (cancelled) return;
        setState({ enabled: false, loading: false });
      },
    );

    // Subscribe to live SSE updates for this key.
    // The callback fires after the cache is updated, with the new enabled value.
    const unsubscribe = client.subscribe(key, (enabled) => {
      if (cancelled) return;
      setState({ enabled, loading: false });
    });

    return () => {
      cancelled = true;
      unsubscribe();
    };
  }, [client, key]);

  return state;
}
