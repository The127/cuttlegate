import fs from 'fs'
import path from 'path'

const ROOT = process.cwd()
const ROUTES_DIR = path.join(ROOT, 'web/src/routes')
const COMPONENTS_DIR = path.join(ROOT, 'web/src/components')
const HOOKS_DIR = path.join(ROOT, 'web/src/hooks')
const API_FILE = path.join(ROOT, 'web/src/api.ts')
const OUT = path.join(ROOT, 'docs/frontend-index.md')

function listFiles(dir: string, ext: string): string[] {
  if (!fs.existsSync(dir)) return []
  return fs
    .readdirSync(dir, { recursive: true, encoding: 'utf8' })
    .filter((f) => f.endsWith(ext) && !f.includes('__tests__') && !f.includes('/test/'))
}

function indexComment(content: string): string {
  const m = content.match(/\/\/ @index (.+)/)
  return m ? m[1].trim() : ''
}

// Derive URL pattern from a TanStack Router flat-file path.
// Returns null for layout/internal files (prefixed with _ or __).
function routeUrl(rel: string): string | null {
  const noExt = rel.replace(/\.tsx$/, '')
  const segments = noExt.split('/').flatMap((p) => p.split('.'))
  if (segments.some((s) => s.startsWith('_'))) return null
  const url = segments
    .filter((s) => s !== 'index')
    .map((s) => (s.startsWith('$') ? `:${s.slice(1)}` : s))
    .join('/')
  return '/' + url
}

function routeParams(rel: string): string {
  return (rel.match(/\$[a-zA-Z]+/g) ?? []).join(', ')
}

function exportedNames(content: string): string[] {
  const names: string[] = []
  for (const m of content.matchAll(/^export (?:async )?function (\w+)|^export const (\w+)/gm)) {
    names.push(m[1] ?? m[2])
  }
  return names
}

function exportedSignatures(content: string): string[] {
  const sigs: string[] = []
  for (const m of content.matchAll(/^export (?:async )?function \w+[^{]*/gm)) {
    sigs.push(m[0].trimEnd())
  }
  for (const m of content.matchAll(/^export class \w+/gm)) {
    sigs.push(m[0])
  }
  return sigs
}

function appendExportTable(
  out: string[],
  dir: string,
  ext: string,
  heading: string,
  kind: string,
): void {
  out.push(`## ${heading}`, '')
  out.push(`| ${kind} | File | Description |`)
  out.push('|---|---|---|')
  for (const rel of listFiles(dir, ext).sort()) {
    const content = fs.readFileSync(path.join(dir, rel), 'utf8')
    const names = exportedNames(content)
    const desc = indexComment(content)
    for (const name of names) {
      out.push(`| \`${name}\` | \`${rel}\` | ${desc || '—'} |`)
    }
  }
  out.push('')
}

const lines: string[] = [
  '# Frontend Index',
  '',
  `_Generated ${new Date().toISOString().slice(0, 16).replace('T', ' ')} UTC. Read this at session start for frontend orientation._`,
  '',
]

// Routes
lines.push('## Routes (`web/src/routes/`)', '')
lines.push('| Route file | URL pattern | Params | Description |')
lines.push('|---|---|---|---|')
for (const rel of listFiles(ROUTES_DIR, '.tsx').sort()) {
  const url = routeUrl(rel)
  if (!url) continue
  const content = fs.readFileSync(path.join(ROUTES_DIR, rel), 'utf8')
  const params = routeParams(rel)
  const desc = indexComment(content)
  lines.push(`| \`${rel}\` | \`${url}\` | ${params || '—'} | ${desc || '—'} |`)
}
lines.push('')

appendExportTable(lines, COMPONENTS_DIR, '.tsx', 'Components (`web/src/components/`)', 'Component')
appendExportTable(lines, HOOKS_DIR, '.ts', 'Hooks (`web/src/hooks/`)', 'Hook')

// API surface
lines.push('## API surface (`web/src/api.ts`)', '')
lines.push('```')
for (const sig of exportedSignatures(fs.readFileSync(API_FILE, 'utf8'))) {
  lines.push(sig)
}
lines.push('```', '')

fs.writeFileSync(OUT, lines.join('\n') + '\n')
console.log(`Frontend index written to ${OUT} (${lines.length} lines)`)
