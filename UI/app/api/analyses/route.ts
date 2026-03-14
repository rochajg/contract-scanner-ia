import { NextRequest, NextResponse } from "next/server"

const BACKEND = process.env.API_URL || "http://localhost:8080"

export async function GET(req: NextRequest) {
  try {
    const auth = req.headers.get("Authorization")
    const res = await fetch(`${BACKEND}/api/analyses`, {
      headers: auth ? { Authorization: auth } : {},
    })
    const data = await res.json()
    return NextResponse.json(data, { status: res.status })
  } catch (err) {
    const message = err instanceof Error ? err.message : "backend unreachable"
    return NextResponse.json({ error: message }, { status: 502 })
  }
}
