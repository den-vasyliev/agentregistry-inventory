"use client"

import { useState, useEffect } from "react"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { GroupedMCPServer, MCPServerWithStatus, Package } from "@/lib/types"
import { Eye, EyeOff, AlertCircle } from "lucide-react"

interface InstallDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  groupedServer: GroupedMCPServer
  onConfirm: (server: MCPServerWithStatus, config: Record<string, string>) => void
}

export function InstallDialog({
  open,
  onOpenChange,
  groupedServer,
  onConfirm,
}: InstallDialogProps) {
  const [selectedVersionIndex, setSelectedVersionIndex] = useState(0)
  const [selectedPackage, setSelectedPackage] = useState(0)
  const [config, setConfig] = useState<Record<string, string>>({})
  const [showSecrets, setShowSecrets] = useState<Record<string, boolean>>({})
  
  const selectedServer = groupedServer.versions[selectedVersionIndex]

  const packages = selectedServer.server.packages || []
  const currentPackage = packages[selectedPackage]
  
  // Reset config and package selection when version changes
  useEffect(() => {
    setConfig({})
    setSelectedPackage(0)
  }, [selectedVersionIndex])

  const handleSubmit = () => {
    onConfirm(selectedServer, config)
    onOpenChange(false)
  }

  const updateConfig = (key: string, value: string) => {
    setConfig((prev) => ({ ...prev, [key]: value }))
  }

  const toggleShowSecret = (key: string) => {
    setShowSecrets((prev) => ({ ...prev, [key]: !prev[key] }))
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-w-3xl max-h-[85vh]"
        onClose={() => onOpenChange(false)}
      >
        <DialogHeader>
          <DialogTitle>
            Install {groupedServer.title || groupedServer.name}
          </DialogTitle>
          <DialogDescription>
            Configure the MCP server before installation. Fill in required
            fields marked with *.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {/* Version Selection */}
          {groupedServer.versions.length > 1 && (
            <div>
              <Label className="mb-2 block">Select Version</Label>
              <select
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                value={selectedVersionIndex}
                onChange={(e) => setSelectedVersionIndex(parseInt(e.target.value))}
              >
                {groupedServer.versions.map((version, idx) => (
                  <option key={idx} value={idx}>
                    v{version.server.version}
                    {version._meta?.["io.modelcontextprotocol.registry/official"]?.isLatest && " (Latest)"}
                    {version.installed && " (Installed)"}
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Package Selection */}
          {packages.length > 1 && (
            <div>
              <Label className="mb-2 block">Select Package</Label>
              <Tabs
                value={selectedPackage.toString()}
                onValueChange={(v) => setSelectedPackage(parseInt(v))}
              >
                <TabsList>
                  {packages.map((pkg, idx) => (
                    <TabsTrigger key={idx} value={idx.toString()}>
                      {pkg.registryType} - {pkg.identifier}
                    </TabsTrigger>
                  ))}
                </TabsList>
              </Tabs>
            </div>
          )}

          {currentPackage && (
            <div className="space-y-4">
              {/* Package Info */}
              <div className="p-4 bg-muted rounded-lg space-y-2">
                <div className="flex items-center gap-2">
                  <Badge variant="secondary">{currentPackage.registryType}</Badge>
                  <Badge variant="outline">v{currentPackage.version}</Badge>
                </div>
                <p className="text-sm text-muted-foreground">
                  {currentPackage.identifier}
                </p>
                {currentPackage.runtimeHint && (
                  <p className="text-xs text-muted-foreground">
                    Runtime: {currentPackage.runtimeHint}
                  </p>
                )}
              </div>

              {/* Environment Variables */}
              {currentPackage.environmentVariables &&
                currentPackage.environmentVariables.length > 0 && (
                  <div>
                    <h4 className="font-semibold mb-3">Environment Variables</h4>
                    <div className="space-y-3">
                      {currentPackage.environmentVariables.map((envVar) => (
                        <div key={envVar.name} className="space-y-2">
                          <Label htmlFor={envVar.name}>
                            {envVar.name}
                            {envVar.isRequired && (
                              <span className="text-destructive ml-1">*</span>
                            )}
                            {envVar.isSecret && (
                              <Badge
                                variant="outline"
                                className="ml-2 text-xs"
                              >
                                Secret
                              </Badge>
                            )}
                          </Label>
                          {envVar.description && (
                            <p className="text-xs text-muted-foreground">
                              {envVar.description}
                            </p>
                          )}
                          <div className="relative">
                            {envVar.choices && envVar.choices.length > 0 ? (
                              <select
                                id={envVar.name}
                                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                                value={
                                  config[envVar.name] ||
                                  envVar.default ||
                                  ""
                                }
                                onChange={(e) =>
                                  updateConfig(envVar.name, e.target.value)
                                }
                              >
                                <option value="">Select...</option>
                                {envVar.choices.map((choice) => (
                                  <option key={choice} value={choice}>
                                    {choice}
                                  </option>
                                ))}
                              </select>
                            ) : (
                              <Input
                                id={envVar.name}
                                type={
                                  envVar.isSecret &&
                                  !showSecrets[envVar.name]
                                    ? "password"
                                    : "text"
                                }
                                placeholder={
                                  envVar.placeholder || envVar.default
                                }
                                value={
                                  config[envVar.name] ||
                                  envVar.default ||
                                  ""
                                }
                                onChange={(e) =>
                                  updateConfig(envVar.name, e.target.value)
                                }
                                required={envVar.isRequired}
                              />
                            )}
                            {envVar.isSecret && (
                              <button
                                type="button"
                                onClick={() => toggleShowSecret(envVar.name)}
                                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                              >
                                {showSecrets[envVar.name] ? (
                                  <EyeOff className="w-4 h-4" />
                                ) : (
                                  <Eye className="w-4 h-4" />
                                )}
                              </button>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

              {/* Runtime Arguments */}
              {currentPackage.runtimeArguments &&
                currentPackage.runtimeArguments.length > 0 && (
                  <div>
                    <h4 className="font-semibold mb-3">Runtime Arguments</h4>
                    <div className="space-y-3">
                      {currentPackage.runtimeArguments.map((arg, idx) => (
                        <div key={`${arg.name}-${idx}`} className="space-y-2">
                          <Label htmlFor={`runtime-${arg.name}-${idx}`}>
                            {arg.name}
                            {arg.isRequired && (
                              <span className="text-destructive ml-1">*</span>
                            )}
                            {arg.type && (
                              <Badge
                                variant="secondary"
                                className="ml-2 text-xs"
                              >
                                {arg.type}
                              </Badge>
                            )}
                          </Label>
                          {arg.description && (
                            <p className="text-xs text-muted-foreground">
                              {arg.description}
                            </p>
                          )}
                          <Input
                            id={`runtime-${arg.name}-${idx}`}
                            type="text"
                            placeholder={arg.placeholder || arg.default}
                            value={
                              config[`runtime-${arg.name}`] ||
                              arg.default ||
                              ""
                            }
                            onChange={(e) =>
                              updateConfig(
                                `runtime-${arg.name}`,
                                e.target.value
                              )
                            }
                            required={arg.isRequired}
                          />
                        </div>
                      ))}
                    </div>
                  </div>
                )}

              {/* Package Arguments */}
              {currentPackage.packageArguments &&
                currentPackage.packageArguments.length > 0 && (
                  <div>
                    <h4 className="font-semibold mb-3">Package Arguments</h4>
                    <div className="space-y-3">
                      {currentPackage.packageArguments.map((arg, idx) => (
                        <div key={`${arg.name}-${idx}`} className="space-y-2">
                          <Label htmlFor={`package-${arg.name}-${idx}`}>
                            {arg.name}
                            {arg.isRequired && (
                              <span className="text-destructive ml-1">*</span>
                            )}
                          </Label>
                          {arg.description && (
                            <p className="text-xs text-muted-foreground">
                              {arg.description}
                            </p>
                          )}
                          <Input
                            id={`package-${arg.name}-${idx}`}
                            type="text"
                            placeholder={arg.placeholder || arg.default}
                            value={
                              config[`package-${arg.name}`] ||
                              arg.default ||
                              ""
                            }
                            onChange={(e) =>
                              updateConfig(
                                `package-${arg.name}`,
                                e.target.value
                              )
                            }
                            required={arg.isRequired}
                          />
                        </div>
                      ))}
                    </div>
                  </div>
                )}

              {/* Transport Configuration */}
              {currentPackage.transport && (
                <div>
                  <h4 className="font-semibold mb-3">Transport</h4>
                  <div className="p-4 bg-muted rounded-lg space-y-2">
                    <div className="flex items-center gap-2">
                      <Badge>{currentPackage.transport.type}</Badge>
                      {currentPackage.transport.url && (
                        <span className="text-sm text-muted-foreground">
                          {currentPackage.transport.url}
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Warning for required fields */}
          <div className="flex items-start gap-2 p-3 bg-yellow-500/10 border border-yellow-500/20 rounded-lg">
            <AlertCircle className="w-5 h-5 text-yellow-600 flex-shrink-0 mt-0.5" />
            <p className="text-sm text-yellow-600">
              Make sure to fill in all required fields before installing. Secret
              values will be stored securely.
            </p>
          </div>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-3 pt-4 border-t">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSubmit}>Install Server</Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}

