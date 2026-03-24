// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, act, waitFor, cleanup } from '@testing-library/react';
import { CuttlegateProvider, useFlag, useFlagVariant, useCachedFlag } from './react.js';
import type { CuttlegateConfig } from './types.js';
import type { StreamOptions, StreamConnection } from './streaming.js';
import type { EvalResult, CuttlegateClient, FlagResult } from './client.js';
import type { CachedClient } from './cached-client.js';

// Mock createClient and connectStream so tests don't make real HTTP calls.
vi.mock('./client.js', async (importOriginal) => {
  const original = await importOriginal<typeof import('./client.js')>();
  return {
    ...original,
    createClient: vi.fn(),
  };
});

vi.mock('./streaming.js', () => ({
  connectStream: vi.fn(),
}));

import { createClient } from './client.js';
import { connectStream } from './streaming.js';

const mockedCreateClient = vi.mocked(createClient);
const mockedConnectStream = vi.mocked(connectStream);

const validConfig: CuttlegateConfig = {
  baseUrl: 'https://cuttlegate.example.com',
  token: 'svc_abc',
  project: 'my-project',
  environment: 'production',
};

function makeEvalResult(
  key: string,
  enabled: boolean,
  value: string | null = null,
): EvalResult {
  return { key, enabled, value, variant: value ?? '', reason: 'default', evaluatedAt: '2026-03-21T12:00:00Z' };
}

/** Test component that renders useFlag output. */
function FlagConsumer({ flagKey }: { flagKey: string }) {
  const { enabled, loading } = useFlag(flagKey);
  return (
    <div data-testid={`flag-${flagKey}`}>
      {loading ? 'loading' : enabled ? 'enabled' : 'disabled'}
    </div>
  );
}

/** Test component that renders useFlagVariant output. */
function VariantConsumer({ flagKey }: { flagKey: string }) {
  const { variant, loading } = useFlagVariant(flagKey);
  return (
    <div data-testid={`variant-${flagKey}`}>
      {loading ? 'loading' : variant ?? 'null'}
    </div>
  );
}

let capturedStreamOptions: StreamOptions;
let mockClose: ReturnType<typeof vi.fn>;
let resolveEvaluate: (results: EvalResult[]) => void;

afterEach(() => {
  cleanup();
});

beforeEach(() => {
  vi.clearAllMocks();
  mockClose = vi.fn();

  // Default: evaluateAll returns a promise we control.
  const evaluateAllPromise = new Promise<EvalResult[]>((resolve) => {
    resolveEvaluate = resolve;
  });
  const mockClient = { evaluateAll: vi.fn().mockReturnValue(evaluateAllPromise) } as unknown as CuttlegateClient;
  mockedCreateClient.mockReturnValue(mockClient);

  // Capture stream options so tests can fire SSE events.
  mockedConnectStream.mockImplementation(
    (_config: CuttlegateConfig, options: StreamOptions): StreamConnection => {
      capturedStreamOptions = options;
      return { close: mockClose };
    },
  );
});

describe('CuttlegateProvider + useFlag', () => {
  it('returns loading state before initial evaluation', () => {
    render(
      <CuttlegateProvider config={validConfig}>
        <FlagConsumer flagKey="dark-mode" />
      </CuttlegateProvider>,
    );

    expect(screen.getByTestId('flag-dark-mode').textContent).toBe('loading');
  });

  it('returns flag state after evaluation completes', async () => {
    render(
      <CuttlegateProvider config={validConfig}>
        <FlagConsumer flagKey="dark-mode" />
      </CuttlegateProvider>,
    );

    await act(async () => {
      resolveEvaluate([makeEvalResult('dark-mode', true)]);
    });

    await waitFor(() => {
      expect(screen.getByTestId('flag-dark-mode').textContent).toBe('enabled');
    });
  });

  it('updates reactively on SSE event', async () => {
    render(
      <CuttlegateProvider config={validConfig}>
        <FlagConsumer flagKey="dark-mode" />
      </CuttlegateProvider>,
    );

    // Complete initial evaluation with dark-mode enabled.
    await act(async () => {
      resolveEvaluate([makeEvalResult('dark-mode', true)]);
    });

    await waitFor(() => {
      expect(screen.getByTestId('flag-dark-mode').textContent).toBe('enabled');
    });

    // Simulate SSE event disabling dark-mode.
    act(() => {
      capturedStreamOptions.onFlagChange({
        type: 'flag.state_changed',
        project: 'my-project',
        environment: 'production',
        flagKey: 'dark-mode',
        enabled: false,
        occurredAt: '2026-03-21T12:01:00Z',
      });
    });

    await waitFor(() => {
      expect(screen.getByTestId('flag-dark-mode').textContent).toBe('disabled');
    });
  });

  it('returns default values for unknown flags', async () => {
    render(
      <CuttlegateProvider config={validConfig}>
        <FlagConsumer flagKey="nonexistent" />
      </CuttlegateProvider>,
    );

    await act(async () => {
      resolveEvaluate([makeEvalResult('dark-mode', true)]);
    });

    await waitFor(() => {
      expect(screen.getByTestId('flag-nonexistent').textContent).toBe('disabled');
    });
  });
});

