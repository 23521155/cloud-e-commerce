"use server";

// Google sign-in and password reset are still mock: each only redirects so the flow can be walked
// through. Credentials arrive in the POST body and are never stored, logged or put in a URL.
import { redirect } from "next/navigation";
import { endSession, getRefreshToken, startSession } from "@/lib/session";
import { login, logout, registerAccount } from "@/lib/user-api";

/** What an auth form shows after a refused attempt, with the values to type in again. */
export type AuthFormState = {
  error?: string;
  name?: string;
  email?: string;
};

const text = (formData: FormData, name: string) => String(formData.get(name) ?? "").trim();

/** Only a path on this site, so the sign-in form cannot be used to send readers elsewhere. */
function localPath(value: FormDataEntryValue | null): string {
  const path = String(value ?? "");
  return path.startsWith("/") && !path.startsWith("//") && !path.startsWith("/\\") ? path : "/";
}

export async function signIn(_prev: AuthFormState, formData: FormData): Promise<AuthFormState> {
  const email = text(formData, "email");
  const result = await login(email, String(formData.get("password") ?? ""));

  if (result.ok) {
    await startSession(result.tokens);
    redirect(localPath(formData.get("next")));
  }

  switch (result.reason) {
    case "wrong":
      return { email, error: "The email or password is incorrect." };
    case "busy":
      return { email, error: "Too many attempts. Please wait a minute and try again." };
    default:
      return { email, error: "You could not be signed in right now. Please try again." };
  }
}

export async function register(_prev: AuthFormState, formData: FormData): Promise<AuthFormState> {
  const name = text(formData, "name");
  const email = text(formData, "email");
  // Passwords are sent as typed: spaces are part of a password.
  const password = String(formData.get("password") ?? "");
  const keep = { name, email };

  // The browser checks this too; a form posted before the page hydrated has not.
  if (password !== String(formData.get("confirm") ?? "")) return { ...keep, error: "The passwords do not match." };

  const result = await registerAccount({ name, email, password });
  if (result.ok) {
    // Signed in straight away; if that fails the account still exists, so send them to sign in.
    const signedIn = await login(email, password);
    if (!signedIn.ok) redirect("/sign-in?registered=1");
    await startSession(signedIn.tokens);
    redirect("/");
  }

  switch (result.reason) {
    case "taken":
      return { ...keep, error: "An account with this email already exists. Sign in instead, or reset the password." };
    case "invalid":
      // The service names the field, e.g. "email is not a valid address"
      return { ...keep, error: result.message ? `${result.message[0].toUpperCase()}${result.message.slice(1)}.` : "Please check the details." };
    case "busy":
      return { ...keep, error: "Too many attempts. Please wait a minute and try again." };
    default:
      return { ...keep, error: "The account could not be created right now. Please try again." };
  }
}

/** Ends the session on the user service too, so the refresh token cannot be used again. */
export async function signOut() {
  const refreshToken = await getRefreshToken();
  if (refreshToken) await logout(refreshToken);
  await endSession();
  redirect("/");
}

export async function continueWithGoogle() {
  redirect("/");
}

/** Same answer whether or not the address has an account, so the form cannot probe for users. */
export async function requestReset() {
  redirect("/forgot-password?sent=1");
}
