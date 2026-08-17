import * as XLSX from 'xlsx'

/** 将二维表写入工作簿并触发浏览器下载 .xlsx */
export function downloadExcel(sheets: { name: string; rows: (string | number | null | undefined)[][] }[], filename: string) {
  const wb = XLSX.utils.book_new()
  for (const s of sheets) {
    const ws = XLSX.utils.aoa_to_sheet(
      s.rows.map((row) => row.map((c) => (c == null ? '' : c))),
    )
    // 按列内容估算列宽
    const colWidths: number[] = []
    for (const row of s.rows) {
      row.forEach((cell, i) => {
        const len = String(cell ?? '').length
        colWidths[i] = Math.max(colWidths[i] || 8, Math.min(len + 2, 36))
      })
    }
    ws['!cols'] = colWidths.map((wch) => ({ wch }))
    const safeName = s.name.replace(/[\\/?*[\]]/g, '_').slice(0, 31) || 'Sheet1'
    XLSX.utils.book_append_sheet(wb, ws, safeName)
  }
  const name = filename.endsWith('.xlsx') ? filename : `${filename}.xlsx`
  XLSX.writeFile(wb, name)
}
