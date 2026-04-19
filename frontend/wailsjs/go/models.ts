export namespace main {
	
	export class ReadSkillFileResult {
	    content: string;
	    size: number;
	    language: string;
	
	    static createFrom(source: any = {}) {
	        return new ReadSkillFileResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content = source["content"];
	        this.size = source["size"];
	        this.language = source["language"];
	    }
	}
	export class UpdateInfo {
	    available: boolean;
	    currentVersion: string;
	    latestVersion: string;
	    downloadURL: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.downloadURL = source["downloadURL"];
	    }
	}

}

export namespace skills {
	
	export class AuxFile {
	    path: string;
	    relPath: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new AuxFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.relPath = source["relPath"];
	        this.type = source["type"];
	    }
	}
	export class Collection {
	    id: string;
	    name: string;
	    skillPaths: string[];
	    created: number;
	    modified: number;
	
	    static createFrom(source: any = {}) {
	        return new Collection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.skillPaths = source["skillPaths"];
	        this.created = source["created"];
	        this.modified = source["modified"];
	    }
	}
	export class DisabledSkillInfo {
	    originalPath: string;
	    tool: string;
	    disabledAt: number;
	    name: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new DisabledSkillInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.originalPath = source["originalPath"];
	        this.tool = source["tool"];
	        this.disabledAt = source["disabledAt"];
	        this.name = source["name"];
	        this.size = source["size"];
	    }
	}
	export class MCPServer {
	    name: string;
	    command: string;
	    args: string[];
	    env?: Record<string, string>;
	    url?: string;
	    tool: string;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPServer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.command = source["command"];
	        this.args = source["args"];
	        this.env = source["env"];
	        this.url = source["url"];
	        this.tool = source["tool"];
	        this.source = source["source"];
	    }
	}
	export class MCPToolConfig {
	    tool: string;
	    label: string;
	    path: string;
	    exists: boolean;
	    servers: MCPServer[];
	
	    static createFrom(source: any = {}) {
	        return new MCPToolConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = source["tool"];
	        this.label = source["label"];
	        this.path = source["path"];
	        this.exists = source["exists"];
	        this.servers = this.convertValues(source["servers"], MCPServer);
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
	export class MarketplaceSkill {
	    name: string;
	    path: string;
	    downloadUrl: string;
	    size: number;
	    source: string;
	    tool: string;
	
	    static createFrom(source: any = {}) {
	        return new MarketplaceSkill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.downloadUrl = source["downloadUrl"];
	        this.size = source["size"];
	        this.source = source["source"];
	        this.tool = source["tool"];
	    }
	}
	export class MarketplaceSource {
	    id: string;
	    name: string;
	    repo: string;
	    branch: string;
	    basePath: string;
	    tool: string;
	
	    static createFrom(source: any = {}) {
	        return new MarketplaceSource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.repo = source["repo"];
	        this.branch = source["branch"];
	        this.basePath = source["basePath"];
	        this.tool = source["tool"];
	    }
	}
	export class PluginInfo {
	    name: string;
	    marketplace: string;
	    version: string;
	    scope: string;
	    installPath: string;
	    installedAt: string;
	    lastUpdated: string;
	
	    static createFrom(source: any = {}) {
	        return new PluginInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.marketplace = source["marketplace"];
	        this.version = source["version"];
	        this.scope = source["scope"];
	        this.installPath = source["installPath"];
	        this.installedAt = source["installedAt"];
	        this.lastUpdated = source["lastUpdated"];
	    }
	}
	export class PluginToolConfig {
	    tool: string;
	    label: string;
	    available: boolean;
	    plugins: PluginInfo[];
	
	    static createFrom(source: any = {}) {
	        return new PluginToolConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = source["tool"];
	        this.label = source["label"];
	        this.available = source["available"];
	        this.plugins = this.convertValues(source["plugins"], PluginInfo);
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
	export class Project {
	    id: string;
	    path: string;
	    name: string;
	    addedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.path = source["path"];
	        this.name = source["name"];
	        this.addedAt = source["addedAt"];
	    }
	}
	export class RecentFile {
	    path: string;
	    name: string;
	    tool: string;
	    openedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new RecentFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.tool = source["tool"];
	        this.openedAt = source["openedAt"];
	    }
	}
	export class RecentStore {
	    files: RecentFile[];
	
	    static createFrom(source: any = {}) {
	        return new RecentStore(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.files = this.convertValues(source["files"], RecentFile);
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
	export class SearchResult {
	    path: string;
	    name: string;
	    tool: string;
	    line: number;
	    content: string;
	    match: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.tool = source["tool"];
	        this.line = source["line"];
	        this.content = source["content"];
	        this.match = source["match"];
	    }
	}
	export class Settings {
	    customPaths: Record<string, Array<string>>;
	    theme: string;
	    scanOnStartup: boolean;
	    runInBackground: boolean;
	    lastUpdateCheck: number;
	    dismissedVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.customPaths = source["customPaths"];
	        this.theme = source["theme"];
	        this.scanOnStartup = source["scanOnStartup"];
	        this.runInBackground = source["runInBackground"];
	        this.lastUpdateCheck = source["lastUpdateCheck"];
	        this.dismissedVersion = source["dismissedVersion"];
	    }
	}
	export class Skill {
	    id: string;
	    name: string;
	    path: string;
	    realPath: string;
	    tool: string;
	    tools: string[];
	    category: string;
	    content: string;
	    frontmatter: Record<string, string>;
	    modified: number;
	    size: number;
	    directory: string;
	    auxiliaryFiles: AuxFile[];
	
	    static createFrom(source: any = {}) {
	        return new Skill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.realPath = source["realPath"];
	        this.tool = source["tool"];
	        this.tools = source["tools"];
	        this.category = source["category"];
	        this.content = source["content"];
	        this.frontmatter = source["frontmatter"];
	        this.modified = source["modified"];
	        this.size = source["size"];
	        this.directory = source["directory"];
	        this.auxiliaryFiles = this.convertValues(source["auxiliaryFiles"], AuxFile);
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
	export class SkillTemplate {
	    id: string;
	    name: string;
	    description: string;
	    tool: string;
	    category: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillTemplate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.tool = source["tool"];
	        this.category = source["category"];
	        this.content = source["content"];
	    }
	}
	export class ToolMeta {
	    id: string;
	    label: string;
	    color: string;
	    icon: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.color = source["color"];
	        this.icon = source["icon"];
	    }
	}

}

export namespace sync {
	
	export class PeerInfo {
	    id: string;
	    humanId: string;
	    name: string;
	    pairedAt: number;
	    lastSeen: number;
	    connected: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PeerInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.humanId = source["humanId"];
	        this.name = source["name"];
	        this.pairedAt = source["pairedAt"];
	        this.lastSeen = source["lastSeen"];
	        this.connected = source["connected"];
	    }
	}

}

export namespace team {
	
	export class ManagedFile {
	    path: string;
	    sha256: string;
	    version: number;
	    tool: string;
	
	    static createFrom(source: any = {}) {
	        return new ManagedFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.sha256 = source["sha256"];
	        this.version = source["version"];
	        this.tool = source["tool"];
	    }
	}
	export class TeamState {
	    server_url: string;
	    token: string;
	    org_name: string;
	    org_id: string;
	    connected: boolean;
	    // Go type: time
	    last_sync: any;
	    managed_files: Record<string, ManagedFile>;
	
	    static createFrom(source: any = {}) {
	        return new TeamState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server_url = source["server_url"];
	        this.token = source["token"];
	        this.org_name = source["org_name"];
	        this.org_id = source["org_id"];
	        this.connected = source["connected"];
	        this.last_sync = this.convertValues(source["last_sync"], null);
	        this.managed_files = this.convertValues(source["managed_files"], ManagedFile, true);
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

}

