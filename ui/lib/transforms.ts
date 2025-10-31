// Transform functions to convert backend data to frontend types

import { ServerDetail } from "./api"
import { MCPServerWithStatus, Server, GroupedMCPServer } from "./types"

export function transformServerDetail(serverDetail: ServerDetail): MCPServerWithStatus {
  // Parse the JSON data blob
  let serverData: Server
  try {
    serverData = JSON.parse(serverDetail.data)
  } catch (err) {
    // Fallback if data is invalid
    serverData = {
      name: serverDetail.name,
      title: serverDetail.title,
      description: serverDetail.description,
      version: serverDetail.version,
      websiteUrl: serverDetail.website_url,
    }
  }

  return {
    installed: serverDetail.installed,
    installedAt: serverDetail.installed ? serverDetail.updated_at : undefined,
    _meta: {
      "io.modelcontextprotocol.registry/official": {
        isLatest: true, // We can enhance this later
        publishedAt: serverDetail.created_at,
        status: "active",
        updatedAt: serverDetail.updated_at,
      },
    },
    server: {
      ...serverData,
      name: serverDetail.name,
      title: serverDetail.title || serverData.title,
      description: serverDetail.description,
      version: serverDetail.version,
      websiteUrl: serverDetail.website_url || serverData.websiteUrl,
    },
    // Store the database ID for API calls
    _dbId: serverDetail.id,
  }
}

export function transformServerList(servers: ServerDetail[]): MCPServerWithStatus[] {
  return servers.map(transformServerDetail)
}

// Group servers by name, combining multiple versions into one entry
export function groupServersByName(servers: MCPServerWithStatus[]): GroupedMCPServer[] {
  const grouped = new Map<string, MCPServerWithStatus[]>()
  
  // Group all versions by server name
  servers.forEach(server => {
    const name = server.server.name
    if (!grouped.has(name)) {
      grouped.set(name, [])
    }
    grouped.get(name)!.push(server)
  })
  
  // Convert to GroupedMCPServer array
  return Array.from(grouped.entries()).map(([name, versions]) => {
    // Sort versions by semver (simple string comparison for now)
    const sortedVersions = [...versions].sort((a, b) => {
      // Try to prioritize isLatest flag first
      const aIsLatest = a._meta?.["io.modelcontextprotocol.registry/official"]?.isLatest
      const bIsLatest = b._meta?.["io.modelcontextprotocol.registry/official"]?.isLatest
      if (aIsLatest && !bIsLatest) return -1
      if (!aIsLatest && bIsLatest) return 1
      
      // Otherwise compare version strings (descending)
      return b.server.version.localeCompare(a.server.version, undefined, { numeric: true })
    })
    
    const latestVersion = sortedVersions[0]
    const hasInstalledVersion = versions.some(v => v.installed)
    
    return {
      name,
      title: latestVersion.server.title,
      description: latestVersion.server.description,
      icon: latestVersion.server.icons?.[0]?.src,
      websiteUrl: latestVersion.server.websiteUrl,
      repository: latestVersion.server.repository,
      versions: sortedVersions,
      latestVersion,
      hasInstalledVersion,
    }
  })
}

