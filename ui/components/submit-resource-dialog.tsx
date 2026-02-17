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

type ResourceKind = "mcp-server" | "agent" | "skill" | "model"

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
  agentType?: string
  mcpServers?: string  // Comma-separated list of MCP server names
  // Skill specific
  category?: string
  skillPackageType?: string
  skillPackageIdentifier?: string
  // Model specific
  provider?: string
  model?: string
  baseUrl?: string
  // Common
  repositoryUrl?: string
  environment?: string
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
    agentType: "",
    mcpServers: "",
    category: "",
    skillPackageType: "",
    skillPackageIdentifier: "",
    provider: "",
    model: "",
    baseUrl: "",
    repositoryUrl: "",
    environment: "dev",
  })

  const generateManifest = (): string => {
    // Generate Kubernetes CRD manifest for Agent Registry
    const resourceName = formData.name.toLowerCase().replace(/[^a-z0-9]/g, '-')
    const crName = formData.kind === "model"
      ? resourceName
      : `${resourceName}-${formData.version.replace(/\./g, '-')}`
    const environment = formData.environment || "dev"
    
    // Generate universal resource UID: name-env-ver
    const resourceUID = `${resourceName}-${environment}-${formData.version.replace(/\./g, '-')}`
    
    // Universal resource labels
    const resourceLabels = {
      "agentregistry.dev/resource-uid": resourceUID,
      "agentregistry.dev/resource-name": resourceName,
      "agentregistry.dev/resource-version": formData.version,
      "agentregistry.dev/resource-environment": environment,
      "agentregistry.dev/resource-source": "manual",
    }
    
    let manifest: Record<string, unknown>
    
    if (formData.kind === "mcp-server") {
      manifest = {
        apiVersion: "agentregistry.dev/v1alpha1",
        kind: "MCPServerCatalog",
        metadata: {
          name: crName,
          labels: resourceLabels,
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

      if (formData.repositoryUrl) {
        (manifest.spec as Record<string, unknown>).repository = {
          url: formData.repositoryUrl,
          source: "github",
        }
      }
    } else if (formData.kind === "agent") {
      const agentSpec: Record<string, unknown> = {
        name: formData.name,
        version: formData.version,
        title: formData.title || formData.name,
        description: formData.description,
        image: formData.image,
      }

      if (formData.framework) agentSpec.framework = formData.framework
      if (formData.language) agentSpec.language = formData.language
      if (formData.modelProvider) agentSpec.modelProvider = formData.modelProvider
      if (formData.modelName) agentSpec.modelName = formData.modelName
      if (formData.agentType) agentSpec.agentType = formData.agentType

      if (formData.mcpServers) {
        const serverNames = formData.mcpServers.split(",").map(s => s.trim()).filter(Boolean)
        if (serverNames.length > 0) {
          agentSpec.mcpServers = serverNames.map(name => ({
            type: "registry",
            name,
            registryServerName: name,
          }))
        }
      }

      if (formData.repositoryUrl) {
        agentSpec.repository = {
          url: formData.repositoryUrl,
          source: "github",
        }
      }

      manifest = {
        apiVersion: "agentregistry.dev/v1alpha1",
        kind: "AgentCatalog",
        metadata: {
          name: crName,
          labels: resourceLabels,
        },
        spec: agentSpec,
      }
    } else if (formData.kind === "model") {
      const modelSpec: Record<string, unknown> = {
        name: formData.name,
        provider: formData.provider,
        model: formData.model,
      }

      if (formData.description) modelSpec.description = formData.description
      if (formData.baseUrl) modelSpec.baseUrl = formData.baseUrl

      manifest = {
        apiVersion: "agentregistry.dev/v1alpha1",
        kind: "ModelCatalog",
        metadata: {
          name: crName,
          namespace: "agentregistry",
          labels: resourceLabels,
        },
        spec: modelSpec,
      }
    } else {
      // skill
      const skillSpec: Record<string, unknown> = {
        name: formData.name,
        version: formData.version,
        title: formData.title || formData.name,
        description: formData.description,
      }

      if (formData.category) skillSpec.category = formData.category

      if (formData.skillPackageIdentifier) {
        skillSpec.packages = [{
          registryType: formData.skillPackageType || "oci",
          identifier: formData.skillPackageIdentifier,
        }]
      }

      if (formData.repositoryUrl) {
        skillSpec.repository = {
          url: formData.repositoryUrl,
          source: "github",
        }
      }

      manifest = {
        apiVersion: "agentregistry.dev/v1alpha1",
        kind: "SkillCatalog",
        metadata: {
          name: crName,
          labels: resourceLabels,
        },
        spec: skillSpec,
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
            // For array items that are objects, render inline with dash
            const itemYaml = formatAsYaml(item as Record<string, unknown>, indent + 2)
            const lines = itemYaml.split('\n').filter(l => l.length > 0)
            if (lines.length > 0) {
              // First line gets the dash prefix
              yaml += `${spaces}  - ${lines[0].trimStart()}\n`
              // Remaining lines keep their indentation (they're already at indent + 2)
              for (let i = 1; i < lines.length; i++) {
                yaml += `${lines[i]}\n`
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
    
    // Generate path for monorepo structure: resources/{kind}/{name}/{name}-{version}.yaml
    const resourceName = formData.name.toLowerCase().replace(/[^a-z0-9]/g, '-')
    const filename = `resources/${formData.kind}/${resourceName}/${resourceName}-${formData.version}.yaml`

    // GitHub new file URL with pre-filled content for monorepo structure
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
    if (formData.kind !== "model" && !formData.version) {
      toast.error("Version is required")
      return
    }
    if (formData.kind === "agent" && !formData.image) {
      toast.error("Container Image is required for agents")
      return
    }
    if (formData.kind === "model" && !formData.provider) {
      toast.error("Provider is required for models")
      return
    }
    if (formData.kind === "model" && !formData.model) {
      toast.error("Model identifier is required for models")
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
      agentType: "",
      mcpServers: "",
      category: "",
      skillPackageType: "",
      skillPackageIdentifier: "",
      provider: "",
      model: "",
      baseUrl: "",
      repositoryUrl: "",
      environment: "dev",
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
              <TabsList className="grid w-full grid-cols-4">
                <TabsTrigger value="mcp-server">MCP Server</TabsTrigger>
                <TabsTrigger value="agent">Agent</TabsTrigger>
                <TabsTrigger value="skill">Skill</TabsTrigger>
                <TabsTrigger value="model">Model</TabsTrigger>
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
                  {kind !== "model" && (
                    <div className="space-y-2">
                      <Label htmlFor="version">Version *</Label>
                      <Input
                        id="version"
                        placeholder="1.0.0"
                        value={formData.version}
                        onChange={(e) => setFormData(prev => ({ ...prev, version: e.target.value }))}
                      />
                    </div>
                  )}
                </div>

                {kind !== "model" && (
                  <div className="space-y-2">
                    <Label htmlFor="title">Title</Label>
                    <Input
                      id="title"
                      placeholder="My Resource Title"
                      value={formData.title}
                      onChange={(e) => setFormData(prev => ({ ...prev, title: e.target.value }))}
                    />
                  </div>
                )}

                <div className="space-y-2">
                  <Label htmlFor="description">Description</Label>
                  <Textarea
                    id="description"
                    placeholder="Describe what this resource does..."
                    value={formData.description}
                    onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
                  />
                </div>

                {kind !== "model" && (
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="environment">Environment</Label>
                      <Select
                        value={formData.environment}
                        onValueChange={(v) => setFormData(prev => ({ ...prev, environment: v }))}
                      >
                        <SelectTrigger id="environment">
                          <SelectValue placeholder="Select environment" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="dev">Development</SelectItem>
                          <SelectItem value="staging">Staging</SelectItem>
                          <SelectItem value="prod">Production</SelectItem>
                        </SelectContent>
                      </Select>
                      <p className="text-xs text-muted-foreground">
                        Used for resource UID generation
                      </p>
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
                  </div>
                )}

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
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="image">Container Image *</Label>
                      <Input
                        id="image"
                        placeholder="ghcr.io/org/agent:latest"
                        value={formData.image}
                        onChange={(e) => setFormData(prev => ({ ...prev, image: e.target.value }))}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="agentType">Agent Type</Label>
                      <Select
                        value={formData.agentType}
                        onValueChange={(v) => setFormData(prev => ({ ...prev, agentType: v }))}
                      >
                        <SelectTrigger id="agentType">
                          <SelectValue placeholder="Select type" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="Declarative">Declarative</SelectItem>
                          <SelectItem value="BYO">BYO (Bring Your Own)</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
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
                    <Label htmlFor="mcpServers">MCP Servers (Tools/Skills)</Label>
                    <Textarea
                      id="mcpServers"
                      placeholder="Comma-separated MCP server names (e.g., brave-search, sqlite, filesystem)"
                      value={formData.mcpServers}
                      onChange={(e) => setFormData(prev => ({ ...prev, mcpServers: e.target.value }))}
                      rows={2}
                    />
                    <p className="text-xs text-muted-foreground">
                      Reference MCP servers from the catalog to provide tools for this agent.
                    </p>
                  </div>
                </TabsContent>

                {/* Skill specific fields */}
                <TabsContent value="skill" className="mt-0 space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="category">Category</Label>
                    <Input
                      id="category"
                      placeholder="e.g., code-generation, testing, data-processing"
                      value={formData.category}
                      onChange={(e) => setFormData(prev => ({ ...prev, category: e.target.value }))}
                    />
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="skillPackageType">Package Type</Label>
                      <Select
                        value={formData.skillPackageType}
                        onValueChange={(v) => setFormData(prev => ({ ...prev, skillPackageType: v }))}
                      >
                        <SelectTrigger id="skillPackageType">
                          <SelectValue placeholder="Select type" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="oci">OCI/Docker</SelectItem>
                          <SelectItem value="npm">NPM</SelectItem>
                          <SelectItem value="pypi">PyPI</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="skillPackageIdentifier">Package Identifier</Label>
                      <Input
                        id="skillPackageIdentifier"
                        placeholder="e.g., ghcr.io/org/skill:v1"
                        value={formData.skillPackageIdentifier}
                        onChange={(e) => setFormData(prev => ({ ...prev, skillPackageIdentifier: e.target.value }))}
                      />
                    </div>
                  </div>
                </TabsContent>

                {/* Model specific fields */}
                <TabsContent value="model" className="mt-0 space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="provider">Provider *</Label>
                      <Select
                        value={formData.provider}
                        onValueChange={(v) => setFormData(prev => ({ ...prev, provider: v }))}
                      >
                        <SelectTrigger id="provider">
                          <SelectValue placeholder="Select provider" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="Anthropic">Anthropic</SelectItem>
                          <SelectItem value="OpenAI">OpenAI</SelectItem>
                          <SelectItem value="AzureOpenAI">Azure OpenAI</SelectItem>
                          <SelectItem value="Ollama">Ollama</SelectItem>
                          <SelectItem value="Gemini">Gemini</SelectItem>
                          <SelectItem value="GeminiVertexAI">Gemini Vertex AI</SelectItem>
                          <SelectItem value="AnthropicVertexAI">Anthropic Vertex AI</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="model">Model *</Label>
                      <Input
                        id="model"
                        placeholder="e.g., gpt-4o, claude-sonnet-4-5-20250514"
                        value={formData.model}
                        onChange={(e) => setFormData(prev => ({ ...prev, model: e.target.value }))}
                      />
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="baseUrl">Base URL</Label>
                    <Input
                      id="baseUrl"
                      placeholder="e.g., https://api.openai.com/v1"
                      value={formData.baseUrl}
                      onChange={(e) => setFormData(prev => ({ ...prev, baseUrl: e.target.value }))}
                    />
                    <p className="text-xs text-muted-foreground">
                      API endpoint URL. Required for Ollama and custom deployments.
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
