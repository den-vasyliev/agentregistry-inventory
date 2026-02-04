// Admin API client for the registry management UI
// This client communicates with the /admin/v0 API endpoints

// In development mode with Next.js dev server, use relative URL to leverage proxy
// In production (static export), API_BASE_URL is set via environment variable or defaults to current origin
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || (typeof window !== 'undefined' && window.location.origin) || ''

// Retry configuration
const DEFAULT_RETRIES = 3
const DEFAULT_RETRY_DELAY = 1000 // ms
const MAX_RETRY_DELAY = 10000 // ms

// HTTP status codes that should trigger a retry
const RETRYABLE_STATUS_CODES = [408, 429, 500, 502, 503, 504]

// Helper function to delay execution
const delay = (ms: number) => new Promise(resolve => setTimeout(resolve, ms))

// Calculate backoff with jitter
const calculateBackoff = (attempt: number, baseDelay: number): number => {
  const exponentialDelay = baseDelay * Math.pow(2, attempt)
  const jitter = Math.random() * 1000 // Add up to 1s of jitter
  return Math.min(exponentialDelay + jitter, MAX_RETRY_DELAY)
}

// MCP Server types based on the official spec
export interface ServerJSON {
  $schema?: string
  name: string
  title?: string
  description: string
  version: string
  icons?: Array<{
    src: string
    mimeType: string
    sizes?: string[]
    theme?: 'light' | 'dark'
  }>
  packages?: Array<{
    identifier: string
    version: string
    registryType: 'npm' | 'pypi' | 'docker'
    transport?: {
      type: string
      url?: string
    }
  }>
  remotes?: Array<{
    type: string
    url?: string
  }>
  repository?: {
    source: 'github' | 'gitlab' | 'bitbucket'
    url: string
  }
  websiteUrl?: string
  _meta?: {
    'io.modelcontextprotocol.registry/publisher-provided'?: {
      'aregistry.ai/metadata'?: {
        stars?: number
        score?: number
        scorecard?: {
          openssf?: number
        }
        repo?: {
          forks_count?: number
          watchers_count?: number
          primary_language?: string
          tags?: string[]
          topics?: string[]
        }
        endpoint_health?: {
          last_checked_at?: string
          reachable?: boolean
          response_ms?: number
        }
        scans?: {
          container_images?: unknown[]
          dependency_health?: {
            copyleft_licenses?: number
            ecosystems?: Record<string, number>
            packages_total?: number
            unknown_licenses?: number
          }
          details?: string[]
          summary?: string
        }
        activity?: {
          created_at?: string
          pushed_at?: string
          updated_at?: string
        }
        identity?: {
          org_is_verified?: boolean
          publisher_identity_verified_by_jwt?: boolean
        }
        semver?: {
          uses_semver?: boolean
        }
        security_scanning?: {
          code_scanning_alerts?: number | null
          codeql_enabled?: boolean
          dependabot_alerts?: number | null
          dependabot_enabled?: boolean
        }
        downloads?: {
          total?: number
        }
        releases?: {
          latest_published_at?: string | null
        }
      }
    }
  }
}

export interface RegistryExtensions {
  status: 'active' | 'deprecated' | 'deleted' | 'pending_review'
  publishedAt: string
  updatedAt: string
  isLatest: boolean
  reviewStatus?: 'pending' | 'approved' | 'rejected'
}

export interface DeploymentInfo {
  namespace?: string
  serviceName?: string
  url?: string
  ready: boolean
  message?: string
  lastChecked?: string
}

export interface ServerResponse {
  server: ServerJSON
  _meta: {
    'io.modelcontextprotocol.registry/official'?: RegistryExtensions
    deployment?: DeploymentInfo
    source?: string // discovery, manual, deployment
    isDiscovered?: boolean
  }
}

export interface ServerListResponse {
  servers: ServerResponse[]
  metadata: {
    count: number
    nextCursor?: string
  }
}

export interface ImportRequest {
  source: string
  headers?: Record<string, string>
  update?: boolean
  skip_validation?: boolean
}

