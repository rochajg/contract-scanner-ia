"use client"

import { useEffect, useState, useCallback } from "react"
import { useRouter } from "next/navigation"
import Link from "next/link"
import { useAuth } from "@/lib/auth-context"
import { listAnalyses, deleteAnalysis, type AnalysisSummary, type RiskLevel } from "@/lib/api"
import { Navbar } from "@/components/navbar"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Loader2,
  FileText,
  AlertTriangle,
  CheckCircle2,
  Clock,
  Plus,
  RefreshCw,
  Trash2,
} from "lucide-react"

// ── Status ─────────────────────────────────────────────────────────────────────

const statusLabel: Record<AnalysisSummary["status"], string> = {
  UPLOADED: "Aguardando",
  PROCESSING: "Processando",
  COMPLETED: "Concluído",
  FAILED: "Falhou",
}

const statusVariant: Record<
  AnalysisSummary["status"],
  "default" | "secondary" | "destructive" | "outline"
> = {
  UPLOADED: "outline",
  PROCESSING: "secondary",
  COMPLETED: "default",
  FAILED: "destructive",
}

// ── Risk ───────────────────────────────────────────────────────────────────────

const riskLabel: Record<RiskLevel, string> = {
  low: "Baixo risco",
  medium: "Risco médio",
  high: "Alto risco",
}

const riskVariant: Record<
  RiskLevel,
  "default" | "secondary" | "destructive"
> = {
  low: "secondary",
  medium: "default",
  high: "destructive",
}

// ── Helpers ────────────────────────────────────────────────────────────────────

