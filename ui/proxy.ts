import { auth } from "@/auth"

export default auth((req) => {
  // For now, just let NextAuth handle its own routes
  // API protection is handled client-side and by backend
  return undefined
})

export const config = {
  // Only run on auth routes
  matcher: ["/api/auth/:path*"],
}