export interface ImportResponse {
  success: boolean
  message: string
}

export interface SubmitRequest {
  repositoryUrl: string
}

export interface SubmitResponse {
  success: boolean
  message: string
  name?: string
  kind?: string
  version?: string
  status?: string
}

export interface ServerStats {
  total_servers: number
  total_server_names: number
  active_servers: number
  deprecated_servers: number
  deleted_servers: number
}

// Skill types
export interface SkillRepository {
  url: string
  source: string
}

export interface SkillPackageInfo {
  registryType: string
  identifier: string
  version: string
  transport: {
    type: string
  }
}

export interface SkillRemoteInfo {
  url: string
}

export interface SkillJSON {
  name: string
  title?: string
  description: string
  version: string
  status?: string
  websiteUrl?: string
  repository?: SkillRepository
  packages?: SkillPackageInfo[]
  remotes?: SkillRemoteInfo[]
  metadata?: Record<string, unknown>
}

export interface SkillRegistryExtensions {
  status: string
  publishedAt: string
  updatedAt: string
  isLatest: boolean
}

export interface SkillResponse {
  skill: SkillJSON
  _meta: {
    'io.modelcontextprotocol.registry/official'?: SkillRegistryExtensions
    'io.modelcontextprotocol.registry/publisher-provided'?: {
      'aregistry.ai/metadata'?: {
        identity?: {
          org_is_verified?: boolean
          publisher_identity_verified_by_jwt?: boolean
        }
      }
    }
  }
}

export interface SkillListResponse {
  skills: SkillResponse[]
  metadata: {
    count: number
    nextCursor?: string
  }
}

// Agent types
export interface AgentJSON {
  name: string
  image: string
  language: string
  framework: string
  modelProvider: string
  modelName: string
  description: string
  updatedAt: string
  version: string
  status: string
  repository?: {
    url: string
    source: string
  }
}

export interface AgentRegistryExtensions {
  status: string
  publishedAt: string
  updatedAt: string
  isLatest: boolean
  published?: boolean
}

export interface AgentResponse {
  agent: AgentJSON
  _meta: {
    'io.modelcontextprotocol.registry/official'?: AgentRegistryExtensions
    deployment?: DeploymentInfo
    source?: string // discovery, manual, deployment
    isDiscovered?: boolean
  }
}

export interface AgentListResponse {
  agents: AgentResponse[]
  metadata: {
    count: number
    nextCursor?: string
  }
}

// Model types
export interface ModelJSON {
  name: string
  provider: string
  model: string
  baseUrl?: string
  description?: string
}

export interface ModelUsageRefJSON {
  namespace: string
  name: string
  kind?: string
}

export interface ModelRegistryExtensions {
  status: string
  publishedAt?: string
  updatedAt: string
  isLatest: boolean
  published?: boolean
}

export interface ModelResponse {
  model: ModelJSON
  _meta: {
    'io.modelcontextprotocol.registry/official'?: ModelRegistryExtensions
    usedBy?: ModelUsageRefJSON[]
    ready?: boolean
    message?: string
    deployment?: DeploymentInfo
    isDiscovered?: boolean
  }
}

export interface ModelListResponse {
  models: ModelResponse[]
  metadata: {
    count: number
    nextCursor?: string
  }
}

class AdminApiClient {
  private baseUrl: string
  private getAuthToken?: () => string | null
  private retries: number
  private retryDelay: number

  constructor(
    baseUrl: string = API_BASE_URL, 
    getAuthToken?: () => string | null,
    retries: number = DEFAULT_RETRIES,
    retryDelay: number = DEFAULT_RETRY_DELAY
  ) {
    this.baseUrl = baseUrl
    this.getAuthToken = getAuthToken
    this.retries = retries
    this.retryDelay = retryDelay
  }

  private getHeaders(): HeadersInit {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    }

    if (this.getAuthToken) {
      const token = this.getAuthToken()
      if (token) {
        headers['Authorization'] = `Bearer ${token}`
      }
    }

