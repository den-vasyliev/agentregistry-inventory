"use client"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"

interface DeployDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  itemName?: string
  itemVersion?: string
  itemType?: 'server' | 'agent'
  deploying: boolean
  onConfirm: () => void
  deployNamespace: string
  deployEnvironment: string
}

export function DeployDialog({
  open,
  onOpenChange,
  itemName,
  itemVersion,
  itemType,
  deploying,
  onConfirm,
  deployNamespace,
  deployEnvironment,
}: DeployDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Sandbox Resource</DialogTitle>
          <DialogDescription>
            Sandbox <strong>{itemName}</strong> (version {itemVersion})?
            <br />
            <br />
            This will start the {itemType === 'server' ? 'MCP server' : 'agent'} in the sandbox environment.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="rounded-md border px-3 py-2 text-sm space-y-1">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Environment</span>
              <span className="font-mono">{deployEnvironment}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Namespace</span>
              <span className="font-mono">{deployNamespace}</span>
            </div>
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
            {deploying ? 'Sandboxing...' : 'Sandbox'}
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
