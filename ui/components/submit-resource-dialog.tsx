"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { GitPullRequest, Copy, ExternalLink } from "lucide-react"
import { toast } from "sonner"

interface SubmitResourceDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

type ResourceKind = "mcp-server" | "agent" | "skill"

interface ManifestData {
  kind: ResourceKind
  name: string
  version: string
  title: string
  description: string
  // MCP Server specific
  packageType?: string
  packageIdentifier?: string
  transport?: string
  remoteUrl?: string
  // Agent specific
  image?: string
  framework?: string
  language?: string
  modelProvider?: string
  modelName?: string
  mcpServers?: string  // Comma-separated list of MCP server names
  // Common
  repositoryUrl?: string
}

export function SubmitResourceDialog({ open, onOpenChange }: SubmitResourceDialogProps) {
  const [step, setStep] = useState<"form" | "manifest">("form")
  const [kind, setKind] = useState<ResourceKind>("mcp-server")
  const [formData, setFormData] = useState<ManifestData>({
    kind: "mcp-server",
    name: "",
    version: "1.0.0",
    title: "",
    description: "",
    packageType: "npm",
    packageIdentifier: "",
    transport: "stdio",
    remoteUrl: "",
    image: "",
    framework: "",
    language: "",
    modelProvider: "",
    modelName: "",
    mcpServers: "",
    repositoryUrl: "",
  })

  const generateManifest = (): string => {
    // Generate Kubernetes CRD manifest for Agent Registry
    const resourceName = formData.name.toLowerCase().replace(/[^a-z0-9]/g, '-')
    const crName = `${resourceName}-${formData.version.replace(/\./g, '-')}`
    
    let manifest: Record<string, unknown>
    
    if (formData.kind === "mcp-server") {
      manifest = {
        apiVersion: "agentregistry.dev/v1alpha1",
        kind: "MCPServerCatalog",
        metadata: {
          name: crName,
          labels: {
            "agentregistry.dev/name": resourceName,
            "agentregistry.dev/version": formData.version,
          },
        },
        spec: {
          name: formData.name,
          version: formData.version,
          title: formData.title || formData.name,
          description: formData.description,
        } as Record<string, unknown>,
      }
      
      if (formData.packageIdentifier) {
        (manifest.spec as Record<string, unknown>).packages = [{
          registryType: formData.packageType,
          identifier: formData.packageIdentifier,
          transport: {
            type: formData.transport,
          },
        }]
      }
      
      if (formData.remoteUrl) {
        (manifest.spec as Record<string, unknown>).remotes = [{
          url: formData.remoteUrl,
          type: "streamable-http",
        }]
      }
    } else if (formData.kind === "agent") {
      manifest = {
        apiVersion: "agentregistry.dev/v1alpha1",
        kind: "AgentCatalog",
        metadata: {
          name: crName,
          labels: {
            "agentregistry.dev/name": resourceName,
            "agentregistry.dev/version": formData.version,
          },
        },
        spec: {
          name: formData.name,
          version: formData.version,
          title: formData.title || formData.name,
          description: formData.description,
          image: formData.image,
        } as Record<string, unknown>,
      }
    } else {
      // skill
      manifest = {
        apiVersion: "agentregistry.dev/v1alpha1",
        kind: "SkillCatalog",
        metadata: {
          name: crName,
          labels: {
            "agentregistry.dev/name": resourceName,
            "agentregistry.dev/version": formData.version,
          },
        },
        spec: {
          name: formData.name,
          version: formData.version,
          title: formData.title || formData.name,
          description: formData.description,
        },
      }
    }

    // Convert to YAML-like format (simple implementation)
    return `# Agent Registry manifest
# Save as agentregistry.yaml in your repository

${formatAsYaml(manifest)}`
  }

  const formatAsYaml = (obj: Record<string, unknown>, indent = 0): string => {
    const spaces = "  ".repeat(indent)
    let yaml = ""

    for (const [key, value] of Object.entries(obj)) {
      if (value === undefined || value === null || value === "") continue

      if (Array.isArray(value)) {
        yaml += `${spaces}${key}:\n`
        for (const item of value) {
          if (typeof item === "object" && item !== null) {
            // For array items that are objects, use the inline dash format
            const itemYaml = formatAsYaml(item as Record<string, unknown>, indent + 2)
            const lines = itemYaml.trim().split('\n')
            if (lines.length > 0) {
              // First line gets the dash
              yaml += `${spaces}  - ${lines[0].trimStart()}\n`
              // Remaining lines get extra indentation
              for (let i = 1; i < lines.length; i++) {
                yaml += `${spaces}    ${lines[i].trimStart()}\n`
              }
            }
          } else {
            yaml += `${spaces}  - ${item}\n`
          }
        }
      } else if (typeof value === "object") {
        yaml += `${spaces}${key}:\n`
        yaml += formatAsYaml(value as Record<string, unknown>, indent + 1)
      } else {
        yaml += `${spaces}${key}: ${value}\n`
      }
    }

    return yaml
  }

  const handleCopyManifest = () => {
    navigator.clipboard.writeText(generateManifest())
    toast.success("Manifest copied to clipboard")
  }

  const handleOpenPR = () => {
    if (!formData.repositoryUrl) {
      toast.error("Please enter a repository URL")
      return
    }

    // Parse GitHub URL
    const match = formData.repositoryUrl.match(/github\.com\/([^\/]+)\/([^\/]+)/)
    if (!match) {
      toast.error("Please enter a valid GitHub repository URL")
      return
    }

    const [, owner, repo] = match
    const repoName = repo.replace(/\.git$/, "")
    const manifest = generateManifest()
    const encodedManifest = encodeURIComponent(manifest)
    const filename = "agentregistry.yaml"

    // GitHub new file URL with pre-filled content
    // Using the new file creation URL pattern
    const prUrl = `https://github.com/${owner}/${repoName}/new/main?filename=${filename}&value=${encodedManifest}&message=feat:%20add%20Agent%20Registry%20manifest&description=This%20PR%20adds%20the%20.agentregistry.yaml%20manifest%20to%20register%20this%20resource%20in%20Agent%20Registry.`

    window.open(prUrl, "_blank")
    toast.success("Opening GitHub to create the manifest file")
  }

  const handleNext = () => {
    if (!formData.name) {
      toast.error("Name is required")
      return
    }
    if (!formData.version) {
      toast.error("Version is required")
      return
    }
    setStep("manifest")
  }

  const handleBack = () => {
    setStep("form")
  }

  const handleKindChange = (newKind: ResourceKind) => {
    setKind(newKind)
    setFormData(prev => ({ ...prev, kind: newKind }))
  }

  const resetForm = () => {
    setStep("form")
    setKind("mcp-server")
    setFormData({
      kind: "mcp-server",
      name: "",
      version: "1.0.0",
      title: "",
      description: "",
      packageType: "npm",
      packageIdentifier: "",
      transport: "stdio",
      remoteUrl: "",
      image: "",
      framework: "",
      language: "",
      modelProvider: "",
      modelName: "",
      mcpServers: "",
      repositoryUrl: "",
    })
  }

  return (
    <Dialog open={open} onOpenChange={(isOpen) => {
      if (!isOpen) resetForm()
      onOpenChange(isOpen)
    }}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <GitPullRequest className="h-5 w-5" />
            Submit Resource for Review
          </DialogTitle>
          <DialogDescription>
            {step === "form"
              ? "Fill in the details to generate an Agent Registry manifest."
              : "Review the manifest and open a PR to submit for review."}
          </DialogDescription>
        </DialogHeader>

        {step === "form" ? (
          <div className="space-y-4 py-4">
            <Tabs value={kind} onValueChange={(v) => handleKindChange(v as ResourceKind)}>
              <TabsList className="grid w-full grid-cols-3">
                <TabsTrigger value="mcp-server">MCP Server</TabsTrigger>
                <TabsTrigger value="agent">Agent</TabsTrigger>
                <TabsTrigger value="skill">Skill</TabsTrigger>
              </TabsList>

              <div className="mt-4 space-y-4">
                {/* Common fields */}
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="name">Name *</Label>
                    <Input
                      id="name"
                      placeholder="my-resource"
                      value={formData.name}
                      onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="version">Version *</Label>
                    <Input
                      id="version"
                      placeholder="1.0.0"
                      value={formData.version}
                      onChange={(e) => setFormData(prev => ({ ...prev, version: e.target.value }))}
                    />
                  </div>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="title">Title</Label>
                  <Input
                    id="title"
                    placeholder="My Resource Title"
                    value={formData.title}
                    onChange={(e) => setFormData(prev => ({ ...prev, title: e.target.value }))}
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="description">Description</Label>
                  <Textarea
                    id="description"
                    placeholder="Describe what this resource does..."
                    value={formData.description}
                    onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="repositoryUrl">Repository URL *</Label>
                  <Input
                    id="repositoryUrl"
                    placeholder="https://github.com/owner/repo"
                    value={formData.repositoryUrl}
                    onChange={(e) => setFormData(prev => ({ ...prev, repositoryUrl: e.target.value }))}
                  />
                </div>

                {/* MCP Server specific fields */}
                <TabsContent value="mcp-server" className="mt-0 space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="packageType">Package Type</Label>
                    <Select
                      value={formData.packageType}
                      onValueChange={(v) => setFormData(prev => ({ ...prev, packageType: v }))}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="npm">NPM</SelectItem>
                        <SelectItem value="pypi">PyPI</SelectItem>
                        <SelectItem value="oci">OCI/Docker</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="packageIdentifier">Package Identifier</Label>
                    <Input
                      id="packageIdentifier"
                      placeholder="@org/package-name or package-name"
                      value={formData.packageIdentifier}
                      onChange={(e) => setFormData(prev => ({ ...prev, packageIdentifier: e.target.value }))}
                    />
                    <p className="text-xs text-muted-foreground">
                      Local MCP servers use stdio transport (standard input/output)
                    </p>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="remoteUrl">Remote URL (optional)</Label>
                    <Input
                      id="remoteUrl"
                      placeholder="https://api.example.com/mcp"
                      value={formData.remoteUrl}
                      onChange={(e) => setFormData(prev => ({ ...prev, remoteUrl: e.target.value }))}
                    />
                    <p className="text-xs text-muted-foreground">
                      For remote MCP servers over HTTP with SSE streaming
                    </p>
                  </div>
                </TabsContent>

                {/* Agent specific fields */}
                <TabsContent value="agent" className="mt-0 space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="image">Container Image</Label>
                    <Input
                      id="image"
                      placeholder="ghcr.io/org/agent:latest"
                      value={formData.image}
                      onChange={(e) => setFormData(prev => ({ ...prev, image: e.target.value }))}
                    />
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="framework">Framework</Label>
                      <Input
                        id="framework"
                        placeholder="e.g., langchain, autogen"
                        value={formData.framework}
                        onChange={(e) => setFormData(prev => ({ ...prev, framework: e.target.value }))}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="language">Language</Label>
                      <Input
                        id="language"
                        placeholder="e.g., python, typescript"
                        value={formData.language}
                        onChange={(e) => setFormData(prev => ({ ...prev, language: e.target.value }))}
                      />
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="modelProvider">Model Provider</Label>
                      <Input
                        id="modelProvider"
                        placeholder="e.g., openai, anthropic"
                        value={formData.modelProvider}
                        onChange={(e) => setFormData(prev => ({ ...prev, modelProvider: e.target.value }))}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="modelName">Model Name</Label>
                      <Input
                        id="modelName"
                        placeholder="e.g., gpt-4, claude-3"
                        value={formData.modelName}
                        onChange={(e) => setFormData(prev => ({ ...prev, modelName: e.target.value }))}
                      />
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="mcpServers">MCP Servers (Tools/Skills) *</Label>
                    <Textarea
                      id="mcpServers"
                      placeholder="Enter MCP server names from registry, comma-separated (e.g., brave-search, sqlite, filesystem)"
                      value={formData.mcpServers}
                      onChange={(e) => setFormData(prev => ({ ...prev, mcpServers: e.target.value }))}
                      rows={3}
                    />
                    <p className="text-xs text-muted-foreground">
                      Agents need at least one MCP server to provide tools/skills. Reference servers from the Agent Registry catalog.
                    </p>
                  </div>
                </TabsContent>

                {/* Skill specific fields */}
                <TabsContent value="skill" className="mt-0 space-y-4">
                  <div className="p-4 bg-muted/50 rounded-lg">
                    <p className="text-sm text-muted-foreground">
                      Skills are typically discovered from deployed agents. The name, version, title, and description fields above are sufficient for manual skill registration.
                    </p>
                  </div>
                </TabsContent>
              </div>
            </Tabs>
          </div>
        ) : (
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label>Generated Manifest (agentregistry.yaml)</Label>
                <Button variant="ghost" size="sm" onClick={handleCopyManifest}>
                  <Copy className="h-4 w-4 mr-1" />
                  Copy
                </Button>
              </div>
              <pre className="bg-muted p-4 rounded-lg text-sm overflow-auto max-h-[300px] font-mono">
                {generateManifest()}
              </pre>
            </div>
            <p className="text-sm text-muted-foreground">
              Click &quot;Open PR&quot; to create a new file in your repository with this manifest.
              After the PR is merged, CI/CD will submit the resource for review.
            </p>
          </div>
        )}

        <DialogFooter>
          {step === "form" ? (
            <Button onClick={handleNext}>
              Next: Preview Manifest
            </Button>
          ) : (
            <div className="flex gap-2 w-full justify-between">
              <Button variant="outline" onClick={handleBack}>
                Back
              </Button>
              <Button onClick={handleOpenPR} className="gap-2">
                <ExternalLink className="h-4 w-4" />
                Open PR on GitHub
              </Button>
            </div>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
