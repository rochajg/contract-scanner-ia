import { NextRequest, NextResponse } from "next/server"

const BACKEND = process.env.API_URL || "http://localhost:8080"

/**
 * Route Handler for POST /api/analyses/:id/process
 *
 * This file exists solely to bypass the Next.js rewrite proxy, which has a
 * short connection timeout that causes ECONNRESET on long-running requests.
 * Route Handlers take precedence over rewrites and use Node's fetch directly,
 * which has no built-in timeout — suitable for PDF extraction + LLM calls
 * that can take several minutes.
 */
export async function POST(
  req: NextRequest,
  context: { params: Promise<{ id: string }> }
) {
  const { id } = await context.params
  const backendUrl = `${BACKEND}/api/analyses/${id}/process`
  const auth = req.headers.get("Authorization")

  try {
    const res = await fetch(backendUrl, {
      method: "POST",
      headers: auth ? { Authorization: auth } : {},
    })
    const data = await res.json()
    return NextResponse.json(data, { status: res.status })
  } catch (err) {
    const message = err instanceof Error ? err.message : "backend unreachable"
    return NextResponse.json({ error: message }, { status: 502 })
  }
}
