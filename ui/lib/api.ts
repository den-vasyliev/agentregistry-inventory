// API client for communicating with the Go backend

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8888'

export interface Registry {
  id: number
  name: string
  url: string
  type: string
  created_at: string
  updated_at: string
}

export interface ServerDetail {
  id: number
  registry_id: number
  name: string
  title: string
  description: string
  version: string
  website_url: string
  installed: boolean
  data: string // JSON string of full server data
  created_at: string
  updated_at: string
}

export interface Installation {
  id: number
  resource_type: string
  resource_id: number
  resource_name: string
  version: string
  config: string // JSON string of config
  created_at: string
  updated_at: string
}

class ApiClient {
  private baseUrl: string

  constructor(baseUrl: string = API_BASE_URL) {
    this.baseUrl = baseUrl
  }

  // Registries
  async getRegistries(): Promise<Registry[]> {
    const response = await fetch(`${this.baseUrl}/api/registries`)
    if (!response.ok) {
      throw new Error('Failed to fetch registries')
    }
    return response.json()
  }

  async addRegistry(name: string, url: string, type: string): Promise<void> {
    const response = await fetch(`${this.baseUrl}/api/registries`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ name, url, type }),
    })
    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.error || 'Failed to add registry')
    }
  }

  async removeRegistry(id: number): Promise<void> {
    const response = await fetch(`${this.baseUrl}/api/registries/${id}`, {
      method: 'DELETE',
    })
    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.error || 'Failed to remove registry')
    }
  }

  // Servers
  async getServers(): Promise<ServerDetail[]> {
    const response = await fetch(`${this.baseUrl}/api/servers`)
    if (!response.ok) {
      throw new Error('Failed to fetch servers')
    }
    return response.json()
  }

  async installServer(id: number, config: Record<string, string>): Promise<void> {
    const response = await fetch(`${this.baseUrl}/api/servers/${id}/install`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ config }),
    })
    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.error || 'Failed to install server')
    }
  }

  async uninstallServer(id: number): Promise<void> {
    const response = await fetch(`${this.baseUrl}/api/servers/${id}/uninstall`, {
      method: 'DELETE',
    })
    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.error || 'Failed to uninstall server')
    }
  }

  // Installations
  async getInstallations(): Promise<Installation[]> {
    const response = await fetch(`${this.baseUrl}/api/installations`)
    if (!response.ok) {
      throw new Error('Failed to fetch installations')
    }
    return response.json()
  }

  // Skills
  async getSkills(): Promise<any[]> {
    const response = await fetch(`${this.baseUrl}/api/skills`)
    if (!response.ok) {
      throw new Error('Failed to fetch skills')
    }
    return response.json()
  }

  // Agents
  async getAgents(): Promise<any[]> {
    const response = await fetch(`${this.baseUrl}/api/agents`)
    if (!response.ok) {
      throw new Error('Failed to fetch agents')
    }
    return response.json()
  }

  // Health check
  async healthCheck(): Promise<{ status: string; message: string }> {
    const response = await fetch(`${this.baseUrl}/api/health`)
    if (!response.ok) {
      throw new Error('Health check failed')
    }
    return response.json()
  }
}

export const apiClient = new ApiClient()

