export type ToolType =
  | "claude-code" | "gemini-cli" | "cursor" | "kiro"
  | "windsurf" | "copilot" | "aider" | "amp" | "codex" | "unknown";

export interface ToolMeta {
  id: ToolType;
  label: string;
  color: string;
  icon: string;
}

export type Category = "agent" | "skill" | "command" | "rule" | "config";

export const categoryLabels: Record<Category, string> = {
  agent: "Agents",
  skill: "Skills",
  command: "Commands",
  rule: "Rules",
  config: "Config",
};

export const categoryIcons: Record<Category, string> = {
  agent: "\u{1F916}",
  skill: "\u26A1",
  command: "\u{1F4AC}",
  rule: "\u{1F4CB}",
  config: "\u2699\uFE0F",
};

export interface AuxFile {
  path: string;
  relPath: string;
  type: "markdown" | "script" | "code" | "other";
}

export interface Skill {
  id: string;
  name: string;
  path: string;
  realPath: string;
  tool: ToolType;
  tools: ToolType[];
  category: Category;
  content: string;
  frontmatter: Record<string, string>;
  modified: number;
  size: number;
  directory?: string;
  auxiliaryFiles?: AuxFile[];
}

export interface Collection {
  id: string;
  name: string;
  skillPaths: string[];
  created: number;
  modified: number;
}

export interface Settings {
  customPaths: Record<string, string[]>;
  theme: string;
  scanOnStartup: boolean;
  runInBackground: boolean;
  lastUpdateCheck: number;
  dismissedVersion: string;
}

export interface UpdateInfo {
  available: boolean;
  currentVersion: string;
  latestVersion: string;
  downloadURL: string;
}

export interface MarketplaceSource {
  id: string;
  name: string;
  repo: string;
  branch: string;
  basePath: string;
  tool: ToolType;
}

export interface MarketplaceSkill {
  name: string;
  path: string;
  downloadUrl: string;
  size: number;
  source: string;
  tool: string;
}

export function getToolMeta(tool: ToolType, tools: ToolMeta[]): ToolMeta {
  return tools.find((t) => t.id === tool) ?? { id: "unknown", label: "Unknown", color: "#6b7280", icon: "\u25cb" };
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function formatDate(ts: number): string {
  if (!ts) return "\u2014";
  return new Date(ts * 1000).toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
}

export interface MCPServer {
  name: string;
  command: string;
  args: string[];
  env?: Record<string, string>;
  url?: string;
  tool: string;
  source: string;
}

export interface MCPToolConfig {
  tool: string;
  label: string;
  path: string;
  exists: boolean;
  servers: MCPServer[];
}

export interface PluginInfo {
  name: string;
  marketplace: string;
  version: string;
  scope: string;
  installPath: string;
  installedAt: string;
  lastUpdated: string;
}

export interface PluginToolConfig {
  tool: string;
  label: string;
  available: boolean;
  plugins: PluginInfo[];
}

export interface SyncStatus {
  peerId: string;
  humanId: string;
  totalPeers: number;
  connectedPeers: number;
  trackedFiles: number;
  running: boolean;
}

export interface TeamManagedFile {
  path: string;
  sha256: string;
  version: number;
  tool: string;
}

export interface TeamState {
  server_url: string;
  org_name: string;
  org_id: string;
  connected: boolean;
  last_sync: string;
  managed_files: Record<string, TeamManagedFile>;
}

export interface SyncPeer {
  id: string;
  humanId: string;
  name: string;
  pairedAt: number;
  lastSeen: number;
  connected: boolean;
}

export interface SyncManifest {
  mode: string;
  syncedPaths: string[];
}

export interface SyncConflict {
  path: string;
  localHash: string;
  remoteHash: string;
  remotePeer: string;
  detectedAt: number;
  localContent: string;
  remoteContent: string;
}

export interface SearchResult {
  path: string;
  name: string;
  tool: string;
  line: number;
  content: string;
  match: string;
}

export interface ValidationWarning {
  path: string;
  type: string;
  message: string;
}

export interface SkillTemplate {
  id: string;
  name: string;
  description: string;
  tool: string;
  category: string;
  content: string;
}

export interface RecentFile {
  path: string;
  name: string;
  tool: string;
  openedAt: number;
}

export interface DisabledSkillInfo {
  originalPath: string;
  tool: ToolType;
  disabledAt: number;
  name: string;
  size: number;
}

export interface Project {
  id: string;
  path: string;
  name: string;
  addedAt: number;
}
