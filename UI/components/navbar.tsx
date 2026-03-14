"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import { useAuth } from "@/lib/auth-context"
import { Button } from "@/components/ui/button"
import { ShieldCheck, LogOut } from "lucide-react"

const navLinks = [
  { label: "Analisar", href: "/" },
  { label: "Contratos", href: "/contracts" },
]

function isLinkActive(href: string, pathname: string): boolean {
  if (href === "/") return pathname === "/"
  return pathname === href || pathname.startsWith(href + "/")
}

export function Navbar() {
  const { user, logout } = useAuth()
  const pathname = usePathname()

  return (
    <nav className="flex items-center justify-between py-5">
      <div className="flex items-center gap-6">
        <Link href="/" className="flex items-center gap-2">
          <ShieldCheck className="h-6 w-6 text-accent" />
          <span className="text-lg font-semibold tracking-tight text-foreground">
            RiskLens
          </span>
        </Link>

        <div className="flex items-center gap-1">
          {navLinks.map((link) => {
            const active = isLinkActive(link.href, pathname)
            return (
              <Link
                key={link.href}
                href={link.href}
                className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                  active
                    ? "text-foreground underline underline-offset-4"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {link.label}
              </Link>
            )
          })}
        </div>
      </div>

      {user ? (
        <div className="flex items-center gap-3">
          <span className="text-sm text-muted-foreground">
            {user.username}
          </span>
          <Button
            variant="outline"
            size="sm"
            className="text-sm"
            onClick={logout}
          >
            <LogOut className="mr-1 h-4 w-4" />
            Sair
          </Button>
        </div>
      ) : (
        <Link href="/login">
          <Button variant="outline" size="sm" className="text-sm">
            Entrar
          </Button>
        </Link>
      )}
    </nav>
  )
}
