"use client"

import { useEffect, useState, useCallback, useMemo } from "react"
import { listAnalyses, type AnalysisSummary, type AnalysisResult, type RiskLevel } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Loader2, FileText, RefreshCw, Search, AlertTriangle, CheckCircle2, Clock } from "lucide-react"

// ── Status ────────────────────────────────────────────────────────────────────

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

// ── Risk ──────────────────────────────────────────────────────────────────────

const riskLabel: Record<RiskLevel, string> = {
  low: "Baixo risco",
  medium: "Risco médio",
  high: "Alto risco",
}

const riskVariant: Record<RiskLevel, "default" | "secondary" | "destructive"> = {
  low: "secondary",
  medium: "default",
  high: "destructive",
}

function RiskBadge({ risk }: { risk: RiskLevel }) {
  return (
    <Badge variant={riskVariant[risk]} className="text-xs">
      {riskLabel[risk]}
    </Badge>
  )
}

// ── Helpers ───────────────────────────────────────────────────────────────────

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
  if (status === "PROCESSING") return <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
  if (status === "COMPLETED") return <CheckCircle2 className="h-4 w-4 text-green-500" />
  if (status === "FAILED") return <AlertTriangle className="h-4 w-4 text-destructive" />
  return <Clock className="h-4 w-4 text-muted-foreground" />
}

// ── Component ─────────────────────────────────────────────────────────────────

interface AnalysesListProps {
  onSelect: (result: AnalysisResult) => void
}

export function AnalysesList({ onSelect }: AnalysesListProps) {
  const [analyses, setAnalyses] = useState<AnalysisSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [query, setQuery] = useState("")

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

  useEffect(() => { load() }, [load])

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return analyses
    return analyses.filter((a) =>
      (a.filename ?? "").toLowerCase().includes(q) ||
      a.id.toLowerCase().includes(q)
    )
  }, [analyses, query])

  // ── Render ────────────────────────────────────────────────────────────────

  return (
    <div className="space-y-3">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-foreground">
          Contratos enviados
          {analyses.length > 0 && (
            <span className="ml-2 text-xs font-normal text-muted-foreground">
              ({analyses.length})
            </span>
          )}
        </h2>
        <Button
          variant="ghost"
          size="sm"
          onClick={load}
          disabled={loading}
          className="h-7 px-2 text-muted-foreground"
          title="Atualizar lista"
        >
          <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
        </Button>
      </div>

      {/* Search */}
      {analyses.length > 0 && (
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Buscar por nome ou ID..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="h-8 pl-8 text-sm"
          />
        </div>
      )}

      {/* Loading state */}
      {loading && analyses.length === 0 && (
        <div className="flex items-center justify-center py-8 text-muted-foreground">
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          <span className="text-sm">Carregando...</span>
        </div>
      )}

      {/* Empty state */}
      {!loading && analyses.length === 0 && (
        <p className="py-6 text-center text-sm text-muted-foreground">
          Nenhum contrato enviado ainda.
        </p>
      )}

      {/* No results */}
      {!loading && analyses.length > 0 && filtered.length === 0 && (
        <p className="py-4 text-center text-sm text-muted-foreground">
          Nenhum contrato encontrado para &quot;{query}&quot;.
        </p>
      )}

      {/* List */}
      {filtered.length > 0 && (
        <ul className="divide-y divide-border rounded-lg border border-border">
          {filtered.map((item) => (
            <li key={item.id} className="flex items-center justify-between gap-3 px-4 py-3">
              {/* Left: icon + info */}
              <div className="flex min-w-0 items-start gap-3">
                <div className="mt-0.5 shrink-0">
                  <StatusIcon status={item.status} />
                </div>
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium text-foreground">
                    {item.filename ?? "Sem nome"}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {formatDate(item.created_at)}
                    {item.completed_at && item.status === "COMPLETED" && (
                      <span className="ml-2 text-muted-foreground/60">
                        · concluído {formatDate(item.completed_at)}
                      </span>
                    )}
                  </p>
                  {/* ID truncado para debug */}
                  <p className="mt-0.5 font-mono text-[10px] text-muted-foreground/50">
                    {item.id}
                  </p>
                </div>
              </div>

              {/* Right: badges + action */}
              <div className="flex shrink-0 flex-col items-end gap-1.5">
                <div className="flex items-center gap-1.5">
                  <Badge variant={statusVariant[item.status]} className="text-xs">
                    {statusLabel[item.status]}
                  </Badge>
                  {item.status === "COMPLETED" && item.result?.overall_risk && (
                    <RiskBadge risk={item.result.overall_risk} />
                  )}
                </div>

                {item.status === "COMPLETED" && item.result && (
                  <Button
                    size="sm"
                    variant="outline"
                    className="h-6 px-2 text-xs"
                    onClick={() => onSelect(item.result!)}
                  >
                    Ver resultado
                  </Button>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