    return headers
  }

  // Fetch with retry logic
  private async fetchWithRetry(
    url: string, 
    options: RequestInit, 
    attempt: number = 0
  ): Promise<Response> {
    try {
      const response = await fetch(url, options)
      
      // If response is ok or not retryable, return immediately
      if (response.ok || !RETRYABLE_STATUS_CODES.includes(response.status)) {
        return response
      }
      
      // If we've exhausted retries, return the response
      if (attempt >= this.retries) {
        console.warn(`Fetch failed after ${this.retries} retries: ${url}`)
        return response
      }
      
      // Calculate backoff and retry
      const backoff = calculateBackoff(attempt, this.retryDelay)
      console.warn(`Fetch failed with ${response.status}, retrying in ${backoff}ms (attempt ${attempt + 1}/${this.retries})`)
      await delay(backoff)
      return this.fetchWithRetry(url, options, attempt + 1)
      
    } catch (error) {
      // Network errors should be retried
      if (attempt >= this.retries) {
        console.error(`Fetch failed after ${this.retries} retries:`, error)
        throw error
      }
      
      const backoff = calculateBackoff(attempt, this.retryDelay)
      console.warn(`Network error, retrying in ${backoff}ms (attempt ${attempt + 1}/${this.retries})`)
      await delay(backoff)
      return this.fetchWithRetry(url, options, attempt + 1)
    }
  }

  // List servers with pagination and filtering (ADMIN - shows all servers)
  async listServers(params?: {
    cursor?: string
    limit?: number
    search?: string
    version?: string
    updated_since?: string
  }): Promise<ServerListResponse> {
    const queryParams = new URLSearchParams()
    if (params?.cursor) queryParams.append('cursor', params.cursor)
    if (params?.limit) queryParams.append('limit', params.limit.toString())
    if (params?.search) queryParams.append('search', params.search)
    if (params?.version) queryParams.append('version', params.version)
    if (params?.updated_since) queryParams.append('updated_since', params.updated_since)

    const url = `${this.baseUrl}/admin/v0/servers${queryParams.toString() ? '?' + queryParams.toString() : ''}`
    const response = await this.fetchWithRetry(url, { headers: this.getHeaders() })
    if (!response.ok) {
      throw new Error(`Failed to fetch servers: ${response.status} ${response.statusText}`)
    }
    return response.json()
  }

  // List PUBLISHED servers only (PUBLIC endpoint)
  async listPublishedServers(params?: {
    cursor?: string
    limit?: number
    search?: string
    version?: string
    updated_since?: string
  }): Promise<ServerListResponse> {
    const queryParams = new URLSearchParams()
    if (params?.cursor) queryParams.append('cursor', params.cursor)
    if (params?.limit) queryParams.append('limit', params.limit.toString())
    if (params?.search) queryParams.append('search', params.search)
    if (params?.version) queryParams.append('version', params.version)
    if (params?.updated_since) queryParams.append('updated_since', params.updated_since)

    const url = `${this.baseUrl}/v0/servers${queryParams.toString() ? '?' + queryParams.toString() : ''}`
    const response = await fetch(url)
    if (!response.ok) {
      throw new Error('Failed to fetch published servers')
    }
    return response.json()
  }

  // Get a specific server version
  async getServer(serverName: string, version: string = 'latest'): Promise<ServerResponse> {
    const encodedName = encodeURIComponent(serverName)
    const encodedVersion = encodeURIComponent(version)
    const response = await fetch(`${this.baseUrl}/admin/v0/servers/${encodedName}/versions/${encodedVersion}`)
    if (!response.ok) {
      throw new Error('Failed to fetch server')
    }
    return response.json()
  }

  // Get all versions of a server
  async getServerVersions(serverName: string): Promise<ServerListResponse> {
    const encodedName = encodeURIComponent(serverName)
    const response = await fetch(`${this.baseUrl}/admin/v0/servers/${encodedName}/versions`)
    if (!response.ok) {
      throw new Error('Failed to fetch server versions')
    }
    return response.json()
  }

  // Import servers from an external registry
  async importServers(request: ImportRequest): Promise<ImportResponse> {
    const response = await fetch(`${this.baseUrl}/admin/v0/import`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify(request),
    })
    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.message || 'Failed to import servers')
    }
    return response.json()
  }

  // Create a new server
  async createServer(server: ServerJSON): Promise<ServerResponse> {
    console.log('Creating server:', server)
    const response = await this.fetchWithRetry(`${this.baseUrl}/admin/v0/servers`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify(server),
    })

    // Get response text first so we can parse it or show it as error
    const responseText = await response.text()
    console.log('Response status:', response.status)
    console.log('Response text:', responseText.substring(0, 200))

    if (!response.ok) {
      let errorMessage = 'Failed to create server'
      try {
        const errorData = JSON.parse(responseText)
        errorMessage = errorData.message || errorData.detail || errorData.title || errorMessage
        if (errorData.errors && Array.isArray(errorData.errors)) {
          errorMessage += ': ' + errorData.errors.map((e: unknown) => (typeof e === 'object' && e && 'message' in e ? (e as { message: string }).message : String(e))).join(', ')
        }
      } catch {
        // If JSON parsing fails, use the text directly (truncate if too long)
        errorMessage = responseText.length > 200
          ? responseText.substring(0, 200) + '...'
          : responseText || `Server error: ${response.status} ${response.statusText}`
      }
      throw new Error(errorMessage)
    }

    // Parse successful response
    try {
      return JSON.parse(responseText)
    } catch (e) {
      console.error('Failed to parse response:', e)
      throw new Error(`Invalid response from server: ${responseText.substring(0, 100)}`)
    }
  }

  // Delete a server
  async deleteServer(serverName: string, version: string): Promise<void> {
    const encodedName = encodeURIComponent(serverName)
    const encodedVersion = encodeURIComponent(version)
    const response = await fetch(`${this.baseUrl}/admin/v0/servers/${encodedName}/versions/${encodedVersion}`, {
      method: 'DELETE',
    })
    if (!response.ok) {
      const error = await response.text()
      throw new Error(error || 'Failed to delete server')
    }
  }

  // Delete an agent
  async deleteAgent(agentName: string, version: string): Promise<void> {
    const encodedName = encodeURIComponent(agentName)
    const encodedVersion = encodeURIComponent(version)
    const response = await fetch(`${this.baseUrl}/admin/v0/agents/${encodedName}/versions/${encodedVersion}`, {
      method: 'DELETE',
    })
    if (!response.ok) {
      const error = await response.text()
      throw new Error(error || 'Failed to delete agent')
    }
  }

  // Delete a skill
  async deleteSkill(skillName: string, version: string): Promise<void> {
    const encodedName = encodeURIComponent(skillName)
    const encodedVersion = encodeURIComponent(version)
    const response = await fetch(`${this.baseUrl}/admin/v0/skills/${encodedName}/versions/${encodedVersion}`, {
      method: 'DELETE',
    })
    if (!response.ok) {
      const error = await response.text()
      throw new Error(error || 'Failed to delete skill')
    }
  }

  // Get registry statistics
  async getStats(): Promise<ServerStats> {
    const response = await fetch(`${this.baseUrl}/admin/v0/stats`)
    if (!response.ok) {
      throw new Error('Failed to fetch statistics')
    }
    return response.json()
  }

  // Health check
  async healthCheck(): Promise<{ status: string }> {
    const response = await fetch(`${this.baseUrl}/admin/v0/health`)
    if (!response.ok) {
      throw new Error('Health check failed')
    }
    return response.json()
  }

  // ===== Skills API =====

  // List skills with pagination and filtering (ADMIN - shows all skills)
  async listSkills(params?: {
    cursor?: string
    limit?: number
    search?: string
    version?: string
    updated_since?: string
  }): Promise<SkillListResponse> {
    const queryParams = new URLSearchParams()
    if (params?.cursor) queryParams.append('cursor', params.cursor)
    if (params?.limit) queryParams.append('limit', params.limit.toString())
    if (params?.search) queryParams.append('search', params.search)
    if (params?.version) queryParams.append('version', params.version)
    if (params?.updated_since) queryParams.append('updated_since', params.updated_since)

    const url = `${this.baseUrl}/admin/v0/skills${queryParams.toString() ? '?' + queryParams.toString() : ''}`
    const response = await fetch(url)
    if (!response.ok) {
      throw new Error('Failed to fetch skills')
    }
    return response.json()
  }

  // List PUBLISHED skills only (PUBLIC endpoint)
  async listPublishedSkills(params?: {
    cursor?: string
    limit?: number
    search?: string
    version?: string
    updated_since?: string
  }): Promise<SkillListResponse> {
    const queryParams = new URLSearchParams()
    if (params?.cursor) queryParams.append('cursor', params.cursor)
    if (params?.limit) queryParams.append('limit', params.limit.toString())
    if (params?.search) queryParams.append('search', params.search)
    if (params?.version) queryParams.append('version', params.version)
    if (params?.updated_since) queryParams.append('updated_since', params.updated_since)

    const url = `${this.baseUrl}/v0/skills${queryParams.toString() ? '?' + queryParams.toString() : ''}`
    const response = await fetch(url)
    if (!response.ok) {
      throw new Error('Failed to fetch published skills')
    }
    return response.json()
  }

  // Get a specific skill version
  async getSkill(skillName: string, version: string = 'latest'): Promise<SkillResponse> {
    const encodedName = encodeURIComponent(skillName)
    const encodedVersion = encodeURIComponent(version)
    const response = await fetch(`${this.baseUrl}/admin/v0/skills/${encodedName}/versions/${encodedVersion}`)
    if (!response.ok) {
      throw new Error('Failed to fetch skill')
    }
    return response.json()
  }

  // Get all versions of a skill
  async getSkillVersions(skillName: string): Promise<SkillListResponse> {
    const encodedName = encodeURIComponent(skillName)
    const response = await fetch(`${this.baseUrl}/admin/v0/skills/${encodedName}/versions`)
    if (!response.ok) {
      throw new Error('Failed to fetch skill versions')
    }
    return response.json()
  }

  // Create a new skill
  async createSkill(skill: SkillJSON): Promise<SkillResponse> {
    const response = await fetch(`${this.baseUrl}/admin/v0/skills`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify(skill),
    })
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw new Error(errorData.detail || 'Failed to create skill')
    }
    return response.json()
  }

  // ===== Agents API =====

  // List agents with pagination and filtering (ADMIN - shows all agents)
  async listAgents(params?: {
    cursor?: string
    limit?: number
    search?: string
    version?: string
    updated_since?: string
  }): Promise<AgentListResponse> {
    const queryParams = new URLSearchParams()
    if (params?.cursor) queryParams.append('cursor', params.cursor)
    if (params?.limit) queryParams.append('limit', params.limit.toString())
    if (params?.search) queryParams.append('search', params.search)
    if (params?.version) queryParams.append('version', params.version)
    if (params?.updated_since) queryParams.append('updated_since', params.updated_since)

    const url = `${this.baseUrl}/admin/v0/agents${queryParams.toString() ? '?' + queryParams.toString() : ''}`
    const response = await fetch(url)
    if (!response.ok) {
      throw new Error('Failed to fetch agents')
    }
    return response.json()
  }

  // List PUBLISHED agents only (PUBLIC endpoint)
  async listPublishedAgents(params?: {
    cursor?: string
    limit?: number
    search?: string
    version?: string
    updated_since?: string
  }): Promise<AgentListResponse> {
    const queryParams = new URLSearchParams()
    if (params?.cursor) queryParams.append('cursor', params.cursor)
    if (params?.limit) queryParams.append('limit', params.limit.toString())
    if (params?.search) queryParams.append('search', params.search)
    if (params?.version) queryParams.append('version', params.version)
    if (params?.updated_since) queryParams.append('updated_since', params.updated_since)

    const url = `${this.baseUrl}/v0/agents${queryParams.toString() ? '?' + queryParams.toString() : ''}`
    const response = await fetch(url)
    if (!response.ok) {
      throw new Error('Failed to fetch published agents')
    }
    return response.json()
  }

  // Get a specific agent version
  async getAgent(agentName: string, version: string = 'latest'): Promise<AgentResponse> {
    const encodedName = encodeURIComponent(agentName)
    const encodedVersion = encodeURIComponent(version)
    const response = await fetch(`${this.baseUrl}/admin/v0/agents/${encodedName}/versions/${encodedVersion}`)
    if (!response.ok) {
      throw new Error('Failed to fetch agent')
    }
    return response.json()
  }

  // Get all versions of an agent
  async getAgentVersions(agentName: string): Promise<AgentListResponse> {
    const encodedName = encodeURIComponent(agentName)
    const response = await fetch(`${this.baseUrl}/admin/v0/agents/${encodedName}/versions`)
    if (!response.ok) {
      throw new Error('Failed to fetch agent versions')
    }
    return response.json()
  }

  // Create an agent in the registry
  async createAgent(agent: AgentJSON): Promise<AgentResponse> {
    const response = await fetch(`${this.baseUrl}/admin/v0/agents`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify(agent),
    })
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw new Error(errorData.detail || 'Failed to create agent')
    }
    return response.json()
  }

  // ===== Models API =====

  // List models with pagination and filtering (ADMIN - shows all models)
  async listModels(params?: {
    cursor?: string
    limit?: number
    search?: string
    provider?: string
  }): Promise<ModelListResponse> {
    const queryParams = new URLSearchParams()
    if (params?.cursor) queryParams.append('cursor', params.cursor)
    if (params?.limit) queryParams.append('limit', params.limit.toString())
    if (params?.search) queryParams.append('search', params.search)
    if (params?.provider) queryParams.append('provider', params.provider)

    const url = `${this.baseUrl}/admin/v0/models${queryParams.toString() ? '?' + queryParams.toString() : ''}`
    const response = await fetch(url)
    if (!response.ok) {
      throw new Error('Failed to fetch models')
    }
    return response.json()
  }

  // List PUBLISHED models only (PUBLIC endpoint)
  async listPublishedModels(params?: {
    cursor?: string
    limit?: number
    search?: string
    provider?: string
  }): Promise<ModelListResponse> {
    const queryParams = new URLSearchParams()
    if (params?.cursor) queryParams.append('cursor', params.cursor)
    if (params?.limit) queryParams.append('limit', params.limit.toString())
    if (params?.search) queryParams.append('search', params.search)
    if (params?.provider) queryParams.append('provider', params.provider)

    const url = `${this.baseUrl}/v0/models${queryParams.toString() ? '?' + queryParams.toString() : ''}`
    const response = await fetch(url)
    if (!response.ok) {
      throw new Error('Failed to fetch published models')
    }
    return response.json()
  }

  // Get a specific model
  async getModel(modelName: string): Promise<ModelResponse> {
    const encodedName = encodeURIComponent(modelName)
    const response = await fetch(`${this.baseUrl}/admin/v0/models/${encodedName}`)
    if (!response.ok) {
      throw new Error('Failed to fetch model')
    }
    return response.json()
  }

  // Create a model in the registry
  async createModel(model: ModelJSON): Promise<ModelResponse> {
    const response = await fetch(`${this.baseUrl}/admin/v0/models`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify(model),
    })
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw new Error(errorData.detail || 'Failed to create model')
    }
    return response.json()
  }

  // Delete a model
  async deleteModel(modelName: string): Promise<void> {
    const encodedName = encodeURIComponent(modelName)
    const response = await fetch(`${this.baseUrl}/admin/v0/models/${encodedName}`, {
      method: 'DELETE',
    })
    if (!response.ok) {
      const error = await response.text()
      throw new Error(error || 'Failed to delete model')
    }
  }

  // ===== Import APIs =====

  // Import skills from an external source
  async importSkills(request: ImportRequest): Promise<ImportResponse> {
    const response = await fetch(`${this.baseUrl}/admin/v0/import/skills`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify(request),
    })
    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.message || 'Failed to import skills')
    }
    return response.json()
  }

  // Import agents from an external source
  async importAgents(request: ImportRequest): Promise<ImportResponse> {
    const response = await fetch(`${this.baseUrl}/admin/v0/import/agents`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify(request),
    })
    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.message || 'Failed to import agents')
    }
    return response.json()
  }

  // Import models from an external source
  async importModels(request: ImportRequest): Promise<ImportResponse> {
    const response = await fetch(`${this.baseUrl}/admin/v0/import/models`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify(request),
    })
    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.message || 'Failed to import models')
    }
    return response.json()
  }

  // ===== Deployments API =====

  // Deploy a server to Kubernetes
  async deployServer(params: {
    serverName: string
    version?: string
    config?: Record<string, string>
    preferRemote?: boolean
    resourceType?: 'mcp' | 'agent'
    namespace?: string
  }): Promise<void> {
    const response = await fetch(`${this.baseUrl}/admin/v0/deployments`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify({
        resourceName: params.serverName,
        version: params.version || 'latest',
        config: params.config || {},
        preferRemote: params.preferRemote || false,
        resourceType: params.resourceType || 'mcp',
        runtime: 'kubernetes',
        namespace: params.namespace,
      }),
    })
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw new Error(errorData.message || errorData.detail || 'Failed to deploy server')
    }
  }

  // Get all deployments (includes both managed and external K8s resources)
  async listDeployments(params?: {
    resourceType?: string // 'mcp' | 'agent'
  }): Promise<Array<{
    serverName: string
    version: string
    deployedAt: string
    updatedAt: string
    status: string
    config: Record<string, string>
    preferRemote: boolean
    resourceType: string
    k8sResourceType?: string
    runtime: string
    namespace?: string
    environment?: string
    isExternal?: boolean
  }>> {
    const queryParams = new URLSearchParams()
    if (params?.resourceType) queryParams.append('resourceType', params.resourceType)

    const url = `${this.baseUrl}/admin/v0/deployments${queryParams.toString() ? '?' + queryParams.toString() : ''}`
    const response = await fetch(url)
    if (!response.ok) {
      throw new Error('Failed to fetch deployments')
    }
    const data = await response.json()
    // Map resourceName to serverName for UI compatibility
    return (data.deployments || []).map((d: Record<string, unknown>) => ({
      serverName: d.resourceName,
      version: d.version,
      deployedAt: d.deployedAt,
      updatedAt: d.updatedAt,
      status: d.status,
      config: d.config || {},
      preferRemote: d.preferRemote,
      resourceType: d.resourceType,
      k8sResourceType: d.k8sResourceType,
      runtime: d.runtime || 'kubernetes',
      namespace: d.namespace,
      environment: d.environment,
      isExternal: d.isExternal,
    }))
  }

  // List available environments from DiscoveryConfig
  async listEnvironments(): Promise<Array<{
    name: string
    namespace: string
    labels?: Record<string, string>
  }>> {
    const response = await fetch(`${this.baseUrl}/v0/environments`)
    if (!response.ok) {
      throw new Error('Failed to fetch environments')
    }
    const data = await response.json()
    console.log('Environments API response:', data)
    // Huma might return data directly or wrapped in Body
    return data.environments || data.Body?.environments || []
  }

  // Remove a deployment
  async removeDeployment(serverName: string, version: string, resourceType: string): Promise<void> {
    const encodedName = encodeURIComponent(serverName)
    const encodedVersion = encodeURIComponent(version)
    const response = await fetch(`${this.baseUrl}/admin/v0/deployments/${encodedName}/versions/${encodedVersion}?resourceType=${resourceType}`, {
      method: 'DELETE',
    })
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw new Error(errorData.message || errorData.detail || 'Failed to remove deployment')
    }
  }
}

// Default client without auth (for public endpoints)
export const adminApiClient = new AdminApiClient()

// Create a client with authentication
export function createAuthenticatedClient(accessToken: string | null | undefined): AdminApiClient {
  return new AdminApiClient(API_BASE_URL, () => accessToken || null)
}

