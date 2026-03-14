import { NextRequest, NextResponse } from "next/server"

const BACKEND = process.env.API_URL || "http://localhost:8080"

export async function DELETE(
  req: NextRequest,
  context: { params: Promise<{ id: string }> }
) {
  const { id } = await context.params
  const auth = req.headers.get("Authorization")

  try {
    const res = await fetch(`${BACKEND}/api/analyses/${id}`, {
      method: "DELETE",
      headers: auth ? { Authorization: auth } : {},
    })

    if (res.status === 204) {
      return new NextResponse(null, { status: 204 })
    }

    const data = await res.json().catch(() => ({}))
    return NextResponse.json(data, { status: res.status })
  } catch (err) {
    const message = err instanceof Error ? err.message : "backend unreachable"
    return NextResponse.json({ error: message }, { status: 502 })
  }
}
