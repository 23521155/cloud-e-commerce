"use server";

// Form data stays in the POST body; passwords are never stored, logged or put in a URL. Outcomes
// come back to /account as a code in the query string.
import { redirect } from "next/navigation";
import QRCode from "qrcode";
import { getSession, renewSession, startSession } from "@/lib/session";
import {
  changePassword as savePassword,
  confirmTotp,
  disableTotp as turnOffTotp,
  resendVerification as sendLinkAgain,
  setupTotp,
  updateProfile as saveProfile,
  verifyEmail as confirmEmail,
} from "@/lib/user-api";

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

/**
 * Confirms the address from the link in the email. The link opens a page with a button rather than
 * confirming on its own, so a mail scanner that opens links cannot confirm for the reader.
 */
export async function verifyEmail(formData: FormData) {
  const result = await confirmEmail(String(formData.get("token") ?? ""));
  if (result.ok) redirect("/verify-email?done=1");
  redirect(`/verify-email?error=${result.reason}`);
}

/** Sends the confirmation link again from the account page. */
export async function resendVerification() {
  const session = await getSession();
  if (!session) redirect("/sign-in?next=/account");

  const result = await sendLinkAgain(session.token);
  if (result.ok) redirect("/account?saved=verify-sent");

  switch (result.reason) {
    case "signed-out":
      redirect("/sign-in?next=/account");
    case "already-verified":
      redirect("/account");
    case "too-soon":
      redirect("/account?error=verify-too-soon");
    default:
      redirect("/account?error=verify-failed");
  }
}

/**
 * The two-step sign-in slip while it is being set up: the QR code to scan, why a step failed, and
 * once it is on, the recovery codes to keep.
 */
export type TotpSetupState = { qr?: string; secret?: string; error?: string; recoveryCodes?: string[] };

/** Starts setup: a new secret, shown as a QR code for the authenticator app and as text. */
export async function startTotpSetup(): Promise<TotpSetupState> {
  const session = await getSession();
  if (!session) redirect("/sign-in?next=/account");

  const result = await setupTotp(session.token);
  if (result.ok) {
    // Drawn here so the secret never goes to a third-party QR service.
    const qr = await QRCode.toDataURL(result.otpauthUrl, { margin: 1, width: 220 });
    return { qr, secret: result.secret };
  }

  switch (result.reason) {
    case "signed-out":
      redirect("/sign-in?next=/account");
    case "already-on":
      redirect("/account");
    case "unavailable":
      return { error: "Two-step sign-in is not available right now." };
    default:
      return { error: "Setup could not start right now. Please try again in a moment." };
  }
}

/** Finishes setup with a first code from the app. */
export async function confirmTotpSetup(_prev: TotpSetupState, formData: FormData): Promise<TotpSetupState> {
  const session = await getSession();
  if (!session) redirect("/sign-in?next=/account");

  const shown = { qr: String(formData.get("qr") ?? ""), secret: String(formData.get("secret") ?? "") };

  const code = String(formData.get("code") ?? "").replace(/\s/g, "");
  const result = await confirmTotp(session.token, code);
  // The recovery codes are shown now and never again, so no redirect yet.
  if (result.ok) return { recoveryCodes: result.recoveryCodes };

  switch (result.reason) {
    case "signed-out":
      redirect("/sign-in?next=/account");
    case "wrong-code":
      // Keep the QR code on screen for another try; the form carries it back.
      return { ...shown, error: "That code is not right. Type the code the app shows now." };
    case "conflict":
      redirect("/account#two-step");
    default:
      return { ...shown, error: "Two-step sign-in could not be turned on right now. Please try again." };
  }
}

/** Turns two-step sign-in off; needs the password and a code from the app (or a recovery code). */
export async function disableTotp(formData: FormData) {
  const session = await getSession();
  if (!session) redirect("/sign-in?next=/account");

  const result = await turnOffTotp(session.token, String(formData.get("password") ?? ""), String(formData.get("code") ?? "").trim());
  if (result.ok) redirect("/account?saved=mfa-off#two-step");

  switch (result.reason) {
    case "signed-out":
      redirect("/sign-in?next=/account");
    case "wrong-password":
      redirect("/account?error=mfa-off-password#two-step");
    case "wrong-code":
      redirect("/account?error=mfa-off-code#two-step");
    default:
      redirect("/account?error=mfa-off-failed#two-step");
  }
}
