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
  environments: Array<{ name: string; namespace: string }>
  loadingEnvironments: boolean
  deployNamespace: string
  onNamespaceChange: (ns: string) => void
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
  onNamespaceChange,
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
            <Label>Deployment destination</Label>
            <p className="text-sm text-muted-foreground">Kubernetes</p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="deploy-namespace">Namespace / Environment</Label>
            <Select value={deployNamespace} onValueChange={onNamespaceChange} disabled={loadingEnvironments}>
              <SelectTrigger id="deploy-namespace">
                <SelectValue placeholder={loadingEnvironments ? "Loading..." : "Select namespace"} />
              </SelectTrigger>
              <SelectContent>
                {environments.map((env) => (
                  <SelectItem key={env.namespace} value={env.namespace}>
                    {env.name}/{env.namespace}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              Choose which namespace/environment to deploy to
            </p>
          </div>
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
