/**
 * Audit MODULES list paths vs Gin GET routes.
 * Usage: node scripts/audit_module_routes.mjs
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const routes = JSON.parse(fs.readFileSync(path.join(ROOT, 'scripts/gin_routes.json'), 'utf8'))
const getSet = new Set(routes.filter((r) => r.method.toUpperCase() === 'GET').map((r) => r.path))
const postSet = new Set(routes.filter((r) => r.method.toUpperCase() === 'POST').map((r) => r.path))
const text = fs.readFileSync(path.join(ROOT, 'web/packages/shared/src/generated/modules.ts'), 'utf8')
const blocks = [...text.matchAll(/\{[\s\S]*?domain: '([^']+)',\s*module: '([^']+)',\s*phase: (\d+),\s*list: '([^']*)',([\s\S]*?)\n  \},/g)]
const bad = []
for (const b of blocks) {
  const domain = b[1]
  const module = b[2]
  const list = b[4]
  if (!list) continue // action-only / unmapped is OK for list loader
  if (list.includes('{')) {
    bad.push({ domain, module, list, reason: 'has_param' })
    continue
  }
  const p = '/api/v1' + list
  if (!getSet.has(p)) {
    bad.push({ domain, module, list, reason: 'no_GET', hasPost: postSet.has(p) })
  }
}
console.log('total modules', blocks.length)
console.log('bad list mappings', bad.length)
for (const x of bad) {
  console.log([x.reason, x.hasPost ? 'POST' : '-', x.domain, x.module, x.list].join('\t'))
}
process.exit(bad.length ? 1 : 0)
