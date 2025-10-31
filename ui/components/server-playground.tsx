"use client"

import { useState } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { MCPServerWithStatus, MCPTool } from "@/lib/types"
import { Play, RotateCw, AlertCircle, CheckCircle2 } from "lucide-react"

interface ServerPlaygroundProps {
  server: MCPServerWithStatus
}

// Mock tools for demonstration
const getMockTools = (): MCPTool[] => {
  return [
    {
      name: "get_weather",
      description: "Get current weather for a location",
      inputSchema: {
        type: "object",
        properties: {
          location: {
            type: "string",
            description: "City name or zip code",
          },
          units: {
            type: "string",
            enum: ["celsius", "fahrenheit"],
            default: "celsius",
          },
        },
        required: ["location"],
      },
    },
    {
      name: "get_forecast",
      description: "Get weather forecast for the next 7 days",
      inputSchema: {
        type: "object",
        properties: {
          location: {
            type: "string",
            description: "City name or zip code",
          },
          days: {
            type: "number",
            description: "Number of days (1-7)",
            minimum: 1,
            maximum: 7,
            default: 5,
          },
        },
        required: ["location"],
      },
    },
  ]
}

export function ServerPlayground({ }: ServerPlaygroundProps) {
  const [selectedTool, setSelectedTool] = useState<MCPTool | null>(null)
  const [inputs, setInputs] = useState<Record<string, string | number>>({})
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<{ success: boolean; data: { tool: string; inputs: Record<string, string | number>; output: string; executedAt: string } } | null>(null)
  const [error, setError] = useState<string | null>(null)

  const tools = getMockTools()

  const handleToolSelect = (tool: MCPTool) => {
    setSelectedTool(tool)
    setInputs({})
    setResult(null)
    setError(null)
  }

  const handleInputChange = (key: string, value: string | number) => {
    setInputs((prev) => ({ ...prev, [key]: value }))
  }

  const handleExecute = async () => {
    if (!selectedTool) return

    setLoading(true)
    setError(null)
    setResult(null)

    // Simulate API call
    setTimeout(() => {
      // Mock successful response
      setResult({
        success: true,
        data: {
          tool: selectedTool.name,
          inputs: inputs,
          output: "Mock response - Replace with actual tool execution",
          executedAt: new Date().toISOString(),
        },
      })
      setLoading(false)
    }, 1000)
  }

  const renderInputField = (
    key: string,
    schema: { type?: string; enum?: string[]; default?: string | number; description?: string; minimum?: number; maximum?: number },
    required: boolean
  ) => {
    if (schema.enum) {
      return (
        <div key={key} className="space-y-2">
          <Label htmlFor={key}>
            {key}
            {required && <span className="text-destructive ml-1">*</span>}
          </Label>
          {schema.description && (
            <p className="text-xs text-muted-foreground">{schema.description}</p>
          )}
          <select
            id={key}
            className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            value={inputs[key] || schema.default || ""}
            onChange={(e) => handleInputChange(key, e.target.value)}
          >
            <option value="">Select...</option>
            {schema.enum.map((option: string) => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
        </div>
      )
    }

    if (schema.type === "number") {
      return (
        <div key={key} className="space-y-2">
          <Label htmlFor={key}>
            {key}
            {required && <span className="text-destructive ml-1">*</span>}
          </Label>
          {schema.description && (
            <p className="text-xs text-muted-foreground">{schema.description}</p>
          )}
          <Input
            id={key}
            type="number"
            min={schema.minimum}
            max={schema.maximum}
            placeholder={schema.default?.toString()}
            value={inputs[key] || ""}
            onChange={(e) => handleInputChange(key, parseFloat(e.target.value))}
            required={required}
          />
        </div>
      )
    }

    return (
      <div key={key} className="space-y-2">
        <Label htmlFor={key}>
          {key}
          {required && <span className="text-destructive ml-1">*</span>}
        </Label>
        {schema.description && (
          <p className="text-xs text-muted-foreground">{schema.description}</p>
        )}
        <Input
          id={key}
          type="text"
          placeholder={schema.default?.toString()}
          value={inputs[key] || ""}
          onChange={(e) => handleInputChange(key, e.target.value)}
          required={required}
        />
      </div>
    )
  }

  return (
    <div className="grid lg:grid-cols-3 gap-6">
      {/* Tools List */}
      <Card className="p-6 lg:col-span-1">
        <h3 className="font-semibold mb-4">Available Tools</h3>
        {tools.length === 0 ? (
          <p className="text-sm text-muted-foreground">No tools available</p>
        ) : (
          <div className="space-y-2">
            {tools.map((tool) => (
              <button
                key={tool.name}
                onClick={() => handleToolSelect(tool)}
                className={`w-full text-left p-3 rounded-lg transition-colors ${
                  selectedTool?.name === tool.name
                    ? "bg-primary text-primary-foreground"
                    : "bg-muted hover:bg-muted/80"
                }`}
              >
                <div className="font-medium text-sm">{tool.name}</div>
                <p className="text-xs opacity-80 line-clamp-2 mt-1">
                  {tool.description}
                </p>
              </button>
            ))}
          </div>
        )}
      </Card>

      {/* Tool Execution */}
      <Card className="p-6 lg:col-span-2">
        {!selectedTool ? (
          <div className="flex items-center justify-center h-full min-h-[300px]">
            <div className="text-center text-muted-foreground">
              <Play className="w-12 h-12 mx-auto mb-4 opacity-50" />
              <p>Select a tool to test</p>
            </div>
          </div>
        ) : (
          <div className="space-y-6">
            <div>
              <h3 className="text-lg font-semibold mb-2">{selectedTool.name}</h3>
              <p className="text-sm text-muted-foreground">
                {selectedTool.description}
              </p>
            </div>

            {/* Input Fields */}
            <div className="space-y-4">
              <h4 className="font-medium">Parameters</h4>
              {Object.entries(selectedTool.inputSchema.properties || {}).map(
                ([key, schema]) => {
                  const required = selectedTool.inputSchema.required?.includes(key) ?? false
                  return renderInputField(key, schema as { type?: string; enum?: string[]; default?: string | number; description?: string; minimum?: number; maximum?: number }, required)
                }
              )}
            </div>

            {/* Execute Button */}
            <div className="flex gap-3">
              <Button
                onClick={handleExecute}
                disabled={loading}
                className="flex-1"
              >
                {loading ? (
                  <>
                    <RotateCw className="w-4 h-4 mr-2 animate-spin" />
                    Executing...
                  </>
                ) : (
                  <>
                    <Play className="w-4 h-4 mr-2" />
                    Execute Tool
                  </>
                )}
              </Button>
              <Button
                variant="outline"
                onClick={() => {
                  setInputs({})
                  setResult(null)
                  setError(null)
                }}
              >
                Reset
              </Button>
            </div>

            {/* Result */}
            {error && (
              <div className="p-4 bg-destructive/10 border border-destructive rounded-lg">
                <div className="flex items-start gap-2">
                  <AlertCircle className="w-5 h-5 text-destructive flex-shrink-0 mt-0.5" />
                  <div>
                    <p className="font-medium text-destructive">Error</p>
                    <p className="text-sm text-destructive/90">{error}</p>
                  </div>
                </div>
              </div>
            )}

            {result && (
              <div className="p-4 bg-green-500/10 border border-green-500/20 rounded-lg">
                <div className="flex items-start gap-2 mb-3">
                  <CheckCircle2 className="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
                  <div>
                    <p className="font-medium text-green-600">Success</p>
                    <p className="text-xs text-green-600/80">
                      Executed at {new Date(result.data.executedAt).toLocaleTimeString()}
                    </p>
                  </div>
                </div>
                <div className="bg-background rounded-md p-4">
                  <pre className="text-xs overflow-x-auto">
                    {JSON.stringify(result, null, 2)}
                  </pre>
                </div>
              </div>
            )}
          </div>
        )}
      </Card>
    </div>
  )
}

