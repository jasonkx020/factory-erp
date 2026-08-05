/**
 * Build retained web apps and assemble under web/dist/
 *
 * web/dist/
 *   index.html          (portal)
 *   admin/
 *   front/boss/         (老板驾驶舱；员工现场端仅 Flutter App)
 */
import { cpSync, mkdirSync, rmSync, existsSync, writeFileSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawnSync } from 'node:child_process'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const dist = join(root, 'dist')

const map = [
  { workspace: '@erp/portal', from: 'apps/portal/dist', to: '.' },
  { workspace: '@erp/admin', from: 'apps/admin/dist', to: 'admin' },
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
mkdirSync(join(dist, 'front'), { recursive: true })
writeFileSync(
  join(dist, 'front', 'index.html'),
  `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"/><title>员工端已迁移</title>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<style>body{font-family:system-ui,sans-serif;max-width:40rem;margin:3rem auto;padding:0 1rem;color:#1a2b34}
a{color:#0d7a6f}</style></head><body>
<h1>员工现场端已改为 App</h1>
<p>统一员工 Web（原 /front/）已下线。请使用 Flutter 员工 App（Android/iOS），详见仓库 <code>mobile/README.md</code>。</p>
<p><a href="/">返回入口</a> · <a href="/admin/">管理后台</a> · <a href="/front/boss/">老板驾驶舱</a></p>
</body></html>
`,
)

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
