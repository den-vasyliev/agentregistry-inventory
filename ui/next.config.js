/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'export',
  distDir: '../internal/api/ui/dist',
  images: {
    unoptimized: true,
  },
  trailingSlash: true,
}

module.exports = nextConfig

