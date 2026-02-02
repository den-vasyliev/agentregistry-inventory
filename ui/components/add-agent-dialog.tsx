"use client"

import { useState } from "react"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { Loader2 } from "lucide-react"
import { toast } from "sonner"

interface AddAgentDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onAgentAdded: () => void
}

export function AddAgentDialog({ open, onOpenChange, onAgentAdded }: AddAgentDialogProps) {
  const [loading, setLoading] = useState(false)
  const [showPreview, setShowPreview] = useState(false)
  const [generatedManifest, setGeneratedManifest] = useState("")

  // Form fields
  const [name, setName] = useState("")
  const [version, setVersion] = useState("")
  const [description, setDescription] = useState("")
  const [ownerEmail, setOwnerEmail] = useState("")
  const [image, setImage] = useState("")
  const [language, setLanguage] = useState("")
  const [framework, setFramework] = useState("")
  const [modelProvider, setModelProvider] = useState("")
  const [modelName, setModelName] = useState("")
  const [repositorySource, setRepositorySource] = useState<"github" | "gitlab" | "bitbucket">("github")
  const [repositoryUrl, setRepositoryUrl] = useState("")

  const resetForm = () => {
    setName("")
    setVersion("")
    setDescription("")
    setOwnerEmail("")
    setImage("")
    setLanguage("")
    setFramework("")
    setModelProvider("")
    setModelName("")
    setRepositoryUrl("")
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
    if (!name.trim()) throw new Error("Agent name is required")
    if (!version.trim()) throw new Error("Version is required")
    if (!description.trim()) throw new Error("Description is required")
    if (!ownerEmail.trim()) throw new Error("Owner email is required")
    if (!image.trim()) throw new Error("Docker image is required")
    if (!language.trim()) throw new Error("Language is required")
    if (!framework.trim()) throw new Error("Framework is required")
    if (!modelProvider.trim()) throw new Error("Model provider is required")
    if (!modelName.trim()) throw new Error("Model name is required")

    // Validate email format
    const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    if (!emailPattern.test(ownerEmail.trim())) {
      throw new Error("Please enter a valid email address")
    }

    // Generate Kubernetes resource name from agent name
    const resourceName = name.trim().toLowerCase().replace(/[\/\.\s]/g, '-')

    // Build descriptor
    const descriptor: Record<string, unknown> = {
      type: 'agent',
      version: version.trim(),
      description: description.trim(),
      owners: [{ name: ownerEmail.trim() }],
      image: image.trim(),
      language: language.trim(),
      framework: framework.trim(),
      model: {
        provider: modelProvider.trim(),
        name: modelName.trim(),
      },
    }

    if (repositoryUrl.trim()) {
      descriptor.links = [{
        url: repositoryUrl.trim(),
        description: 'Source code repository'
      }]
    }

    // Build Application manifest
    const manifest = {
      apiVersion: 'agentregistry.dev/v1alpha1',
      kind: 'Application',
      metadata: {
        name: resourceName,
        labels: {
          'app.kubernetes.io/name': resourceName,
          'app.kubernetes.io/component': 'agent',
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
            kind: 'Agent'
          }
        ],
        addOwnerRef: true,
        descriptor,
      },
    }

    return `# Generated Application manifest for Agent
# Save as application.yaml, commit and push to your repository
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
      toast.error(err instanceof Error ? err.message : "Failed to generate manifest")
    } finally {
      setLoading(false)
    }
  }

  const handleDownload = () => {
    const blob = new Blob([generatedManifest], { type: 'text/yaml' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `application.yaml`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)

    toast.success(`Manifest downloaded as application.yaml`)
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

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{showPreview ? "Application Manifest" : "Add New Agent"}</DialogTitle>
          <DialogDescription>
            {showPreview
              ? "Review, copy, or download the manifest to commit to your repository"
              : "Fill in the agent details to generate a Kubernetes Application manifest"
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
                  <li>Download the manifest as <code className="bg-background px-1 py-0.5 rounded">application.yaml</code></li>
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
                  <Label htmlFor="name">Agent Name *</Label>
                  <Input
                    id="name"
                    placeholder="my-agent"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
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
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="description">Description *</Label>
                <Textarea
                  id="description"
                  placeholder="Describe what this agent does..."
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={3}
                  disabled={loading}
                />
              </div>

              {/* Docker Image */}
              <div className="space-y-2">
                <Label htmlFor="image">Docker Image *</Label>
                <Input
                  id="image"
                  placeholder="ghcr.io/user/agent:latest"
                  value={image}
                  onChange={(e) => setImage(e.target.value)}
                  disabled={loading}
                />
                <p className="text-xs text-muted-foreground">
                  Full Docker image path including registry and tag
                </p>
              </div>

              {/* Technical Details */}
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="language">Language *</Label>
                  <Input
                    id="language"
                    placeholder="python"
                    value={language}
                    onChange={(e) => setLanguage(e.target.value)}
                    disabled={loading}
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="framework">Framework *</Label>
                  <Input
                    id="framework"
                    placeholder="langchain"
                    value={framework}
                    onChange={(e) => setFramework(e.target.value)}
                    disabled={loading}
                  />
                </div>
              </div>

              {/* Model Configuration */}
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="modelProvider">Model Provider *</Label>
                  <Input
                    id="modelProvider"
                    placeholder="openai"
                    value={modelProvider}
                    onChange={(e) => setModelProvider(e.target.value)}
                    disabled={loading}
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="modelName">Model Name *</Label>
                  <Input
                    id="modelName"
                    placeholder="gpt-4"
                    value={modelName}
                    onChange={(e) => setModelName(e.target.value)}
                    disabled={loading}
                  />
                </div>
              </div>

              {/* Repository */}
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
                disabled={loading || !name.trim() || !version.trim() || !description.trim() || !ownerEmail.trim() || !image.trim() || !language.trim() || !framework.trim() || !modelProvider.trim() || !modelName.trim()}
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

