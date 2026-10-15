import { app, InvocationContext } from "@azure/functions";
import nodemailer from "nodemailer";

/**
 * One email, already written by the service that queued it (user-service writes its emails in
 * internal/email). This function only delivers it.
 */
type Email = { to: string; subject: string; text: string; html?: string };

// SMTP_* are app settings: Azure Communication Services on Azure, Mailpit locally.
const transport = nodemailer.createTransport({
    host: process.env.SMTP_HOST,
    port: Number(process.env.SMTP_PORT ?? 587),
    // Port 465 speaks TLS from the start; 587 upgrades with STARTTLS.
    secure: process.env.SMTP_PORT === "465",
    requireTLS: process.env.SMTP_REQUIRE_TLS === "true",
    auth: process.env.SMTP_USER ? { user: process.env.SMTP_USER, pass: process.env.SMTP_PASSWORD } : undefined,
});

function isEmail(value: unknown): value is Email {
    const e = value as Email;
    return typeof e?.to === "string" && e.to !== "" && typeof e.subject === "string" && typeof e.text === "string";
}

/**
 * Sends each message of the email-outbox queue. A failed send throws, so the queue offers the
 * message again; after maxDequeueCount tries (host.json) it moves to email-outbox-poison.
 */
export async function sendEmail(message: unknown, context: InvocationContext): Promise<void> {
    if (!isEmail(message)) {
        // Retrying cannot fix it: log and drop, so it does not block the queue.
        context.error(`Dropping message ${context.triggerMetadata?.id}: not an email`);
        return;
    }

    const info = await transport.sendMail({
        from: process.env.SMTP_FROM,
        to: message.to,
        subject: message.subject,
        text: message.text,
        html: message.html,
    });

    // The address is personal data: log the message ids only.
    context.log(`Sent queue message ${context.triggerMetadata?.id} as ${info.messageId} (try ${context.triggerMetadata?.dequeueCount})`);
}

app.storageQueue("sendEmail", {
    queueName: "email-outbox",
    connection: "AzureWebJobsStorage",
    handler: sendEmail,
});
