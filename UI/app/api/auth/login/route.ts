import { NextRequest, NextResponse } from "next/server"

const BACKEND = process.env.API_URL || "http://localhost:8080"

export async function POST(req: NextRequest) {
  try {
    const body = await req.json()
    const res = await fetch(`${BACKEND}/api/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    })
    const data = await res.json()
    return NextResponse.json(data, { status: res.status })
  } catch (err) {
    const message = err instanceof Error ? err.message : "backend unreachable"
    return NextResponse.json({ error: message }, { status: 502 })
  }
}
