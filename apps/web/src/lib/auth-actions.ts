"use server";

// Sign-in, Google and password reset are still mock: each only redirects so the flow can be walked
// through. Credentials arrive in the POST body and are never stored, logged or put in a URL.
import { redirect } from "next/navigation";
import { registerAccount } from "@/lib/user-api";

/** What the register form shows after a refused attempt, with the values to type in again. */
export type RegisterState = {
  error?: string;
  name?: string;
  email?: string;
};

const text = (formData: FormData, name: string) => String(formData.get(name) ?? "").trim();

export async function signIn() {
  redirect("/");
}

export async function register(_prev: RegisterState, formData: FormData): Promise<RegisterState> {
  const name = text(formData, "name");
  const email = text(formData, "email");
  // Passwords are sent as typed: spaces are part of a password.
  const password = String(formData.get("password") ?? "");
  const keep = { name, email };

  // The browser checks this too; a form posted before the page hydrated has not.
  if (password !== String(formData.get("confirm") ?? "")) return { ...keep, error: "The passwords do not match." };

  const result = await registerAccount({ name, email, password });
  if (result.ok) redirect("/sign-in?registered=1");

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

export async function continueWithGoogle() {
  redirect("/");
}

/** Same answer whether or not the address has an account, so the form cannot probe for users. */
export async function requestReset() {
  redirect("/forgot-password?sent=1");
}
