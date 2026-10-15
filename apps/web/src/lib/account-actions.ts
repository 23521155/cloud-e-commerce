"use server";

// Form data stays in the POST body; passwords are never stored, logged or put in a URL. Outcomes
// come back to /account as a code in the query string.
import { redirect } from "next/navigation";
import { getSession, renewSession, startSession } from "@/lib/session";
import { changePassword as savePassword, updateProfile as saveProfile } from "@/lib/user-api";

const text = (formData: FormData, name: string) => String(formData.get(name) ?? "").trim();

/** Saves the ledger page: name, phone and delivery address. The email is shown but not sent. */
export async function updateProfile(formData: FormData) {
  const session = await getSession();
  if (!session) redirect("/sign-in?next=/account");

  const result = await saveProfile(session.token, {
    name: text(formData, "name"),
    phone: text(formData, "phone"),
    address: text(formData, "address"),
    city: text(formData, "city"),
    postalCode: text(formData, "postal"),
    country: text(formData, "country"),
  });

  if (result.ok) {
    // The header shows the name from the access token: get one with the new name.
    await renewSession();
    redirect("/account?saved=profile");
  }
  if (result.reason === "signed-out") redirect("/sign-in?next=/account");
  redirect(`/account?error=${result.reason === "invalid" ? "profile-invalid" : "profile-failed"}`);
}

/** Sets a new password. Every other device is signed out; this one gets a new session. */
export async function changePassword(formData: FormData) {
  const session = await getSession();
  if (!session) redirect("/sign-in?next=/account");

  const password = String(formData.get("password") ?? "");
  // The browser checks this too; a form posted before the page hydrated has not.
  if (password !== String(formData.get("confirm") ?? "")) redirect("/account?error=password-mismatch#password-slip");

  const result = await savePassword(session.token, String(formData.get("current") ?? ""), password);
  if (result.ok) {
    await startSession(result.tokens);
    redirect("/account?saved=password#password-slip");
  }

  switch (result.reason) {
    case "signed-out":
      redirect("/sign-in?next=/account");
    case "wrong-current":
      redirect("/account?error=password-wrong#password-slip");
    case "invalid":
      redirect("/account?error=password-invalid#password-slip");
    default:
      redirect("/account?error=password-failed#password-slip");
  }
}
