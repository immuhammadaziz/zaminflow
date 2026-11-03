import { NextResponse } from "next/server";
import { getPublicRuntimeConfig } from "@/lib/config/runtime";

export const dynamic = "force-dynamic";

export function GET() {
  const config = getPublicRuntimeConfig();
  return NextResponse.json(config, {
    headers: {
      "Cache-Control": "no-store, max-age=0",
    },
  });
}
