export namespace database {
	
	export class ColumnInfo {
	    name: string;
	    position: number;
	    default?: string;
	    nullable: boolean;
	    dataType: string;
	    columnType: string;
	    maxLength?: number;
	    charSet?: string;
	    collation?: string;
	    key: string;
	    extra: string;
	    comment: string;
	
	    static createFrom(source: any = {}) {
	        return new ColumnInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.position = source["position"];
	        this.default = source["default"];
	        this.nullable = source["nullable"];
	        this.dataType = source["dataType"];
	        this.columnType = source["columnType"];
	        this.maxLength = source["maxLength"];
	        this.charSet = source["charSet"];
	        this.collation = source["collation"];
	        this.key = source["key"];
	        this.extra = source["extra"];
	        this.comment = source["comment"];
	    }
	}
	export class ColumnMapping {
	    csvIndex: number;
	    columnName: string;
	
	    static createFrom(source: any = {}) {
	        return new ColumnMapping(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.csvIndex = source["csvIndex"];
	        this.columnName = source["columnName"];
	    }
	}
	export class DatabaseInfo {
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new DatabaseInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	    }
	}
	export class ForeignKeyInfo {
	    name: string;
	    column: string;
	    refTable: string;
	    refColumn: string;
	    updateRule: string;
	    deleteRule: string;
	
	    static createFrom(source: any = {}) {
	        return new ForeignKeyInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.column = source["column"];
	        this.refTable = source["refTable"];
	        this.refColumn = source["refColumn"];
	        this.updateRule = source["updateRule"];
	        this.deleteRule = source["deleteRule"];
	    }
	}
	export class IndexInfo {
	    name: string;
	    columns: string;
	    unique: boolean;
	    type: string;
	    comment: string;
	
	    static createFrom(source: any = {}) {
	        return new IndexInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.columns = source["columns"];
	        this.unique = source["unique"];
	        this.type = source["type"];
	        this.comment = source["comment"];
	    }
	}
	export class QueryResult {
	    columns: string[];
	    rows: string[][];
	    rowCount: number;
	    affectedRows: number;
	    duration: string;
	    isSelect: boolean;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new QueryResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.columns = source["columns"];
	        this.rows = source["rows"];
	        this.rowCount = source["rowCount"];
	        this.affectedRows = source["affectedRows"];
	        this.duration = source["duration"];
	        this.isSelect = source["isSelect"];
	        this.error = source["error"];
	    }
	}
	export class RoutineInfo {
	    name: string;
	    type: string;
	    created: string;
	
	    static createFrom(source: any = {}) {
	        return new RoutineInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.created = source["created"];
	    }
	}
	export class TableDetail {
	    columns: ColumnInfo[];
	    indexes: IndexInfo[];
	    foreignKeys: ForeignKeyInfo[];
	    createSql: string;
	
	    static createFrom(source: any = {}) {
	        return new TableDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.columns = this.convertValues(source["columns"], ColumnInfo);
	        this.indexes = this.convertValues(source["indexes"], IndexInfo);
	        this.foreignKeys = this.convertValues(source["foreignKeys"], ForeignKeyInfo);
	        this.createSql = source["createSql"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TableInfo {
	    name: string;
	    type: string;
	    engine: string;
	    rowCount: number;
	    dataSize: number;
	    collation: string;
	
	    static createFrom(source: any = {}) {
	        return new TableInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.engine = source["engine"];
	        this.rowCount = source["rowCount"];
	        this.dataSize = source["dataSize"];
	        this.collation = source["collation"];
	    }
	}
	export class TriggerInfo {
	    name: string;
	    event: string;
	    timing: string;
	    table: string;
	    statement: string;
	
	    static createFrom(source: any = {}) {
	        return new TriggerInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.event = source["event"];
	        this.timing = source["timing"];
	        this.table = source["table"];
	        this.statement = source["statement"];
	    }
	}
	export class UserDetail {
	    user: string;
	    host: string;
	    plugin: string;
	    grants: string[];
	
	    static createFrom(source: any = {}) {
	        return new UserDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user = source["user"];
	        this.host = source["host"];
	        this.plugin = source["plugin"];
	        this.grants = source["grants"];
	    }
	}
	export class UserInfo {
	    user: string;
	    host: string;
	    plugin: string;
	
	    static createFrom(source: any = {}) {
	        return new UserInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user = source["user"];
	        this.host = source["host"];
	        this.plugin = source["plugin"];
	    }
	}

}

export namespace main {
	
	export class CSVImportPreview {
	    filePath: string;
	    headers: string[];
	    sampleRows: string[][];
	    totalRows: number;
	
	    static createFrom(source: any = {}) {
	        return new CSVImportPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.headers = source["headers"];
	        this.sampleRows = source["sampleRows"];
	        this.totalRows = source["totalRows"];
	    }
	}
	export class ConnectionProfile {
	    id: string;
	    name: string;
	    host: string;
	    port: number;
	    username: string;
	    password: string;
	    defaultDb: string;
	    useSsl: boolean;
	    sshEnabled: boolean;
	    sshHost: string;
	    sshPort: number;
	    sshUser: string;
	    sshAuth: string;
	    sshKeyPath: string;
	    sshPassword: string;
	    sortOrder: number;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.defaultDb = source["defaultDb"];
	        this.useSsl = source["useSsl"];
	        this.sshEnabled = source["sshEnabled"];
	        this.sshHost = source["sshHost"];
	        this.sshPort = source["sshPort"];
	        this.sshUser = source["sshUser"];
	        this.sshAuth = source["sshAuth"];
	        this.sshKeyPath = source["sshKeyPath"];
	        this.sshPassword = source["sshPassword"];
	        this.sortOrder = source["sortOrder"];
	    }
	}

}

