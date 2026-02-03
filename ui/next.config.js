/** @type {import('next').NextConfig} */
const nextConfig = {
  images: {
    unoptimized: true,
  },
  async rewrites() {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
    return [
      {
        source: '/admin/:path*',
        destination: `${apiUrl}/admin/:path*`,
      },
      {
        source: '/v0/:path*',
        destination: `${apiUrl}/v0/:path*`,
      },
      {
        source: '/v0.1/:path*',
        destination: `${apiUrl}/v0.1/:path*`,
      },
      // Don't proxy /api/auth/* - handled by NextAuth
      // Only proxy other /api routes to backend if needed
    ]
  },
}

// Only use static export for production builds
if (process.env.NEXT_BUILD_EXPORT === 'true') {
  nextConfig.output = 'export'
  // Disable trailingSlash for static export to avoid redirect loops
  nextConfig.trailingSlash = false
  // Remove rewrites for static export (not supported)
  delete nextConfig.rewrites
}

module.exports = nextConfig

