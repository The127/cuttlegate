// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, act, waitFor, cleanup } from '@testing-library/react';
import { CuttlegateProvider, useFlag, useFlagVariant } from './react.js';
import type { CuttlegateConfig } from './types.js';
import type { StreamOptions, StreamConnection } from './streaming.js';
import type { EvaluationResult, CuttlegateClient } from './client.js';

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
): EvaluationResult {
  return { key, enabled, value, reason: 'default', evaluatedAt: '2026-03-21T12:00:00Z' };
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
  const { value, loading } = useFlagVariant(flagKey);
  return (
    <div data-testid={`variant-${flagKey}`}>
      {loading ? 'loading' : value ?? 'null'}
    </div>
  );
}

let capturedStreamOptions: StreamOptions;
let mockClose: ReturnType<typeof vi.fn>;
let resolveEvaluate: (results: EvaluationResult[]) => void;

afterEach(() => {
  cleanup();
});

beforeEach(() => {
  vi.clearAllMocks();
  mockClose = vi.fn();

  // Default: evaluate returns a promise we control.
  const evaluatePromise = new Promise<EvaluationResult[]>((resolve) => {
    resolveEvaluate = resolve;
  });
  const mockClient = { evaluate: vi.fn().mockReturnValue(evaluatePromise) } as unknown as CuttlegateClient;
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
