import { redirect } from "next/navigation";
import type { NextRequest } from "next/server";
import { takeGoogleFlow } from "@/lib/google-flow";
import { startCodeStep, startSession } from "@/lib/session";
import { googleSignIn } from "@/lib/user-api";

/**
 * GET /auth/google/callback: Google sends the reader back here with a code (or an error when they
 * cancelled). The state must be the one this browser left with; the user service does the rest.
 */
export async function GET(request: NextRequest) {
  const params = request.nextUrl.searchParams;
  const flow = await takeGoogleFlow();

  if (params.get("error")) redirect("/sign-in?google=cancelled");

  const code = params.get("code");
  if (!flow || !code || params.get("state") !== flow.state) redirect("/sign-in?google=failed");

  const result = await googleSignIn(code, flow.verifier, flow.nonce);
  if (result.ok) {
    await startSession(result.tokens);
    redirect(flow.next);
  }

  if (result.reason === "mfa") {
    // Two-step sign-in is on: the authenticator code comes next, as after a password.
    await startCodeStep(result.mfaToken, result.mfaExpiresAt);
    redirect(`/sign-in/code?${new URLSearchParams({ next: flow.next })}`);
  }

  redirect(`/sign-in?google=${result.reason === "refused" ? "failed" : result.reason}`);
}
