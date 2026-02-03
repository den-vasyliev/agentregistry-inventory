"use client"

import { useState } from "react"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { adminApiClient, ModelJSON } from "@/lib/admin-api"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

interface AddModelDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onModelAdded: () => void
}

export function AddModelDialog({ open, onOpenChange, onModelAdded }: AddModelDialogProps) {
  const [name, setName] = useState("")
  const [provider, setProvider] = useState("")
  const [model, setModel] = useState("")
  const [baseUrl, setBaseUrl] = useState("")
  const [description, setDescription] = useState("")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      // Validate required fields
      if (!name.trim()) {
        throw new Error("Model name is required")
      }
      if (!provider.trim()) {
        throw new Error("Provider is required")
      }
      if (!model.trim()) {
        throw new Error("Model identifier is required")
      }

      // Construct the ModelJSON object
      const modelData: ModelJSON = {
        name: name.trim(),
        provider: provider.trim(),
        model: model.trim(),
        baseUrl: baseUrl.trim() || undefined,
        description: description.trim() || undefined,
      }

      // Create the model
      await adminApiClient.createModel(modelData)

      // Reset form
      setName("")
      setProvider("")
      setModel("")
      setBaseUrl("")
      setDescription("")

      // Notify parent and close dialog
      onModelAdded()
      onOpenChange(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to add model")
    } finally {
      setLoading(false)
    }
  }

  const handleCancel = () => {
    setName("")
    setProvider("")
    setModel("")
    setBaseUrl("")
    setDescription("")
    setError(null)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Add Model</DialogTitle>
          <DialogDescription>
            Add a new model to the inventory
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="name">
              Model Name <span className="text-red-500">*</span>
            </Label>
            <Input
              id="name"
              placeholder="my-model"
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={loading}
              required
            />
            <p className="text-xs text-muted-foreground">
              A unique name for this model configuration
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="provider">
              Provider <span className="text-red-500">*</span>
            </Label>
            <Select value={provider} onValueChange={setProvider} disabled={loading}>
              <SelectTrigger id="provider">
                <SelectValue placeholder="Select a provider" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="openai">OpenAI</SelectItem>
                <SelectItem value="anthropic">Anthropic</SelectItem>
                <SelectItem value="ollama">Ollama</SelectItem>
                <SelectItem value="azure-openai">Azure OpenAI</SelectItem>
                <SelectItem value="google">Google</SelectItem>
                <SelectItem value="cohere">Cohere</SelectItem>
                <SelectItem value="custom">Custom</SelectItem>
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              The model provider (e.g., OpenAI, Anthropic)
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="model">
              Model Identifier <span className="text-red-500">*</span>
            </Label>
            <Input
              id="model"
              placeholder="gpt-4, claude-3-opus, llama3:70b"
              value={model}
              onChange={(e) => setModel(e.target.value)}
              disabled={loading}
              required
            />
            <p className="text-xs text-muted-foreground">
              The specific model identifier used by the provider
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="baseUrl">
              Base URL
            </Label>
            <Input
              id="baseUrl"
              placeholder="https://api.openai.com/v1"
              value={baseUrl}
              onChange={(e) => setBaseUrl(e.target.value)}
              disabled={loading}
              type="url"
            />
            <p className="text-xs text-muted-foreground">
              Optional: Custom base URL for the model API endpoint
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="description">
              Description
            </Label>
            <Textarea
              id="description"
              placeholder="A description of this model configuration"
              rows={3}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              disabled={loading}
            />
            <p className="text-xs text-muted-foreground">
              Optional: Additional details about this model
            </p>
          </div>

          {error && (
            <div className="rounded-md bg-red-50 p-3 text-sm text-red-800">
              {error}
            </div>
          )}

          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={handleCancel} disabled={loading}>
              Cancel
            </Button>
            <Button type="submit" disabled={loading}>
              {loading ? "Adding..." : "Add Model"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
