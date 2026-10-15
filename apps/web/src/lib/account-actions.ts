"use server";

// Password change is still mock: it only redirects back with a notice. Form data stays in the POST
// body; nothing is stored or logged.
import { redirect } from "next/navigation";
import { getSession, renewSession } from "@/lib/session";
import { updateProfile as saveProfile } from "@/lib/user-api";

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

export async function changePassword() {
  redirect("/account?saved=password");
}