function formatDate(iso: string) {
  return new Date(iso).toLocaleString("pt-BR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })
}

function StatusIcon({ status }: { status: AnalysisSummary["status"] }) {
  if (status === "PROCESSING")
    return <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
  if (status === "COMPLETED")
    return <CheckCircle2 className="h-4 w-4 text-green-500" />
  if (status === "FAILED")
    return <AlertTriangle className="h-4 w-4 text-destructive" />
  return <Clock className="h-4 w-4 text-muted-foreground" />
}

// ── Page ───────────────────────────────────────────────────────────────────────

export default function ContractsPage() {
  const { user, isLoading } = useAuth()
  const router = useRouter()

  const [analyses, setAnalyses] = useState<AnalysisSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [toast, setToast] = useState<{ message: string; type: "success" | "error" } | null>(null)

  useEffect(() => {
    if (!isLoading && !user) {
      router.push("/login")
    }
  }, [user, isLoading, router])

  const load = useCallback(async () => {
    setLoading(true)
    try {
      setAnalyses(await listAnalyses())
    } catch {
      // silently ignore
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (user) load()
  }, [user, load])

  const handleDelete = useCallback(
    async (id: string, filename: string | null) => {
      const label = filename ?? "este contrato"
      if (!window.confirm(`Excluir "${label}"? Esta ação não pode ser desfeita.`)) return

      setDeletingId(id)
      try {
        await deleteAnalysis(id)
        setAnalyses((prev) => prev.filter((a) => a.id !== id))
        setToast({ message: "Contrato excluído com sucesso.", type: "success" })
      } catch (err) {
        const message = err instanceof Error ? err.message : "Erro ao excluir"
        setToast({ message, type: "error" })
      } finally {
        setDeletingId(null)
      }
    },
    []
  )

  useEffect(() => {
    if (!toast) return
    const timer = setTimeout(() => setToast(null), 4000)
    return () => clearTimeout(timer)
  }, [toast])

  if (isLoading || !user) return null

  return (
    <div className="min-h-screen bg-background">
      <div className="mx-auto max-w-4xl px-4">
        <Navbar />

        <main className="pb-16 pt-8">
          {/* Toast notification */}
          {toast && (
            <div
              className={`mb-4 flex items-center justify-between rounded-md border px-4 py-3 text-sm ${
                toast.type === "success"
                  ? "border-green-200 bg-green-50 text-green-800 dark:border-green-800 dark:bg-green-950 dark:text-green-200"
                  : "border-destructive/30 bg-destructive/10 text-destructive"
              }`}
            >
              <span>{toast.message}</span>
              <button
                onClick={() => setToast(null)}
                className="ml-4 opacity-60 hover:opacity-100"
                aria-label="Fechar notificação"
              >
                ×
              </button>
            </div>
          )}

          {/* Page header */}
          <div className="mb-6 flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-foreground">
                Meus Contratos
              </h1>
              <p className="mt-1 text-sm text-muted-foreground">
                Histórico de todos os contratos analisados
              </p>
            </div>
            <div className="flex items-center gap-2">
              <Button
                variant="ghost"
                size="sm"
                onClick={load}
                disabled={loading}
                className="text-muted-foreground"
                title="Atualizar lista"
              >
                <RefreshCw
                  className={`h-4 w-4 ${loading ? "animate-spin" : ""}`}
                />
              </Button>
              <Link href="/">
                <Button size="sm">
                  <Plus className="mr-1.5 h-4 w-4" />
                  Analisar novo contrato
                </Button>
              </Link>
            </div>
          </div>

          <Card className="border-border bg-card shadow-sm">
            <CardContent className="p-0">
              {/* Loading state */}
              {loading && analyses.length === 0 && (
                <div className="flex items-center justify-center py-16 text-muted-foreground">
                  <Loader2 className="mr-2 h-5 w-5 animate-spin" />
                  <span className="text-sm">Carregando contratos...</span>
                </div>
              )}

              {/* Empty state */}
              {!loading && analyses.length === 0 && (
                <div className="flex flex-col items-center justify-center gap-3 py-16">
                  <FileText className="h-10 w-10 text-muted-foreground/40" />
                  <p className="text-sm text-muted-foreground">
                    Nenhum contrato analisado ainda.
                  </p>
                  <Link href="/">
                    <Button size="sm" variant="outline">
                      <Plus className="mr-1.5 h-4 w-4" />
                      Analisar primeiro contrato
                    </Button>
                  </Link>
                </div>
              )}

              {/* Table */}
              {analyses.length > 0 && (
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-border bg-muted/40">
                        <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                          Arquivo
                        </th>
                        <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                          Status
                        </th>
                        <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                          Risco
                        </th>
                        <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                          Enviado em
                        </th>
                        <th className="px-4 py-3 text-right font-medium text-muted-foreground">
                          Ações
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border">
                      {analyses.map((item) => (
                        <tr
                          key={item.id}
                          className="group transition-colors hover:bg-muted/20"
                        >
                          {/* Filename */}
                          <td className="px-4 py-3">
                            <div className="flex items-center gap-2.5">
                              <StatusIcon status={item.status} />
                              <span className="max-w-xs truncate font-medium text-foreground">
                                {item.filename ?? "Sem nome"}
                              </span>
                            </div>
                          </td>

                          {/* Status badge */}
                          <td className="px-4 py-3">
                            <Badge
                              variant={statusVariant[item.status]}
                              className="text-xs"
                            >
                              {statusLabel[item.status]}
                            </Badge>
                          </td>

                          {/* Risk badge */}
                          <td className="px-4 py-3">
                            {item.status === "COMPLETED" &&
                            item.result?.overall_risk ? (
                              <Badge
                                variant={riskVariant[item.result.overall_risk]}
                                className="text-xs"
                              >
                                {riskLabel[item.result.overall_risk]}
                              </Badge>
                            ) : (
                              <span className="text-xs text-muted-foreground">
                                —
                              </span>
                            )}
                          </td>

                          {/* Date */}
                          <td className="px-4 py-3 text-muted-foreground">
                            {formatDate(item.created_at)}
                          </td>

                          {/* Actions */}
                          <td className="px-4 py-3 text-right">
                            <div className="flex items-center justify-end gap-1">
                              {item.status === "COMPLETED" && item.result ? (
                                <Link href={`/contracts/${item.id}`}>
                                  <Button
                                    size="sm"
                                    variant="outline"
                                    className="h-7 px-2.5 text-xs"
                                  >
                                    Ver resultado
                                  </Button>
                                </Link>
                              ) : item.status === "FAILED" ? (
                                <span className="text-xs text-destructive">
                                  Falhou
                                </span>
                              ) : null}
                              <Button
                                size="sm"
                                variant="ghost"
                                className="h-7 w-7 p-0 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                                onClick={() => handleDelete(item.id, item.filename)}
                                disabled={deletingId === item.id}
                                title="Excluir contrato"
                                aria-label="Excluir contrato"
                              >
                                {deletingId === item.id ? (
                                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                                ) : (
                                  <Trash2 className="h-3.5 w-3.5" />
                                )}
                              </Button>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </CardContent>
          </Card>
        </main>
      </div>
    </div>
  )
}
