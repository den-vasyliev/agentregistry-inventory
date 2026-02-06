"use client"

import { useEffect, useState } from "react"
import { adminApiClient } from "@/lib/admin-api"

export function BuildVersion() {
  const [version, setVersion] = useState<string | null>(null)

  useEffect(() => {
    adminApiClient
      .getVersion()
      .then((v) => {
        const parts = []
        if (v.version) parts.push(v.version)
        if (v.commit && v.commit !== "unknown") parts.push(v.commit.slice(0, 7))
        setVersion(parts.join("+") || null)
      })
      .catch(() => {
        // silently ignore — version display is best-effort
      })
  }, [])

  return (
    <div className="fixed bottom-2 right-2 text-xs text-muted-foreground/50">
      {version ? `build ${version}` : ""}
    </div>
  )
}
