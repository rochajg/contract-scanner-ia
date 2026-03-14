export interface AuthResponse {
  token: string
  user_id: string
  username: string
}

export async function loginUser(email: string, password: string): Promise<AuthResponse> {
  const res = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || "Falha ao fazer login")
  }
  return res.json()
}

export async function registerUser(
  username: string,
  email: string,
  password: string
): Promise<AuthResponse> {
  const res = await fetch("/api/auth/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, email, password }),
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || "Falha ao criar conta")
  }
  return res.json()
}

function getAuthHeaders(): HeadersInit {
  if (typeof window === "undefined") return {}
  const token = localStorage.getItem("auth_token")
  return token ? { Authorization: `Bearer ${token}` } : {}
}

export type RiskLevel = "low" | "medium" | "high"

export interface AnalysisSummary {
  id: string
  filename: string | null
  status: "UPLOADED" | "PROCESSING" | "COMPLETED" | "FAILED"
  created_at: string
  completed_at: string | null
  result?: AnalysisResult
}

export async function deleteAnalysis(id: string): Promise<void> {
  const res = await fetch(`/api/analyses/${id}`, {
    method: "DELETE",
    headers: { ...getAuthHeaders() },
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || "Falha ao excluir análise")
  }
}

export async function listAnalyses(): Promise<AnalysisSummary[]> {
  const res = await fetch("/api/analyses", {
    headers: { ...getAuthHeaders() },
  })
  if (!res.ok) throw new Error("Falha ao listar análises")
  const body = await res.json()
  return body.analyses ?? []
}

export interface Party {
  name: string
  role: string
  identifier: string | null
  address: string | null
}

export interface KeyClause {
  clause_name: string
  clause_summary: string
  clause_text_snippet: string
  clause_location: string
  risk_level: RiskLevel
  risk_explanation: string
  recommended_fix: string | null
}

export interface Risk {
  risk: string
  impact: RiskLevel
  clause_ref: string
  mitigation: string
}

export interface Recommendation {
  priority: RiskLevel
  action: string
  proposed_text: string | null
}

export interface AmbiguousTerm {
  term: string
  why_problematic: string
  suggested_clarification: string
}

export interface Obligation {
  obligation: string
  deadline_or_frequency: string
  clause_ref: string
}

export interface AnalysisResult {
  metadata: {
    language: string
    filename: string | null
    analysis_date: string
    confidence: number
  }
  summary: string
  parties: Party[]
  contract_type: string | null
  term_and_termination: {
    effective_date: string | null
    expiry_or_term: string | null
    renewal: string | null
    termination_rights: Array<{
      party: string
      notice_period: string
      cause: string
      clause_ref: string
    }>
  }
  financials: {
    payment_terms: string | null
    amounts: string[]
    penalties_and_interest: string | null
    security_or_guarantee: string | null
  }
  key_clauses: KeyClause[]
  obligations_of_signatory: Obligation[]
  ambiguous_terms: AmbiguousTerm[]
  indemnities_and_liabilities: {
    indemnity_summary: string | null
    liability_limit: string | null
    exclusions: string | null
  }
  confidentiality_and_ip: {
    confidentiality_summary: string | null
    ip_assignment_or_license: string | null
    risks: string[]
  }
  dispute_resolution: {
    governing_law: string | null
    forum_or_arbitration: string | null
    costs_allocation: string | null
  }
  risks: Risk[]
  recommendations: Recommendation[]
  quick_checks: string[]
  overall_risk: RiskLevel
  confidence?: number
  analysis_warnings?: string[]
}

interface UploadResponse {
  analysis_id: string
  s3_key: string
}

interface ProcessResponse {
  analysis_id: string
  status: string
  result: AnalysisResult
}

async function uploadPDF(file: File): Promise<UploadResponse> {
  const form = new FormData()
  form.append("file", file)

  const res = await fetch("/api/uploads", {
    method: "POST",
    headers: { ...getAuthHeaders() },
    body: form,
  })

  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || "Falha ao enviar arquivo")
  }

  return res.json()
}

async function processAnalysis(analysisId: string): Promise<ProcessResponse> {
  const res = await fetch(`/api/analyses/${analysisId}/process`, {
    method: "POST",
    headers: { ...getAuthHeaders() },
  })

  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || "Falha ao processar contrato")
  }

  return res.json()
}

export type AnalysisStep = "uploading" | "processing"

export async function analyzeContract(
  file: File,
  onStepChange?: (step: AnalysisStep) => void
): Promise<AnalysisResult> {
  onStepChange?.("uploading")
  const { analysis_id } = await uploadPDF(file)

  onStepChange?.("processing")
  const { result } = await processAnalysis(analysis_id)

  return result
}
