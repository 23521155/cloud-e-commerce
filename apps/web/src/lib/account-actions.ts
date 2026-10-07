"use server";

// Mock account: no users or sessions yet. Each action only redirects back with a notice so the
// flow can be walked through. Form data stays in the POST body; nothing is stored or logged.
import { redirect } from "next/navigation";

export async function updateProfile() {
  redirect("/account?saved=profile");
}

export async function changePassword() {
  redirect("/account?saved=password");
}
