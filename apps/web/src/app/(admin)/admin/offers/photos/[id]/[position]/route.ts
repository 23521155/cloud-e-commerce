import { getSession, isStaffWithCode } from "@/lib/session";
import { sellPhoto } from "@/lib/sell-api";

export async function GET(_req: Request, { params }: RouteContext<"/admin/offers/photos/[id]/[position]">) {
  const session = await getSession();
  if (!session || !session.roles.includes("staff") || !isStaffWithCode(session)) return new Response(null, { status: 403 });

  const { id, position } = await params;
  if (!/^\d+$/.test(id) || !/^\d+$/.test(position)) return new Response(null, { status: 404 });

  const res = await sellPhoto(Number(id), Number(position));
  if (!res.ok || !res.body) return new Response(null, { status: res.status === 404 ? 404 : 502 });

  return new Response(res.body, {
    headers: {
      "Content-Type": res.headers.get("Content-Type") ?? "application/octet-stream",
      "Cache-Control": "private, max-age=3600",
    },
  });
}
