"use client"

import { useEffect, useState } from "react"
import { useRouter, useParams } from "next/navigation"
import Link from "next/link"
import { useAuth } from "@/lib/auth-context"
import { listAnalyses, type AnalysisSummary } from "@/lib/api"
import { Navbar } from "@/components/navbar"
import { ResultPreview } from "@/components/result-preview"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { ArrowLeft, Loader2, FileText, AlertTriangle } from "lucide-react"

export default function ContractDetailPage() {
  const { user, isLoading } = useAuth()
  const router = useRouter()
  const params = useParams()
  const id = params.id as string

  const [contract, setContract] = useState<AnalysisSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [notFound, setNotFound] = useState(false)

  useEffect(() => {
    if (!isLoading && !user) {
      router.push("/login")
    }
  }, [user, isLoading, router])

  useEffect(() => {
    if (!user) return

    async function fetchContract() {
      setLoading(true)
      try {
        const analyses = await listAnalyses()
        const found = analyses.find((a) => a.id === id) ?? null
        if (found) {
          setContract(found)
        } else {
          setNotFound(true)
        }
      } catch {
        setNotFound(true)
      } finally {
        setLoading(false)
      }
    }

    fetchContract()
  }, [user, id])

  if (isLoading || !user) return null

  return (
    <div className="min-h-screen bg-background">
      <div className="mx-auto max-w-4xl px-4">
        <Navbar />

        <main className="pb-16 pt-8">
          {/* Back link */}
          <Link
            href="/contracts"
            className="mb-6 inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            <ArrowLeft className="h-4 w-4" />
            Voltar para Contratos
          </Link>

          {/* Loading state */}
          {loading && (
            <div className="mt-12 flex items-center justify-center text-muted-foreground">
              <Loader2 className="mr-2 h-5 w-5 animate-spin" />
              <span className="text-sm">Carregando contrato...</span>
            </div>
          )}

          {/* Not found */}
          {!loading && notFound && (
            <div className="mt-12 flex flex-col items-center gap-3 text-center">
              <FileText className="h-10 w-10 text-muted-foreground/40" />
              <p className="font-medium text-foreground">
                Contrato não encontrado
              </p>
              <p className="text-sm text-muted-foreground">
                O contrato solicitado não existe ou foi removido.
              </p>
              <Link href="/contracts">
                <Button variant="outline" size="sm" className="mt-2">
                  Ver todos os contratos
                </Button>
              </Link>
            </div>
          )}

          {/* Contract not completed */}
          {!loading && contract && contract.status !== "COMPLETED" && (
            <div className="mt-12 flex flex-col items-center gap-3 text-center">
              <AlertTriangle className="h-10 w-10 text-muted-foreground/40" />
              <p className="font-medium text-foreground">
                Análise não disponível
              </p>
              <p className="text-sm text-muted-foreground">
                Este contrato ainda não foi processado ou encontrou um erro
                durante a análise.
              </p>
              <Link href="/contracts">
                <Button variant="outline" size="sm" className="mt-2">
                  Ver todos os contratos
                </Button>
              </Link>
            </div>
          )}

          {/* Result */}
          {!loading && contract && contract.status === "COMPLETED" && (
            <>
              <div className="mb-6">
                <div className="flex items-center gap-2">
                  <FileText className="h-5 w-5 text-muted-foreground" />
                  <h1 className="text-xl font-bold tracking-tight text-foreground">
                    {contract.filename ?? "Sem nome"}
                  </h1>
                </div>
              </div>

              <Separator className="mb-8" />

              <ResultPreview isLoading={false} result={contract.result ?? null} />
            </>
          )}
        </main>
      </div>
    </div>
  )
}
