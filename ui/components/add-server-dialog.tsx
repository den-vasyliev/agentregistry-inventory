"use client"

import { useState } from "react"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { adminApiClient, ServerJSON } from "@/lib/admin-api"
import { Loader2, AlertCircle, Plus, Trash2 } from "lucide-react"
import { toast } from "sonner"

interface AddServerDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onServerAdded: () => void
}

export function AddServerDialog({ open, onOpenChange, onServerAdded }: AddServerDialogProps) {
  const [loading, setLoading] = useState(false)
  const [showPreview, setShowPreview] = useState(false)
  const [generatedManifest, setGeneratedManifest] = useState("")

  // Form fields
  const [schema, setSchema] = useState("2025-10-17")
  const [name, setName] = useState("")
  const [title, setTitle] = useState("")
  const [description, setDescription] = useState("")
  const [version, setVersion] = useState("")
  const [ownerEmail, setOwnerEmail] = useState("")
  const [repositorySource, setRepositorySource] = useState<"github" | "gitlab" | "bitbucket">("github")
  const [repositoryUrl, setRepositoryUrl] = useState("")

  // Dynamic fields
  const [packages, setPackages] = useState<Array<{ identifier: string; version: string; registryType: string; transport: string }>>([])
  const [remotes, setRemotes] = useState<Array<{ type: string; url: string }>>([])

  const resetForm = () => {
    setSchema("2025-10-17")
    setName("")
    setTitle("")
    setDescription("")
    setVersion("")
    setOwnerEmail("")
    setRepositoryUrl("")
    setPackages([])
    setRemotes([])
    setShowPreview(false)
    setGeneratedManifest("")
  }

  const toYaml = (obj: any, indent = 0): string => {
    const spaces = '  '.repeat(indent)
    let yaml = ''

    if (Array.isArray(obj)) {
      obj.forEach(item => {
        if (typeof item === 'object' && item !== null) {
          const entries = Object.entries(item)
          if (entries.length > 0) {
            // First key-value pair on same line as dash
            const [firstKey, firstValue] = entries[0]
            if (typeof firstValue === 'object' && firstValue !== null && !Array.isArray(firstValue)) {
              yaml += `${spaces}- ${firstKey}:\n${toYaml(firstValue, indent + 2)}`
            } else if (Array.isArray(firstValue)) {
              yaml += `${spaces}- ${firstKey}:\n${toYaml(firstValue, indent + 2)}`
            } else {
              yaml += `${spaces}- ${firstKey}: ${firstValue}\n`
            }
            // Remaining key-value pairs indented to align with first key
            for (let i = 1; i < entries.length; i++) {
              const [key, value] = entries[i]
              if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
                yaml += `${spaces}  ${key}:\n${toYaml(value, indent + 2)}`
              } else if (Array.isArray(value)) {
                yaml += `${spaces}  ${key}:\n${toYaml(value, indent + 2)}`
              } else if (typeof value === 'string') {
                if (value.includes('\n') || value.includes(':') || value.includes('#') || value.includes('"')) {
                  yaml += `${spaces}  ${key}: "${value.replace(/"/g, '\\"')}"\n`
                } else {
                  yaml += `${spaces}  ${key}: ${value}\n`
                }
              } else {
                yaml += `${spaces}  ${key}: ${value}\n`
              }
            }
          }
        } else {
          yaml += `${spaces}- ${item}\n`
        }
      })
    } else if (typeof obj === 'object' && obj !== null) {
      Object.entries(obj).forEach(([key, value]) => {
        if (value === null || value === undefined) return

        if (Array.isArray(value)) {
          if (value.length === 0) {
            yaml += `${spaces}${key}: []\n`
          } else {
            yaml += `${spaces}${key}:\n${toYaml(value, indent + 1)}`
          }
        } else if (typeof value === 'object') {
          yaml += `${spaces}${key}:\n${toYaml(value, indent + 1)}`
        } else if (typeof value === 'string') {
          // Quote strings with special characters
          if (value.includes('\n') || value.includes(':') || value.includes('#') || value.includes('"')) {
            yaml += `${spaces}${key}: "${value.replace(/"/g, '\\"')}"\n`
          } else {
            yaml += `${spaces}${key}: ${value}\n`
          }
        } else {
          yaml += `${spaces}${key}: ${value}\n`
        }
      })
    }

    return yaml
  }

  const generateManifest = (): string => {
    // Validate required fields
    if (!name.trim()) {
      throw new Error("Server name is required")
    }

    if (!version.trim()) {
      throw new Error("Version is required")
    }
    if (!description.trim()) {
      throw new Error("Description is required")
    }
    if (!ownerEmail.trim()) {
      throw new Error("Owner email is required")
    }

    // Validate email format
    const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    if (!emailPattern.test(ownerEmail.trim())) {
      throw new Error("Please enter a valid email address")
    }

    // Generate Kubernetes resource name from server name
    const resourceName = name.trim().toLowerCase().replace(/[\/\.\s]/g, '-')

    // Build descriptor
    const descriptor: Record<string, unknown> = {
      type: 'mcp-server',
      version: version.trim(),
      description: description.trim(),
      owners: [{ name: ownerEmail.trim() }],
    }

    if (title.trim()) {
      descriptor.title = title.trim()
    }

    if (repositoryUrl.trim()) {
      descriptor.links = [{
        url: repositoryUrl.trim(),
        description: 'Source code repository'
      }]
    }

    if (packages.length > 0) {
      descriptor.packages = packages
        .filter(p => p.identifier.trim() && p.version.trim())
        .map(p => ({
          identifier: p.identifier.trim(),
          version: p.version.trim(),
          registryType: p.registryType,
          transport: {
            type: p.transport || 'stdio',
          },
        }))
    }

    if (remotes.length > 0) {
      descriptor.remotes = remotes
        .filter(r => r.type.trim() && r.url.trim())
        .map(r => ({
          type: r.type.trim(),
          url: r.url.trim(),
        }))
    }

    // Build Application manifest
    const manifest = {
      apiVersion: 'agentregistry.dev/v1alpha1',
      kind: 'Application',
      metadata: {
        name: resourceName,
        labels: {
          'app.kubernetes.io/name': resourceName,
          'app.kubernetes.io/component': 'mcp-server',
          'app.kubernetes.io/part-of': 'agentregistry',
        }
      },
      spec: {
        selector: {
          matchLabels: {
            'app.kubernetes.io/name': resourceName,
          }
        },
        componentKinds: [
          {
            group: 'agentregistry.dev/v1alpha1',
            kind: 'MCPServer'
          }
        ],
        addOwnerRef: true,
        descriptor,
      },
    }

    return `# Generated Agent Registry manifest for MCP Server
# Save as agentregistry.yaml, commit and push to your repository
# Import using the "Import" option in the inventory UI

${toYaml(manifest)}
`
  }

  const handleGeneratePreview = async () => {
    setLoading(true)
    try {
      const manifest = generateManifest()
      setGeneratedManifest(manifest)
      setShowPreview(true)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to generate agentregistry.yaml manifest")
    } finally {
      setLoading(false)
    }
  }

  const handleDownload = () => {
    const blob = new Blob([generatedManifest], { type: 'text/yaml' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `agentregistry.yaml`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)

    toast.success(`Manifest downloaded as agentregistry.yaml`)
  }

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(generatedManifest)
      toast.success("Manifest copied to clipboard!")
    } catch (err) {
      toast.error("Failed to copy manifest")
    }
  }

  const handleBackToForm = () => {
    setShowPreview(false)
  }

  const handleClose = () => {
    onOpenChange(false)
    resetForm()
  }

  const addPackage = () => {
    setPackages([...packages, { identifier: "", version: "", registryType: "npm", transport: "stdio" }])
  }

  const removePackage = (index: number) => {
    setPackages(packages.filter((_, i) => i !== index))
  }

  const updatePackage = (index: number, field: string, value: string) => {
    const updated = [...packages]
    updated[index] = { ...updated[index], [field]: value }
    setPackages(updated)
  }

  const addRemote = () => {
    setRemotes([...remotes, { type: "streamable-http", url: "" }])
  }

  const removeRemote = (index: number) => {
    setRemotes(remotes.filter((_, i) => i !== index))
  }

  const updateRemote = (index: number, field: string, value: string) => {
    const updated = [...remotes]
    updated[index] = { ...updated[index], [field]: value }
    setRemotes(updated)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-6xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{showPreview ? "Application Manifest" : "Add New MCP Server"}</DialogTitle>
          <DialogDescription>
            {showPreview
              ? "Review, copy, or download the manifest to commit to your repository"
              : "Fill in the server details to generate a Kubernetes Application manifest"
            }
          </DialogDescription>
        </DialogHeader>

        {showPreview ? (
          <>
            <div className="space-y-4 py-4">
              <div className="bg-muted p-4 rounded-lg">
                <p className="text-sm text-muted-foreground mb-2">
                  <strong>Next steps:</strong>
                </p>
                <ol className="text-sm text-muted-foreground list-decimal list-inside space-y-1">
                  <li>Download the manifest as <code className="bg-background px-1 py-0.5 rounded">agentregistry.yaml</code></li>
                  <li>Commit and push to your Git repository</li>
                  <li>Import from repository using the &ldquo;Import&rdquo; option in the inventory UI</li>
                </ol>
              </div>

              <div className="space-y-2">
                <Label>Generated Manifest</Label>
                <Textarea
                  value={generatedManifest}
                  readOnly
                  rows={20}
                  className="font-mono text-xs"
                />
              </div>
            </div>

            <div className="flex justify-between gap-2">
              <Button
                variant="outline"
                onClick={handleBackToForm}
                disabled={loading}
              >
                Back to Form
              </Button>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  onClick={handleCopy}
                  disabled={loading}
                >
                  Copy to Clipboard
                </Button>
                <Button
                  onClick={handleDownload}
                  disabled={loading}
                >
                  Download YAML
                </Button>
              </div>
            </div>
          </>
        ) : (
          <>
            <div className="space-y-4 py-4">
              {/* Basic Information */}
              <div className="grid grid-cols-3 gap-4">
            <div className="space-y-2">
              <Label htmlFor="name">Server Name *</Label>
              <Input
                id="name"
                placeholder="my-server"
                value={name}
                onChange={(e) => setName(e.target.value)}
                disabled={loading}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="title">Display Title</Label>
              <Input
                id="title"
                placeholder="My Server"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                disabled={loading}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="version">Version *</Label>
              <Input
                id="version"
                placeholder="1.0.0"
                value={version}
                onChange={(e) => setVersion(e.target.value)}
                disabled={loading}
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="ownerEmail">Owner Email *</Label>
            <Input
              id="ownerEmail"
              type="email"
              placeholder="owner@example.com"
              value={ownerEmail}
              onChange={(e) => setOwnerEmail(e.target.value)}
              disabled={loading}
            />
            <p className="text-xs text-muted-foreground">
              Contact email for the resource owner
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="description">Description *</Label>
            <Textarea
              id="description"
              placeholder="Describe what this server does..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              disabled={loading}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="repositoryUrl">Repository URL</Label>
            <div className="flex gap-2">
              <select
                value={repositorySource}
                onChange={(e) => setRepositorySource(e.target.value as any)}
                className="px-3 py-2 border rounded-md bg-background text-foreground border-input focus:outline-none focus:ring-2 focus:ring-ring"
                disabled={loading}
              >
                <option value="github">GitHub</option>
                <option value="gitlab">GitLab</option>
                <option value="bitbucket">Bitbucket</option>
              </select>
              <Input
                id="repositoryUrl"
                placeholder="https://github.com/user/repo"
                value={repositoryUrl}
                onChange={(e) => setRepositoryUrl(e.target.value)}
                disabled={loading}
                className="flex-1"
              />
            </div>
          </div>

          {/* Packages */}
          <div className="space-y-4 p-4 border rounded-lg">
            <div className="flex items-center justify-between">
              <h3 className="font-semibold text-sm">Packages</h3>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={addPackage}
                disabled={loading}
              >
                <Plus className="h-4 w-4 mr-1" />
                Add Package
              </Button>
            </div>

            {packages.map((pkg, index) => (
              <div key={index} className="space-y-2 p-3 border rounded-md">
                <div className="flex gap-2 items-start">
                  <Input
                    placeholder="Package identifier"
                    value={pkg.identifier}
                    onChange={(e) => updatePackage(index, "identifier", e.target.value)}
                    disabled={loading}
                    className="flex-1"
                  />
                  <Input
                    placeholder="Version"
                    value={pkg.version}
                    onChange={(e) => updatePackage(index, "version", e.target.value)}
                    disabled={loading}
                    className="w-32"
                  />
                  <select
                    value={pkg.registryType}
                    onChange={(e) => updatePackage(index, "registryType", e.target.value)}
                    className="px-3 py-2 border rounded-md bg-background text-foreground border-input focus:outline-none focus:ring-2 focus:ring-ring"
                    disabled={loading}
                  >
                    <option value="npm">npm</option>
                    <option value="pypi">pypi</option>
                    <option value="docker">docker</option>
                  </select>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    onClick={() => removePackage(index)}
                    disabled={loading}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
                <div className="flex gap-2 items-center pl-2">
                  <Label className="text-xs text-muted-foreground">Transport *:</Label>
                  {["stdio", "streamable-http"].map((transport) => (
                    <label key={transport} className="flex items-center gap-1 cursor-pointer">
                      <input
                        type="radio"
                        name={`transport-${index}`}
                        checked={pkg.transport === transport}
                        onChange={() => updatePackage(index, "transport", transport)}
                        disabled={loading}
                        className="border-gray-300"
                      />
                      <span className="text-xs">{transport}</span>
                    </label>
                  ))}
                </div>
              </div>
            ))}

            {packages.length === 0 && (
              <p className="text-sm text-muted-foreground text-center py-2">
                No packages added
              </p>
            )}
          </div>

          {/* Remotes */}
          <div className="space-y-4 p-4 border rounded-lg">
            <div className="flex items-center justify-between">
              <h3 className="font-semibold text-sm">Remotes</h3>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={addRemote}
                disabled={loading}
              >
                <Plus className="h-4 w-4 mr-1" />
                Add Remote
              </Button>
            </div>

            {remotes.map((remote, index) => (
              <div key={index} className="flex gap-2 items-start">
                <Input
                  placeholder="Type (e.g., streamable-http)"
                  value={remote.type}
                  onChange={(e) => updateRemote(index, "type", e.target.value)}
                  disabled={loading}
                  className="w-40"
                />
                <Input
                  placeholder="URL (e.g., https://example.com/mcp)"
                  value={remote.url}
                  onChange={(e) => updateRemote(index, "url", e.target.value)}
                  disabled={loading}
                  className="flex-1"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  onClick={() => removeRemote(index)}
                  disabled={loading}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            ))}

            {remotes.length === 0 && (
              <p className="text-sm text-muted-foreground text-center py-2">
                No remotes added
              </p>
            )}
          </div>
        </div>

        <div className="flex justify-end gap-2">
          <Button
            variant="outline"
            onClick={handleClose}
            disabled={loading}
          >
            Cancel
          </Button>
          <Button
            onClick={handleGeneratePreview}
            disabled={loading || !name.trim() || !version.trim() || !description.trim() || !ownerEmail.trim()}
          >
            {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Generate Manifest
          </Button>
        </div>
      </>
    )}
      </DialogContent>
    </Dialog>
  )
}

