"use client"

import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

interface DeployDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  itemName?: string
  itemVersion?: string
  itemType?: 'server' | 'agent'
  deploying: boolean
  onConfirm: () => void
  environments: Array<{ name: string; cluster: string; namespace: string }>
  loadingEnvironments: boolean
  deployNamespace: string
  deployEnvironment: string
  onEnvironmentChange: (envName: string, namespace: string) => void
}

export function DeployDialog({
  open,
  onOpenChange,
  itemName,
  itemVersion,
  itemType,
  deploying,
  onConfirm,
  environments,
  loadingEnvironments,
  deployNamespace,
  deployEnvironment,
  onEnvironmentChange,
}: DeployDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Deploy Resource</DialogTitle>
          <DialogDescription>
            Deploy <strong>{itemName}</strong> (version {itemVersion})?
            <br />
            <br />
            This will start the {itemType === 'server' ? 'MCP server' : 'agent'} on your system.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="deploy-environment">Target environment</Label>
            <Select
              value={deployEnvironment}
              onValueChange={(envName) => {
                const env = environments.find(e => e.name === envName)
                if (env) onEnvironmentChange(env.name, env.namespace)
              }}
              disabled={loadingEnvironments}
            >
              <SelectTrigger id="deploy-environment">
                <SelectValue placeholder={loadingEnvironments ? "Loading..." : "Select environment"} />
              </SelectTrigger>
              <SelectContent>
                {environments.map((env) => (
                  <SelectItem key={env.name} value={env.name}>
                    {env.name}
                    {env.cluster && <span className="text-muted-foreground ml-1">· {env.cluster}</span>}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              Cluster and namespace where the resource will be deployed
            </p>
          </div>
          {deployEnvironment && (
            <div className="rounded-md border px-3 py-2 text-sm space-y-1">
              {(() => {
                const env = environments.find(e => e.name === deployEnvironment)
                return env ? (
                  <>
                    {env.cluster && (
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">Cluster</span>
                        <span className="font-mono">{env.cluster}</span>
                      </div>
                    )}
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Namespace</span>
                      <span className="font-mono">{deployNamespace}</span>
                    </div>
                  </>
                ) : null
              })()}
            </div>
          )}
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={deploying}
          >
            Cancel
          </Button>
          <Button
            variant="default"
            onClick={onConfirm}
            disabled={deploying}
          >
            {deploying ? 'Deploying...' : 'Deploy'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

interface UndeployDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  itemName?: string
  itemVersion?: string
  itemType?: 'server' | 'agent'
  undeploying: boolean
  onConfirm: () => void
}

export function UndeployDialog({
  open,
  onOpenChange,
  itemName,
  itemVersion,
  itemType,
  undeploying,
  onConfirm,
}: UndeployDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Undeploy Resource</DialogTitle>
          <DialogDescription>
            Are you sure you want to undeploy <strong>{itemName}</strong> (version {itemVersion})?
            <br />
            <br />
            This will stop the {itemType === 'server' ? 'MCP server' : 'agent'} and remove it from your deployments.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={undeploying}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={onConfirm}
            disabled={undeploying}
          >
            {undeploying ? 'Undeploying...' : 'Undeploy'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
