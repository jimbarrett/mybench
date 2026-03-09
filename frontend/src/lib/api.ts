const API = '/api'

async function request(url: string, options?: RequestInit) {
  const res = await fetch(url, options)
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error || res.statusText)
  }
  return res.json()
}

function post(url: string, body?: any) {
  return request(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
}

function put(url: string, body?: any) {
  return request(url, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
}

function del(url: string) {
  return request(url, { method: 'DELETE' })
}

// --- Vault / Auth ---

export async function getVaultStatus(): Promise<{ hasMasterPassword: boolean; isUnlocked: boolean }> {
  return request(`${API}/vault/status`)
}

export async function createVault(password: string): Promise<{ ok: boolean }> {
  return post(`${API}/vault/create`, { password })
}

export async function unlockVault(password: string): Promise<{ ok: boolean }> {
  return post(`${API}/vault/unlock`, { password })
}

// --- Connections ---

export async function listConnections(): Promise<any[]> {
  return request(`${API}/connections`)
}

export async function saveConnection(conn: any): Promise<{ id: string }> {
  if (conn.id) {
    return put(`${API}/connections/${conn.id}`, conn)
  }
  return post(`${API}/connections`, conn)
}

export async function deleteConnection(id: string): Promise<void> {
  return del(`${API}/connections/${id}`)
}

export async function testConnection(conn: any): Promise<{ ok: boolean }> {
  return post(`${API}/connections/${conn.id || '_'}/test`, conn)
}

// --- Tabs / Active Connections ---

export async function connect(tabId: string, profileId: string): Promise<void> {
  return post(`${API}/tabs/${tabId}/connect`, { profileId })
}

export async function disconnect(tabId: string): Promise<void> {
  return post(`${API}/tabs/${tabId}/disconnect`)
}

export async function pingConnection(tabId: string): Promise<void> {
  return request(`${API}/tabs/${tabId}/ping`)
}

// --- Schema ---

export async function getDatabases(tabId: string): Promise<any[]> {
  return request(`${API}/tabs/${tabId}/databases`)
}

export async function getTables(tabId: string, db: string): Promise<any[]> {
  return request(`${API}/tabs/${tabId}/databases/${db}/tables`)
}

export async function getTableDetail(tabId: string, db: string, table: string): Promise<any> {
  return request(`${API}/tabs/${tabId}/databases/${db}/tables/${table}`)
}

export async function getTableColumns(tabId: string, db: string, table: string): Promise<string[]> {
  return request(`${API}/tabs/${tabId}/databases/${db}/tables/${table}/columns`)
}

export async function getRoutines(tabId: string, db: string): Promise<any[]> {
  return request(`${API}/tabs/${tabId}/databases/${db}/routines`)
}

export async function getTriggers(tabId: string, db: string): Promise<any[]> {
  return request(`${API}/tabs/${tabId}/databases/${db}/triggers`)
}

export async function getSchemaCompletions(tabId: string): Promise<Record<string, string[]>> {
  return request(`${API}/tabs/${tabId}/completions`)
}

// --- Queries ---

export async function executeQuery(tabId: string, sql: string): Promise<any[]> {
  return post(`${API}/tabs/${tabId}/query`, { sql })
}

export async function explainQuery(tabId: string, sql: string): Promise<any> {
  return post(`${API}/tabs/${tabId}/explain`, { sql })
}

export async function cancelQuery(tabId: string): Promise<void> {
  return post(`${API}/tabs/${tabId}/cancel`)
}

// --- Users ---

export async function listUsers(tabId: string): Promise<any[]> {
  return request(`${API}/tabs/${tabId}/users`)
}

export async function getUserDetail(tabId: string, user: string, host: string): Promise<any> {
  return request(`${API}/tabs/${tabId}/users/${user}/${host}`)
}

export async function createUser(tabId: string, user: string, host: string, password: string, plugin: string): Promise<void> {
  return post(`${API}/tabs/${tabId}/users`, { user, host, password, plugin })
}

export async function dropUser(tabId: string, user: string, host: string): Promise<void> {
  return del(`${API}/tabs/${tabId}/users/${user}/${host}`)
}

export async function changeUserPassword(tabId: string, user: string, host: string, password: string): Promise<void> {
  return put(`${API}/tabs/${tabId}/users/${user}/${host}/password`, { password })
}

export async function grantPrivileges(tabId: string, user: string, host: string, privileges: string, on: string): Promise<void> {
  return post(`${API}/tabs/${tabId}/users/${user}/${host}/grant`, { privileges, on })
}

export async function revokePrivileges(tabId: string, user: string, host: string, privileges: string, on: string): Promise<void> {
  return post(`${API}/tabs/${tabId}/users/${user}/${host}/revoke`, { privileges, on })
}

// --- Export ---

export function exportTableCSV(tabId: string, db: string, table: string): void {
  triggerDownload(`${API}/tabs/${tabId}/export/csv?db=${encodeURIComponent(db)}&table=${encodeURIComponent(table)}`)
}

export function exportTableSQL(tabId: string, db: string, table: string): void {
  triggerDownload(`${API}/tabs/${tabId}/export/sql?db=${encodeURIComponent(db)}&table=${encodeURIComponent(table)}`)
}

export async function exportResultsCSV(columns: string[], rows: string[][]): Promise<void> {
  const res = await fetch(`${API}/tabs/_/export/results/csv`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ columns, rows }),
  })
  await downloadBlob(res, 'results.csv')
}

export async function exportResultsSQL(tableName: string, columns: string[], rows: string[][]): Promise<void> {
  const res = await fetch(`${API}/tabs/_/export/results/sql`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ tableName, columns, rows }),
  })
  await downloadBlob(res, `${tableName}.sql`)
}

// --- Import ---

export async function importCSVPreview(tabId: string, file: File): Promise<any> {
  const form = new FormData()
  form.append('file', file)
  const res = await fetch(`${API}/tabs/${tabId}/import/csv/preview`, {
    method: 'POST',
    body: form,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error || res.statusText)
  }
  return res.json()
}

export async function importCSV(
  tabId: string,
  db: string,
  table: string,
  filePath: string,
  mappings: { csvIndex: number; columnName: string }[],
): Promise<{ rows: number; error?: string }> {
  const form = new FormData()
  form.append('filePath', filePath)
  form.append('db', db)
  form.append('table', table)
  form.append('mappings', JSON.stringify(mappings))
  const res = await fetch(`${API}/tabs/${tabId}/import/csv`, {
    method: 'POST',
    body: form,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error || res.statusText)
  }
  return res.json()
}

export async function importSQL(tabId: string, file: File): Promise<{ statements: number; error?: string }> {
  const form = new FormData()
  form.append('file', file)
  const res = await fetch(`${API}/tabs/${tabId}/import/sql`, {
    method: 'POST',
    body: form,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error || res.statusText)
  }
  return res.json()
}

export async function cancelImportExport(tabId: string): Promise<void> {
  return post(`${API}/tabs/${tabId}/import-export/cancel`)
}

// --- Helpers ---

function triggerDownload(url: string) {
  const a = document.createElement('a')
  a.href = url
  a.download = ''
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

async function downloadBlob(res: Response, filename: string) {
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
