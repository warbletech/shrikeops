import { NextRequest, NextResponse } from "next/server";

export async function POST(req: NextRequest) {
  try {
    const { manifest, source } = await req.json();
    if (!manifest) {
      return NextResponse.json({ error: "manifest required" }, { status: 400 });
    }

    const scannerUrl = process.env.SCANNER_URL || "http://localhost:8090";

    const res = await fetch(`${scannerUrl}/scan`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ manifest, source: source || "upload" }),
    });

    if (!res.ok) {
      const text = await res.text();
      return NextResponse.json({ error: `Scanner error: ${text}` }, { status: 502 });
    }

    return NextResponse.json(await res.json());
  } catch (e: unknown) {
    return NextResponse.json({ error: e instanceof Error ? e.message : "Server error" }, { status: 500 });
  }
}
