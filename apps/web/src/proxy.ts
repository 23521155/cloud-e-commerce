import { NextResponse, type NextRequest } from "next/server";
import { visitorHeadersFrom } from "@/lib/gateway";
import { ACCESS_COOKIE, readAccessToken, REFRESH_COOKIE, sessionCookies } from "@/lib/session";
import { refreshSession } from "@/lib/user-api";

// Renew an access token that has run out (or is about to) before the page renders, so pages and
// Server Functions always see a usable session. One call to the user service, and only when the
// access token needs it.
const RENEW_BEFORE_SECONDS = 30;

export async function proxy(request: NextRequest) {
  const refreshToken = request.cookies.get(REFRESH_COOKIE)?.value;
  if (!refreshToken) return NextResponse.next();

  const session = readAccessToken(request.cookies.get(ACCESS_COOKIE)?.value);
  if (session && session.expiresAt - Date.now() / 1000 > RENEW_BEFORE_SECONDS) return NextResponse.next();

  const result = await refreshSession(refreshToken, visitorHeadersFrom(request.headers));

  if (!result.ok) {
    // The user service is unreachable: render signed out this time and try again next request.
    if (result.reason === "failed") return NextResponse.next();

    // Signed out elsewhere, expired, or the token was reused: forget the session.
    request.cookies.delete(ACCESS_COOKIE);
    request.cookies.delete(REFRESH_COOKIE);
    const response = NextResponse.next({ request: { headers: request.headers } });
    response.cookies.delete(ACCESS_COOKIE);
    response.cookies.delete(REFRESH_COOKIE);
    return response;
  }

  // The page rendering now reads the new token from the request; the browser keeps it from the response.
  const cookies = sessionCookies(result.tokens);
  for (const { name, value } of cookies) request.cookies.set(name, value);
  const response = NextResponse.next({ request: { headers: request.headers } });
  for (const cookie of cookies) response.cookies.set(cookie);
  return response;
}

export const config = {
  // Pages and Server Functions only: not build output, images or files from public/.
  matcher: ["/((?!_next/static|_next/image|images/|favicon.ico|icon.svg|logo.png|.*\\.(?:svg|png|jpg|jpeg|webp|ico)$).*)"],
};
