import { NextRequest, NextResponse } from "next/server"
import { auth } from "@/auth"

const controllerBaseUrl =
  process.env.AGENTREGISTRY_CONTROLLER_URL ||
  process.env.NEXT_PUBLIC_API_URL ||
  "http://localhost:8080"

const adminGroup = process.env.AGENTREGISTRY_OIDC_ADMIN_GROUP

function isDeployWriteRequest(url: URL, method: string): boolean {
  if (!url.pathname.startsWith("/admin/v0/deployments")) {
    return false
  }
  return method === "POST" || method === "PATCH" || method === "DELETE"
}

async function handle(req: NextRequest, params: { path: string[] }) {
  const disableAuth = process.env.NEXT_PUBLIC_DISABLE_AUTH !== "false"
  const session = disableAuth ? null : await auth()
  if (!disableAuth && !session) {
    return NextResponse.json({ message: "Unauthorized" }, { status: 401 })
  }

  const path = params.path.join("/")
  const url = new URL(req.url)
  const targetUrl = `${controllerBaseUrl}/${path}${url.search}`

  if (!disableAuth && adminGroup && isDeployWriteRequest(new URL(targetUrl), req.method)) {
    const groups = (session?.user as { groups?: string[] } | undefined)?.groups || []
    if (!groups.includes(adminGroup)) {
      return NextResponse.json({ message: "Forbidden" }, { status: 403 })
    }
  }

  const headers = new Headers()
  if (!disableAuth && session) {
    const authSession = session as { accessToken?: string; idToken?: string }
    const bearerToken = authSession.idToken || authSession.accessToken
    if (bearerToken) {
      headers.set("Authorization", `Bearer ${bearerToken}`)
    }
  }
  const contentType = req.headers.get("content-type")
  if (contentType) {
    headers.set("Content-Type", contentType)
  }
  const accept = req.headers.get("accept")
  if (accept) {
    headers.set("Accept", accept)
  }

  const init: RequestInit = {
    method: req.method,
    headers,
  }

  if (req.method !== "GET" && req.method !== "HEAD") {
    init.body = await req.text()
  }

  const resp = await fetch(targetUrl, init)
  const respBody = await resp.text()

  return new NextResponse(respBody, {
    status: resp.status,
    headers: {
      "Content-Type": resp.headers.get("content-type") || "application/json",
    },
  })
}

export async function GET(req: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const params = await context.params
  return handle(req, params)
}

export async function POST(req: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const params = await context.params
  return handle(req, params)
}

export async function PATCH(req: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const params = await context.params
  return handle(req, params)
}

export async function DELETE(req: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const params = await context.params
  return handle(req, params)
}
