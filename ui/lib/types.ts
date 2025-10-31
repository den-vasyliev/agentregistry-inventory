// Type definitions based on the MCP Registry API response

export interface Registry {
  id: number
  name: string
  url: string
  type: string
  connected: boolean
}

export interface Variable {
  choices?: string[]
  default?: string
  description?: string
  format?: string
  isRequired?: boolean
  isSecret?: boolean
  placeholder?: string
  value?: string
}

export interface EnvironmentVariable extends Variable {
  name: string
  variables?: Record<string, Variable>
}

export interface Argument extends Variable {
  name: string
  type?: "positional" | "flag"
  isRepeated?: boolean
  valueHint?: string
  variables?: Record<string, Variable>
}

export interface Transport {
  type: "stdio" | "http" | "sse"
  url?: string
  headers?: EnvironmentVariable[]
}

export interface Package {
  identifier: string
  version: string
  registryType: "npm" | "pypi" | "docker"
  registryBaseUrl?: string
  runtimeHint?: string
  environmentVariables?: EnvironmentVariable[]
  packageArguments?: Argument[]
  runtimeArguments?: Argument[]
  transport?: Transport
  fileSha256?: string
}

export interface Remote {
  type: string
  url?: string
  headers?: EnvironmentVariable[]
}

export interface Repository {
  id?: string
  source: "github" | "gitlab" | "bitbucket"
  url: string
  subfolder?: string
}

export interface Icon {
  src: string
  mimeType: string
  sizes?: string[]
  theme?: "light" | "dark"
}

export interface ServerMeta {
  "io.modelcontextprotocol.registry/official"?: {
    isLatest: boolean
    publishedAt: string
    status: "active" | "deprecated" | "archived"
    updatedAt: string
  }
}

export interface Server {
  $schema?: string
  name: string
  title?: string
  description: string
  version: string
  icons?: Icon[]
  packages?: Package[]
  remotes?: Remote[]
  repository?: Repository
  websiteUrl?: string
  _meta?: {
    "io.modelcontextprotocol.registry/publisher-provided"?: Record<string, any>
  }
}

export interface MCPServerListItem {
  _meta?: ServerMeta
  server: Server
}

export interface MCPServerResponse {
  metadata: {
    count: number
    nextCursor?: string
  }
  servers: MCPServerListItem[]
}

// Extended types for local state
export interface MCPServerWithStatus extends MCPServerListItem {
  installed: boolean
  installedAt?: string
  configuredVariables?: Record<string, string>
  _dbId?: number // Internal database ID for API calls
}

// Grouped server type for displaying multiple versions
export interface GroupedMCPServer {
  name: string
  title?: string
  description: string
  icon?: string
  websiteUrl?: string
  repository?: Repository
  versions: MCPServerWithStatus[]
  latestVersion: MCPServerWithStatus
  hasInstalledVersion: boolean
}

// Tool definition for playground
export interface MCPTool {
  name: string
  description: string
  inputSchema: Record<string, any>
}

// Score metrics
export interface ServerScore {
  overall: number
  security: number
  reliability: number
  performance: number
  documentation: number
  community: number
  maintenance: number
}

