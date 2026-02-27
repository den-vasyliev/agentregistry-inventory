import type { Metadata } from "next"
import { Inter } from "next/font/google"
import Script from "next/script"
import { Navigation } from "@/components/navigation"
import { SessionProvider } from "@/components/session-provider"
import { BuildVersion } from "@/components/build-version"
import { Toaster } from "@/components/ui/sonner"
import "./globals.css"

const inter = Inter({ subsets: ["latin"], variable: "--font-inter" })

export const metadata: Metadata = {
  title: "artcl - agent inventory",
  description: "Browse and manage MCP servers, agents, and skills",
  icons: {
    icon: "/icon.svg",
  },
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="en" className="dark">
      {/* Load runtime MSAL config before app bundle. Served by Go binary at /ui/config.js.
          The inline fallback ensures window.__APP_CONFIG__ always exists so MSAL
          doesn't throw before the external script executes. */}
      <head>
        <Script
          id="app-config-fallback"
          strategy="beforeInteractive"
          dangerouslySetInnerHTML={{ __html: "window.__APP_CONFIG__ = window.__APP_CONFIG__ || {};" }}
        />
        <Script src="/config.js" strategy="beforeInteractive" />
      </head>
      <body className={`${inter.variable} font-sans`}>
        <SessionProvider>
          <Navigation />
          {children}
          <Toaster />
          <BuildVersion />
        </SessionProvider>
      </body>
    </html>
  )
}