describe('CuttlegateProvider + useFlagVariant', () => {
  it('returns variant value after evaluation', async () => {
    render(
      <CuttlegateProvider config={validConfig}>
        <VariantConsumer flagKey="theme" />
      </CuttlegateProvider>,
    );

    await act(async () => {
      resolveEvaluate([makeEvalResult('theme', true, 'dark')]);
    });

    await waitFor(() => {
      expect(screen.getByTestId('variant-theme').textContent).toBe('dark');
    });
  });

  it('returns loading state before evaluation', () => {
    render(
      <CuttlegateProvider config={validConfig}>
        <VariantConsumer flagKey="theme" />
      </CuttlegateProvider>,
    );

    expect(screen.getByTestId('variant-theme').textContent).toBe('loading');
  });
});

describe('Provider lifecycle', () => {
  it('closes SSE connection on unmount', async () => {
    const { unmount } = render(
      <CuttlegateProvider config={validConfig}>
        <FlagConsumer flagKey="dark-mode" />
      </CuttlegateProvider>,
    );

    await act(async () => {
      resolveEvaluate([makeEvalResult('dark-mode', true)]);
    });

    unmount();

    expect(mockClose).toHaveBeenCalledOnce();
  });

  it('calls createClient and connectStream with the provided config', () => {
    render(
      <CuttlegateProvider config={validConfig}>
        <FlagConsumer flagKey="dark-mode" />
      </CuttlegateProvider>,
    );

    expect(mockedCreateClient).toHaveBeenCalledWith(validConfig);
    expect(mockedConnectStream).toHaveBeenCalledWith(
      validConfig,
      expect.objectContaining({ onFlagChange: expect.any(Function) }),
    );
  });
});

describe('Multiple hooks sharing state', () => {
  it('both components re-render when SSE event changes a shared flag', async () => {
    function App() {
      return (
        <CuttlegateProvider config={validConfig}>
          <FlagConsumer flagKey="dark-mode" />
          <div data-testid="second">
            <FlagConsumer flagKey="dark-mode" />
          </div>
        </CuttlegateProvider>
      );
    }

    render(<App />);

    await act(async () => {
      resolveEvaluate([makeEvalResult('dark-mode', true)]);
    });

    const flags = screen.getAllByTestId('flag-dark-mode');
    await waitFor(() => {
      expect(flags[0].textContent).toBe('enabled');
      expect(flags[1].textContent).toBe('enabled');
    });

    // SSE event disables it.
    act(() => {
      capturedStreamOptions.onFlagChange({
        type: 'flag.state_changed',
        project: 'my-project',
        environment: 'production',
        flagKey: 'dark-mode',
        enabled: false,
        occurredAt: '2026-03-21T12:01:00Z',
      });
    });

    await waitFor(() => {
      expect(flags[0].textContent).toBe('disabled');
      expect(flags[1].textContent).toBe('disabled');
    });
  });
});

describe('useFlag/useFlagVariant outside provider', () => {
  it('throws when used without CuttlegateProvider', () => {
    // Suppress React error boundary console output.
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {});

    expect(() => render(<FlagConsumer flagKey="x" />)).toThrow(
      'useFlag/useFlagVariant must be used within a CuttlegateProvider',
    );

    spy.mockRestore();
  });
});

// ---------------------------------------------------------------------------
// useCachedFlag tests
// ---------------------------------------------------------------------------

