"use server";

// Mock auth: no users, no sessions yet. Each action only redirects so the flows can be walked
// through. Credentials arrive in the POST body and are dropped; nothing is stored or logged.
import { redirect } from "next/navigation";

export async function signIn() {
  redirect("/");
}

export async function register() {
  redirect("/");
}

export async function continueWithGoogle() {
  redirect("/");
}

/** Same answer whether or not the address has an account, so the form cannot probe for users. */
export async function requestReset() {
  redirect("/forgot-password?sent=1");
}
