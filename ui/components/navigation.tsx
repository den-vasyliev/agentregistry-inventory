"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import { useSession } from "next-auth/react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { LogIn, LogOut, User } from "lucide-react"

export function Navigation() {
  const pathname = usePathname()
  const { data: session, status } = useSession()

  const isActive = (path: string) => {
    if (path === "/") {
      return pathname === "/"
    }
    return pathname.startsWith(path)
  }

  const getLinkClasses = (path: string) => {
    const baseClasses = "text-sm font-medium transition-colors"
    if (isActive(path)) {
      return `${baseClasses} text-foreground hover:text-foreground/80 border-b-2 border-foreground pb-1`
    }
    return `${baseClasses} text-muted-foreground hover:text-foreground`
  }

  return (
    <nav className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 sticky top-0 z-50">
      <div className="container mx-auto px-6">
        <div className="flex items-center justify-between h-20">
          <Link href="/" className="flex items-center">
            <img
              src="/arlogo.png"
              alt="Agent Inventory"
              width={180}
              height={60}
              className="h-12 w-auto"
            />
          </Link>

          <div className="flex items-center gap-6">
            <Link
              href="/"
              className={getLinkClasses("/")}
            >
              Inventory
            </Link>
            <Link
              href="/published"
              className={getLinkClasses("/published")}
            >
              Published
            </Link>
            <Link
              href="/deployed"
              className={getLinkClasses("/deployed")}
            >
              Deployed
            </Link>

            {/* Auth Section */}
            {status === "loading" ? (
              <div className="w-24 h-10 bg-muted animate-pulse rounded-md" />
            ) : session ? (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="sm" className="gap-2">
                    <User className="h-4 w-4" />
                    <span className="hidden sm:inline">{session.user?.name || session.user?.email}</span>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem disabled className="text-xs text-muted-foreground">
                    {session.user?.email}
                  </DropdownMenuItem>
                  <DropdownMenuItem>
                    <Link href="/api/auth/signout" className="cursor-pointer flex items-center">
                      <LogOut className="h-4 w-4 mr-2" />
                      Sign Out
                    </Link>
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            ) : (
              <Link href="/auth/signin">
                <Button variant="default" size="sm" className="gap-2">
                  <LogIn className="h-4 w-4" />
                  Sign In
                </Button>
              </Link>
            )}
          </div>
        </div>
      </div>
    </nav>
  )
}
