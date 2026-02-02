"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
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
import { Card } from "@/components/ui/card"
import { toast } from "sonner"
import { Copy, Download, Check, GitBranch, Loader2 } from "lucide-react"

interface SubmitResourceDialogProps {
  trigger?: React.ReactNode
  onSubmitted?: () => void
}

type ResourceKind = "mcp-server" | "agent" | "skill"
type TransportType = "stdio" | "streamable-http"
type PackageType = "oci" | "npm" | "pypi"

interface ManifestData {
  kind: ResourceKind
  name: string
  version: string
  title: string
  description: string
  // Package
  packageType: PackageType
  packageIdentifier: string
  transport: TransportType
  // Agent specific
  framework: string
  language: string
  modelProvider: string
  modelName: string
  // Skill specific
  category: string
  // Config
  requiredEnvVars: string
}

const defaultManifest: ManifestData = {
  kind: "mcp-server",
  name: "",
  version: "1.0.0",
  title: "",
  description: "",
  packageType: "oci",
  packageIdentifier: "",
  transport: "stdio",
  framework: "kagent",
  language: "python",
  modelProvider: "anthropic",
  modelName: "claude-sonnet-4",
  category: "general",
  requiredEnvVars: "",
}

export function SubmitResourceDialog({ trigger, onSubmitted }: SubmitResourceDialogProps) {
  const [open, setOpen] = useState(false)
  const [step, setStep] = useState<"form" | "generate" | "apply">("form")
  const [manifest, setManifest] = useState<ManifestData>(defaultManifest)
  const [repoUrl, setRepoUrl] = useState("")
  const [copied, setCopied] = useState(false)
  const [applying, setApplying] = useState(false)

  const updateManifest = (field: keyof ManifestData, value: string) => {
    setManifest(prev => ({ ...prev, [field]: value }))
  }

  const generateYaml = (): string => {
    const lines: string[] = [
      "# .agentregistry.yaml - Agent Registry Manifest",
      "",
      "# Required",
      `kind: ${manifest.kind}`,
      `name: ${manifest.name || "my-resource"}`,
      `version: ${manifest.version}`,
      "",
      "# Metadata",
      `title: ${manifest.title || manifest.name || "My Resource"}`,
      `description: ${manifest.description || "Description of your resource"}`,
    ]

    // MCP Server - includes package info
    if (manifest.kind === "mcp-server") {
      lines.push("")
      lines.push("# Package info (how to run it)")
      lines.push("packages:")
      lines.push(`  - type: ${manifest.packageType}`)
      if (manifest.packageType === "oci") {
        lines.push(`    image: ${manifest.packageIdentifier || "ghcr.io/org/image:latest"}`)
      } else {
        lines.push(`    identifier: ${manifest.packageIdentifier || "package-name"}`)
      }
      lines.push(`    transport: ${manifest.transport}`)

      // Config requirements (only for MCP servers)
      if (manifest.requiredEnvVars.trim()) {
        lines.push("")
        lines.push("# Configuration requirements")
        lines.push("config:")
        lines.push("  required:")
        manifest.requiredEnvVars.split(",").map(v => v.trim()).filter(Boolean).forEach(envVar => {
          lines.push(`    - name: ${envVar}`)
          lines.push(`      description: Required environment variable`)
        })
      }
    }

    // Agent - includes image and agent config
    if (manifest.kind === "agent") {
      lines.push("")
      lines.push("# Container image")
      lines.push("packages:")
      lines.push("  - type: oci")
      lines.push(`    image: ${manifest.packageIdentifier || "ghcr.io/org/my-agent:latest"}`)
      lines.push("")
      lines.push("# Agent configuration")
      lines.push("agent:")
      lines.push(`  framework: ${manifest.framework}`)
      lines.push(`  language: ${manifest.language}`)
      if (manifest.modelProvider) {
        lines.push(`  modelProvider: ${manifest.modelProvider}`)
      }
      if (manifest.modelName) {
        lines.push(`  modelName: ${manifest.modelName}`)
      }
    }

    // Skill - simple config only
    if (manifest.kind === "skill") {
      lines.push("")
      lines.push("# Skill configuration")
      lines.push("skill:")
      lines.push(`  category: ${manifest.category}`)
    }

    return lines.join("\n")
  }

  const copyToClipboard = async () => {
    await navigator.clipboard.writeText(generateYaml())
    setCopied(true)
    toast.success("Copied to clipboard!")
    setTimeout(() => setCopied(false), 2000)
  }

  const downloadFile = () => {
    const blob = new Blob([generateYaml()], { type: "text/yaml" })
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = ".agentregistry.yaml"
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    toast.success("File downloaded!")
  }

  const handleApply = async () => {
    if (!repoUrl.trim()) {
      toast.error("Please enter your repository URL")
      return
    }

    setApplying(true)
    try {
      const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL || ""}/admin/v0/submit`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ repositoryUrl: repoUrl }),
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.message || error.detail || "Failed to submit")
      }

      toast.success("Resource submitted for review!")
      setOpen(false)
      setStep("form")
      setManifest(defaultManifest)
      setRepoUrl("")
      onSubmitted?.()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to submit resource")
    } finally {
      setApplying(false)
    }
  }

  const handleClose = () => {
    setOpen(false)
    // Reset after animation
    setTimeout(() => {
      setStep("form")
      setManifest(defaultManifest)
      setRepoUrl("")
    }, 200)
  }

  const triggerElement = trigger ? (
    <span onClick={() => setOpen(true)}>{trigger}</span>
  ) : (
    <Button onClick={() => setOpen(true)}>Submit Resource</Button>
  )

  return (
    <>
      {triggerElement}
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto" onClose={handleClose}>
        <DialogHeader>
          <DialogTitle>
            {step === "form" && "Submit Resource to Registry"}
            {step === "generate" && "Generated Manifest"}
            {step === "apply" && "Apply from Repository"}
          </DialogTitle>
          <DialogDescription>
            {step === "form" && "Fill in the details to generate your .agentregistry.yaml manifest file."}
            {step === "generate" && "Copy this file to the root of your repository and commit it."}
            {step === "apply" && "Enter your repository URL to submit for review."}
          </DialogDescription>
        </DialogHeader>

        {step === "form" && (
          <div className="space-y-6 py-4">
            {/* Resource Kind */}
            <div className="space-y-2">
              <Label>Resource Type</Label>
              <Select value={manifest.kind} onValueChange={(v) => updateManifest("kind", v)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="mcp-server">MCP Server</SelectItem>
                  <SelectItem value="agent">Agent</SelectItem>
                  <SelectItem value="skill">Skill</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Basic Info */}
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Name *</Label>
                <Input
                  placeholder={
                    manifest.kind === "mcp-server" ? "my-mcp-server" :
                    manifest.kind === "agent" ? "my-agent" :
                    "my-skill"
                  }
                  value={manifest.name}
                  onChange={(e) => updateManifest("name", e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label>Version *</Label>
                <Input
                  placeholder="1.0.0"
                  value={manifest.version}
                  onChange={(e) => updateManifest("version", e.target.value)}
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label>Title</Label>
              <Input
                placeholder={
                  manifest.kind === "mcp-server" ? "My MCP Server" :
                  manifest.kind === "agent" ? "My Agent" :
                  "My Skill"
                }
                value={manifest.title}
                onChange={(e) => updateManifest("title", e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label>Description</Label>
              <Textarea
                placeholder={
                  manifest.kind === "mcp-server" ? "A server that provides tools for..." :
                  manifest.kind === "agent" ? "An AI agent that helps with..." :
                  "A skill that enables..."
                }
                value={manifest.description}
                onChange={(e) => updateManifest("description", e.target.value)}
                rows={3}
              />
            </div>

            {/* Package Info - Only for MCP servers */}
            {manifest.kind === "mcp-server" && (
              <div className="space-y-4">
                <h4 className="font-medium">Package Information</h4>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Package Type</Label>
                    <Select value={manifest.packageType} onValueChange={(v) => updateManifest("packageType", v)}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="oci">OCI Image (Docker)</SelectItem>
                        <SelectItem value="npm">npm Package</SelectItem>
                        <SelectItem value="pypi">PyPI Package</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label>Transport</Label>
                    <Select value={manifest.transport} onValueChange={(v) => updateManifest("transport", v)}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="stdio">stdio</SelectItem>
                        <SelectItem value="streamable-http">streamable-http</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
                <div className="space-y-2">
                  <Label>{manifest.packageType === "oci" ? "Image" : "Package Identifier"} *</Label>
                  <Input
                    placeholder={manifest.packageType === "oci" ? "ghcr.io/org/image:latest" : "package-name"}
                    value={manifest.packageIdentifier}
                    onChange={(e) => updateManifest("packageIdentifier", e.target.value)}
                  />
                </div>

                {/* Environment Variables - Only for MCP servers */}
                <div className="space-y-2">
                  <Label>Required Environment Variables</Label>
                  <Input
                    placeholder="API_KEY, SECRET_TOKEN (comma-separated)"
                    value={manifest.requiredEnvVars}
                    onChange={(e) => updateManifest("requiredEnvVars", e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">
                    Comma-separated list of required environment variables
                  </p>
                </div>
              </div>
            )}

            {/* Agent-specific fields */}
            {manifest.kind === "agent" && (
              <div className="space-y-4">
                <h4 className="font-medium">Agent Configuration</h4>
                <div className="space-y-2">
                  <Label>Container Image *</Label>
                  <Input
                    placeholder="ghcr.io/org/my-agent:latest"
                    value={manifest.packageIdentifier}
                    onChange={(e) => updateManifest("packageIdentifier", e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">
                    OCI image reference for your agent container
                  </p>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Framework</Label>
                    <Select value={manifest.framework} onValueChange={(v) => updateManifest("framework", v)}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="kagent">KAgent</SelectItem>
                        <SelectItem value="langchain">LangChain</SelectItem>
                        <SelectItem value="autogen">AutoGen</SelectItem>
                        <SelectItem value="crewai">CrewAI</SelectItem>
                        <SelectItem value="custom">Custom</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label>Language</Label>
                    <Select value={manifest.language} onValueChange={(v) => updateManifest("language", v)}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="python">Python</SelectItem>
                        <SelectItem value="typescript">TypeScript</SelectItem>
                        <SelectItem value="go">Go</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label>Model Provider</Label>
                    <Select value={manifest.modelProvider} onValueChange={(v) => updateManifest("modelProvider", v)}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="anthropic">Anthropic</SelectItem>
                        <SelectItem value="openai">OpenAI</SelectItem>
                        <SelectItem value="google">Google</SelectItem>
                        <SelectItem value="ollama">Ollama</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label>Model Name</Label>
                    <Input
                      placeholder="claude-sonnet-4"
                      value={manifest.modelName}
                      onChange={(e) => updateManifest("modelName", e.target.value)}
                    />
                  </div>
                </div>
              </div>
            )}

            {/* Skill-specific fields */}
            {manifest.kind === "skill" && (
              <div className="space-y-4">
                <h4 className="font-medium">Skill Configuration</h4>
                <div className="space-y-2">
                  <Label>Category</Label>
                  <Select value={manifest.category} onValueChange={(v) => updateManifest("category", v)}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="general">General</SelectItem>
                      <SelectItem value="coding">Coding</SelectItem>
                      <SelectItem value="infrastructure">Infrastructure</SelectItem>
                      <SelectItem value="data">Data</SelectItem>
                      <SelectItem value="security">Security</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <p className="text-xs text-muted-foreground">
                  Skills are typically defined as SKILL.md files in your repository describing the skill&apos;s capabilities.
                </p>
              </div>
            )}

            <div className="flex justify-end gap-2 pt-4">
              <Button variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              <Button
                onClick={() => setStep("generate")}
                disabled={
                  !manifest.name ||
                  (manifest.kind !== "skill" && !manifest.packageIdentifier)
                }
              >
                Generate Manifest
              </Button>
            </div>
          </div>
        )}

        {step === "generate" && (
          <div className="space-y-4 py-4">
            <Card className="p-4 bg-muted">
              <pre className="text-sm overflow-x-auto whitespace-pre-wrap font-mono">
                {generateYaml()}
              </pre>
            </Card>

            <div className="flex gap-2">
              <Button variant="outline" onClick={copyToClipboard} className="flex-1">
                {copied ? <Check className="h-4 w-4 mr-2" /> : <Copy className="h-4 w-4 mr-2" />}
                {copied ? "Copied!" : "Copy to Clipboard"}
              </Button>
              <Button variant="outline" onClick={downloadFile} className="flex-1">
                <Download className="h-4 w-4 mr-2" />
                Download File
              </Button>
            </div>

            <div className="bg-blue-500/10 border border-blue-500/20 rounded-lg p-4 text-sm">
              <p className="font-medium text-blue-600 dark:text-blue-400 mb-2">Next Steps:</p>
              <ol className="list-decimal list-inside space-y-1 text-muted-foreground">
                <li>Save this file as <code className="bg-muted px-1 rounded">.agentregistry.yaml</code> in your repository root</li>
                <li>Commit and push to your repository</li>
                <li>Click &ldquo;Continue to Apply&rdquo; below</li>
              </ol>
            </div>

            <div className="flex justify-between pt-4">
              <Button variant="ghost" onClick={() => setStep("form")}>
                Back to Form
              </Button>
              <Button onClick={() => setStep("apply")}>
                Continue to Apply
              </Button>
            </div>
          </div>
        )}

        {step === "apply" && (
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>Repository URL</Label>
              <div className="flex gap-2">
                <div className="relative flex-1">
                  <GitBranch className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="https://github.com/org/repo"
                    value={repoUrl}
                    onChange={(e) => setRepoUrl(e.target.value)}
                    className="pl-10"
                  />
                </div>
              </div>
              <p className="text-xs text-muted-foreground">
                Enter the URL of your repository containing the .agentregistry.yaml file
              </p>
            </div>

            <div className="bg-yellow-500/10 border border-yellow-500/20 rounded-lg p-4 text-sm">
              <p className="font-medium text-yellow-600 dark:text-yellow-400 mb-2">Before applying:</p>
              <ul className="list-disc list-inside space-y-1 text-muted-foreground">
                <li>Make sure <code className="bg-muted px-1 rounded">.agentregistry.yaml</code> is committed</li>
                <li>Repository must be public (or provide access token)</li>
                <li>Your submission will be reviewed before publishing</li>
              </ul>
            </div>

            <div className="flex justify-between pt-4">
              <Button variant="ghost" onClick={() => setStep("generate")}>
                Back
              </Button>
              <Button onClick={handleApply} disabled={applying || !repoUrl.trim()}>
                {applying && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
                Submit for Review
              </Button>
            </div>
          </div>
        )}
      </DialogContent>
      </Dialog>
    </>
  )
}