/**
 * Build a controllable CachedClient mock.
 *
 * Returns:
 * - `client` — the mock CachedClient
 * - `resolveReady` — call to resolve the ready promise
 * - `rejectReady` — call to reject the ready promise
 * - `fireSubscriber` — simulate an SSE event for a given key
 * - `subscribeCalls` — list of [key, callback] pairs passed to subscribe
 */
function makeCachedClient(initialFlags: Record<string, boolean> = {}) {
  let resolveReady!: () => void;
  let rejectReady!: (err: unknown) => void;

  const ready = new Promise<void>((res, rej) => {
    resolveReady = res;
    rejectReady = rej;
  });

  // Prevent unhandled rejection in tests that don't reject.
  ready.catch(() => {});

  const subscribeCalls: Array<[string, (enabled: boolean) => void]> = [];
  const subscriberMap = new Map<string, Set<(enabled: boolean) => void>>();

  const client: CachedClient = {
    ready,
    close: vi.fn(),
    evaluate: vi.fn().mockImplementation((key: string): Promise<EvalResult> => {
      const enabled = initialFlags[key] ?? false;
      const reason = key in initialFlags ? 'default' : 'not_found';
      return Promise.resolve({
        key,
        enabled,
        value: null,
        variant: enabled ? 'true' : '',
        reason,
        evaluatedAt: '2026-03-21T12:00:00Z',
      });
    }),
    evaluateAll: vi.fn().mockResolvedValue([]),
    bool: vi.fn().mockImplementation((key: string): Promise<boolean> => {
      return Promise.resolve(initialFlags[key] ?? false);
    }),
    string: vi.fn().mockImplementation((key: string): Promise<string> => {
      return Promise.resolve(initialFlags[key] ? 'true' : '');
    }),
    evaluateFlag: vi.fn().mockImplementation((key: string): Promise<FlagResult> => {
      const enabled = initialFlags[key] ?? false;
      const reason = key in initialFlags ? 'default' : 'not_found';
      return Promise.resolve({ enabled, value: null, variant: '', reason });
    }),
    subscribe: vi.fn().mockImplementation((key: string, cb: (enabled: boolean) => void) => {
      subscribeCalls.push([key, cb]);
      let cbs = subscriberMap.get(key);
      if (!cbs) {
        cbs = new Set();
        subscriberMap.set(key, cbs);
      }
      cbs.add(cb);
      return () => {
        subscriberMap.get(key)?.delete(cb);
      };
    }),
  };

  function fireSubscriber(key: string, enabled: boolean): void {
    subscriberMap.get(key)?.forEach((cb) => cb(enabled));
  }

  return { client, resolveReady, rejectReady, fireSubscriber, subscribeCalls };
}

/** Test component for useCachedFlag. */
function CachedFlagConsumer({ client, flagKey }: { client: CachedClient; flagKey: string }) {
  const { enabled, loading } = useCachedFlag(client, flagKey);
  return (
    <div data-testid={`cached-${flagKey}`}>
      {loading ? 'loading' : enabled ? 'enabled' : 'disabled'}
    </div>
  );
}

