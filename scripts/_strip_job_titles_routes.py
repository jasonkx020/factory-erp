from pathlib import Path

p = Path("internal/apigen/routes_gen.go")
t = p.read_text(encoding="utf-8")
lines = t.splitlines(keepends=True)
out = [ln for ln in lines if "hr/job-titles" not in ln]
if len(out) == len(lines):
    raise SystemExit("no job-titles lines found")
p.write_text("".join(out), encoding="utf-8")
print(f"removed {len(lines) - len(out)} lines")
