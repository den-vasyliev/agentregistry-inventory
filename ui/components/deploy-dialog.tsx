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

const AGENT_SANDBOX_ENVIRONMENT = "agent-sandbox"

interface DeployDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  itemName?: string
  itemVersion?: string
  itemType?: 'server' | 'agent'
  deploying: boolean
  onConfirm: () => void
  environments: Array<{ name: string; cluster: string; provider?: string; region?: string; namespace: string; deployEnabled: boolean }>
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
          <DialogTitle>Sandbox Resource</DialogTitle>
          <DialogDescription>
            Sandbox <strong>{itemName}</strong> (version {itemVersion})?
            <br />
            <br />
            This will start the {itemType === 'server' ? 'MCP server' : 'agent'} in the sandbox environment.
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
                {environments.map((env) => {
                  const selectable = env.name === AGENT_SANDBOX_ENVIRONMENT && env.deployEnabled
                  return (
                    <SelectItem key={env.name} value={env.name} disabled={!selectable}>
                      <span className="flex items-center gap-1.5">
                        {env.provider && (
                          <span className="inline-flex items-center rounded bg-muted px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wider">
                            {env.provider}
                          </span>
                        )}
                        <span>{env.name}</span>
                        {env.cluster && <span className="text-muted-foreground">· {env.cluster}</span>}
                        {env.region && <span className="text-muted-foreground text-xs">({env.region})</span>}
                      </span>
                    </SelectItem>
                  )
                })}
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              Environments are loaded from DiscoveryConfig. Only <code>agent-sandbox</code> can be selected.
            </p>
          </div>
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
            disabled={deploying || deployEnvironment !== AGENT_SANDBOX_ENVIRONMENT}
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
