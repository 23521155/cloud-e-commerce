import { redirect } from "next/navigation";
import type { NextRequest } from "next/server";
import { localPath, startGoogleFlow } from "@/lib/google-flow";
import { googleStart } from "@/lib/user-api";

/** GET /auth/google?next=/orders: sends the reader to Google to choose an account. */
export async function GET(request: NextRequest) {
  const result = await googleStart();
  if (!result.ok) redirect(`/sign-in?google=${result.reason}`);

  const { url, state, nonce, verifier } = result.start;
  await startGoogleFlow({ state, nonce, verifier, next: localPath(request.nextUrl.searchParams.get("next")) });
  redirect(url);
}
