/**
 * Build retained web apps and assemble under web/dist/
 *
 * web/dist/
 *   index.html          (portal)
 *   admin/
 *   front/              (unified employee)
 *   front/boss/
 */
import { cpSync, mkdirSync, rmSync, existsSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawnSync } from 'node:child_process'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const dist = join(root, 'dist')

const map = [
  { workspace: '@erp/portal', from: 'apps/portal/dist', to: '.' },
  { workspace: '@erp/admin', from: 'apps/admin/dist', to: 'admin' },
  { workspace: '@erp/employee', from: 'apps/employee/dist', to: 'front' },
  { workspace: '@erp/boss', from: 'apps/boss/dist', to: 'front/boss' },
]

function run(cmd, args) {
  const r = spawnSync(cmd, args, { cwd: root, stdio: 'inherit', shell: true })
  if (r.status !== 0) process.exit(r.status || 1)
}

console.log('→ building apps…')
for (const m of map) {
  run('npm', ['run', 'build', '-w', m.workspace])
}

if (existsSync(dist)) rmSync(dist, { recursive: true, force: true })
mkdirSync(dist, { recursive: true })

for (const m of map) {
  const src = join(root, m.from)
  if (!existsSync(src)) {
    console.error('missing build output:', src)
    process.exit(1)
  }
  const dest = m.to === '.' ? dist : join(dist, m.to)
  mkdirSync(dirname(dest === dist ? join(dist, '_') : dest), { recursive: true })
  cpSync(src, dest, { recursive: true })
  console.log('✓', m.from, '→', m.to === '.' ? 'dist/' : `dist/${m.to}/`)
}

console.log('\nDone. Preview: npx serve dist  (or npm run preview:dist)')