describe('useCachedFlag', () => {
  afterEach(() => {
    cleanup();
  });

  it('@happy: returns loading state before ready resolves', () => {
    const { client } = makeCachedClient({ 'my-flag': true });

    render(<CachedFlagConsumer client={client} flagKey="my-flag" />);

    expect(screen.getByTestId('cached-my-flag').textContent).toBe('loading');
  });

  it('@happy: returns flag value after ready resolves with flag enabled', async () => {
    const { client, resolveReady } = makeCachedClient({ 'my-flag': true });

    render(<CachedFlagConsumer client={client} flagKey="my-flag" />);

    await act(async () => {
      resolveReady();
    });

    await waitFor(() => {
      expect(screen.getByTestId('cached-my-flag').textContent).toBe('enabled');
    });
  });

  it('@happy: returns enabled:false after ready when flag is disabled in cache', async () => {
    const { client, resolveReady } = makeCachedClient({ 'my-flag': false });

    render(<CachedFlagConsumer client={client} flagKey="my-flag" />);

    await act(async () => {
      resolveReady();
    });

    await waitFor(() => {
      expect(screen.getByTestId('cached-my-flag').textContent).toBe('disabled');
    });
  });

  it('@happy: reactive update — subscribe callback fires → hook re-renders with new value', async () => {
    const { client, resolveReady, fireSubscriber } = makeCachedClient({ 'my-flag': true });

    render(<CachedFlagConsumer client={client} flagKey="my-flag" />);

    await act(async () => {
      resolveReady();
    });

    await waitFor(() => {
      expect(screen.getByTestId('cached-my-flag').textContent).toBe('enabled');
    });

    act(() => {
      fireSubscriber('my-flag', false);
    });

    await waitFor(() => {
      expect(screen.getByTestId('cached-my-flag').textContent).toBe('disabled');
    });
  });

  it('@edge: ready rejects → returns enabled:false, loading:false, no error thrown', async () => {
    const { client, rejectReady } = makeCachedClient();

    render(<CachedFlagConsumer client={client} flagKey="my-flag" />);

    // Starts loading.
    expect(screen.getByTestId('cached-my-flag').textContent).toBe('loading');

    await act(async () => {
      rejectReady(new Error('unauthorized'));
    });

    await waitFor(() => {
      expect(screen.getByTestId('cached-my-flag').textContent).toBe('disabled');
    });
  });

  it('@edge: key does not exist in cache after ready resolves → returns enabled:false, loading:false', async () => {
    // 'other-flag' is in cache, 'my-flag' is not.
    const { client, resolveReady } = makeCachedClient({ 'other-flag': true });

    render(<CachedFlagConsumer client={client} flagKey="my-flag" />);

    await act(async () => {
      resolveReady();
    });

    await waitFor(() => {
      // Not stuck in loading — unknown key resolves to disabled.
      expect(screen.getByTestId('cached-my-flag').textContent).toBe('disabled');
    });
  });

  it('@edge: hook unmounts before ready resolves → no setState after unmount', async () => {
    const { client, resolveReady } = makeCachedClient({ 'my-flag': true });

    const { unmount } = render(<CachedFlagConsumer client={client} flagKey="my-flag" />);

    // Unmount before ready resolves.
    unmount();

    // Now resolve — should not trigger any setState (no warning emitted).
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    await act(async () => {
      resolveReady();
    });

    // No React state-update-after-unmount warning.
    expect(consoleSpy).not.toHaveBeenCalledWith(
      expect.stringContaining("Can't perform a React state update"),
    );
    consoleSpy.mockRestore();
  });

  it('@edge: hook unmounts → unsubscribe is called', async () => {
    const { client, resolveReady } = makeCachedClient({ 'my-flag': true });
    const mockUnsubscribe = vi.fn();
    vi.mocked(client.subscribe).mockReturnValueOnce(mockUnsubscribe);

    const { unmount } = render(<CachedFlagConsumer client={client} flagKey="my-flag" />);

    await act(async () => {
      resolveReady();
    });

    unmount();

    expect(mockUnsubscribe).toHaveBeenCalledOnce();
  });

  it('@edge: key changes while mounted → old subscription cleaned up, new subscription started', async () => {
    const { client, resolveReady, fireSubscriber } = makeCachedClient({
      'flag-a': true,
      'flag-b': false,
    });

    const { rerender } = render(<CachedFlagConsumer client={client} flagKey="flag-a" />);

    await act(async () => {
      resolveReady();
    });

    await waitFor(() => {
      expect(screen.getByTestId('cached-flag-a').textContent).toBe('enabled');
    });

    // Change key prop to flag-b.
    act(() => {
      rerender(<CachedFlagConsumer client={client} flagKey="flag-b" />);
    });

    await waitFor(() => {
      expect(screen.getByTestId('cached-flag-b').textContent).toBe('disabled');
    });

    // SSE event for flag-a should not trigger a re-render (old subscription cleaned up).
    act(() => {
      fireSubscriber('flag-a', false);
    });

    // Still shows flag-b disabled.
    expect(screen.getByTestId('cached-flag-b').textContent).toBe('disabled');

    // SSE event for flag-b should trigger a re-render.
    act(() => {
      fireSubscriber('flag-b', true);
    });

    await waitFor(() => {
      expect(screen.getByTestId('cached-flag-b').textContent).toBe('enabled');
    });
  });
});
