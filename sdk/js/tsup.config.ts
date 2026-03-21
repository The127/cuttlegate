import { defineConfig } from 'tsup';

export default defineConfig([
  {
    entry: { index: 'src/index.ts' },
    format: ['esm', 'cjs'],
    dts: true,
    clean: true,
    sourcemap: true,
  },
  {
    entry: { 'browser/index': 'src/browser.ts' },
    format: ['esm'],
    platform: 'browser',
    dts: true,
    sourcemap: true,
  },
  {
    entry: { testing: 'src/testing.ts' },
    format: ['esm', 'cjs'],
    dts: true,
    sourcemap: true,
  },
]);
