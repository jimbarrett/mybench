// Types matching the Go backend structs.
// These replace the auto-generated Wails models.

export interface ConnectionProfile {
  id: string
  name: string
  host: string
  port: number
  username: string
  password: string
  defaultDb: string
  useSsl: boolean
  sshEnabled: boolean
  sshHost: string
  sshPort: number
  sshUser: string
  sshAuth: string
  sshKeyPath: string
  sshPassword: string
  sortOrder: number
}

export interface DatabaseInfo {
  name: string
}

export interface TableInfo {
  name: string
  type: string
  engine: string
  rowCount: number
  dataSize: number
  collation: string
}

export interface ColumnInfo {
  name: string
  position: number
  default: string | null
  nullable: boolean
  dataType: string
  columnType: string
  maxLength: number | null
  charSet: string | null
  collation: string | null
  key: string
  extra: string
  comment: string
}

export interface IndexInfo {
  name: string
  columns: string
  unique: boolean
  type: string
  comment: string
}

export interface ForeignKeyInfo {
  name: string
  column: string
  refTable: string
  refColumn: string
  updateRule: string
  deleteRule: string
}

export interface TableDetail {
  columns: ColumnInfo[]
  indexes: IndexInfo[]
  foreignKeys: ForeignKeyInfo[]
  createSql: string
}

export interface RoutineInfo {
  name: string
  type: string
  created: string
}

export interface TriggerInfo {
  name: string
  event: string
  timing: string
  table: string
  statement: string
}

export interface QueryResult {
  columns: string[]
  rows: string[][]
  rowCount: number
  affectedRows: number
  duration: string
  isSelect: boolean
  error: string
}

export interface UserInfo {
  user: string
  host: string
  plugin: string
}

export interface UserDetail {
  user: string
  host: string
  plugin: string
  grants: string[]
}

export interface ColumnMapping {
  csvIndex: number
  columnName: string
}

export interface CSVImportPreview {
  filePath: string
  headers: string[]
  sampleRows: string[][]
  totalRows: number
}

export function newConnectionProfile(data?: Partial<ConnectionProfile>): ConnectionProfile {
  return {
    id: '',
    name: '',
    host: '127.0.0.1',
    port: 3306,
    username: 'root',
    password: '',
    defaultDb: '',
    useSsl: false,
    sshEnabled: false,
    sshHost: '',
    sshPort: 22,
    sshUser: '',
    sshAuth: 'key',
    sshKeyPath: '',
    sshPassword: '',
    sortOrder: 0,
    ...data,
  }
}
