import type { APIRoute } from "astro";
import { env } from "cloudflare:workers";

export const prerender = false;

interface ContactEnv {
  TURNSTILE_SECRET_KEY?: string;
  AWS_SES_REGION?: string;
  AWS_ACCESS_KEY_ID?: string;
  AWS_SECRET_ACCESS_KEY?: string;
  CONTACT_FROM_ADDRESS?: string;
  CONTACT_TO_ADDRESS?: string;
}

interface ContactPayload {
  name?: string;
  email?: string;
  organization?: string;
  position?: string;
  message?: string;
  turnstileToken?: string;
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

function jsonResponse(status: number, body: Record<string, unknown>): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  });
}

async function verifyTurnstile(token: string, secret: string, ip?: string): Promise<boolean> {
  const body = new FormData();
  body.append("secret", secret);
  body.append("response", token);
  if (ip) body.append("remoteip", ip);
  const res = await fetch("https://challenges.cloudflare.com/turnstile/v0/siteverify", {
    method: "POST",
    body,
  });
  if (!res.ok) return false;
  const data = (await res.json()) as { success?: boolean };
  return data.success === true;
}

// --- AWS SigV4 helpers (Web Crypto, no SDK) ---

async function sha256Hex(data: string | ArrayBuffer): Promise<string> {
  const buf = typeof data === "string" ? new TextEncoder().encode(data) : data;
  const hash = await crypto.subtle.digest("SHA-256", buf);
  return [...new Uint8Array(hash)].map((b) => b.toString(16).padStart(2, "0")).join("");
}

async function hmac(key: ArrayBuffer | Uint8Array, data: string): Promise<ArrayBuffer> {
  const cryptoKey = await crypto.subtle.importKey(
    "raw",
    key,
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  return crypto.subtle.sign("HMAC", cryptoKey, new TextEncoder().encode(data));
}

async function signingKey(secret: string, date: string, region: string, service: string): Promise<ArrayBuffer> {
  const kDate = await hmac(new TextEncoder().encode("AWS4" + secret), date);
  const kRegion = await hmac(kDate, region);
  const kService = await hmac(kRegion, service);
  return hmac(kService, "aws4_request");
}

interface SesSendArgs {
  region: string;
  accessKeyId: string;
  secretAccessKey: string;
  fromAddress: string;
  toAddress: string;
  replyTo?: string;
  subject: string;
  bodyText: string;
}

async function sendSesEmail(args: SesSendArgs): Promise<{ ok: boolean; status: number; body: string }> {
  const host = `email.${args.region}.amazonaws.com`;
  const path = "/v2/email/outbound-emails";
  const endpoint = `https://${host}${path}`;
  const payload = JSON.stringify({
    FromEmailAddress: args.fromAddress,
    Destination: { ToAddresses: [args.toAddress] },
    ReplyToAddresses: args.replyTo ? [args.replyTo] : undefined,
    Content: {
      Simple: {
        Subject: { Data: args.subject, Charset: "UTF-8" },
        Body: { Text: { Data: args.bodyText, Charset: "UTF-8" } },
      },
    },
  });

  const now = new Date();
  const amzDate = now.toISOString().replace(/[:-]|\.\d{3}/g, "");
  const dateStamp = amzDate.slice(0, 8);
  const service = "ses";

  const payloadHash = await sha256Hex(payload);
  const canonicalHeaders =
    `content-type:application/json\nhost:${host}\nx-amz-content-sha256:${payloadHash}\nx-amz-date:${amzDate}\n`;
  const signedHeaders = "content-type;host;x-amz-content-sha256;x-amz-date";
  const canonicalRequest = `POST\n${path}\n\n${canonicalHeaders}\n${signedHeaders}\n${payloadHash}`;
  const credentialScope = `${dateStamp}/${args.region}/${service}/aws4_request`;
  const stringToSign =
    `AWS4-HMAC-SHA256\n${amzDate}\n${credentialScope}\n${await sha256Hex(canonicalRequest)}`;
  const sigKey = await signingKey(args.secretAccessKey, dateStamp, args.region, service);
  const signature = [...new Uint8Array(await hmac(sigKey, stringToSign))]
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
  const authHeader =
    `AWS4-HMAC-SHA256 Credential=${args.accessKeyId}/${credentialScope}, ` +
    `SignedHeaders=${signedHeaders}, Signature=${signature}`;

  const res = await fetch(endpoint, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      host,
      "x-amz-date": amzDate,
      "x-amz-content-sha256": payloadHash,
      authorization: authHeader,
    },
    body: payload,
  });
  return { ok: res.ok, status: res.status, body: await res.text() };
}

export const POST: APIRoute = async ({ request, clientAddress }) => {
  const e = env as unknown as ContactEnv;
  let payload: ContactPayload;
  try {
    payload = (await request.json()) as ContactPayload;
  } catch {
    return jsonResponse(400, { error: "Invalid JSON body" });
  }

  const name = payload.name?.trim() ?? "";
  const email = payload.email?.trim() ?? "";
  const organization = payload.organization?.trim() ?? "";
  const position = payload.position?.trim() ?? "";
  const message = payload.message?.trim() ?? "";
  const token = payload.turnstileToken?.trim() ?? "";

  if (name.length < 2 || name.length > 200) return jsonResponse(400, { error: "Valid name is required" });
  if (!EMAIL_RE.test(email) || email.length > 320) return jsonResponse(400, { error: "Valid email required" });
  if (organization.length < 1 || organization.length > 200) return jsonResponse(400, { error: "Valid organization is required" });
  if (position.length < 1 || position.length > 200) return jsonResponse(400, { error: "Valid title is required" });
  if (message.length < 5 || message.length > 5000) return jsonResponse(400, { error: "Valid message is required" });
  if (!token) return jsonResponse(400, { error: "Captcha challenge missing" });

  const turnstileSecret = e.TURNSTILE_SECRET_KEY;
  const awsRegion = e.AWS_SES_REGION;
  const awsKey = e.AWS_ACCESS_KEY_ID;
  const awsSecret = e.AWS_SECRET_ACCESS_KEY;
  const fromAddr = e.CONTACT_FROM_ADDRESS;
  const toAddr = e.CONTACT_TO_ADDRESS ?? "info@stackific.com";

  if (!turnstileSecret || !awsRegion || !awsKey || !awsSecret || !fromAddr) {
    return jsonResponse(500, { error: "Server is not configured" });
  }

  const ok = await verifyTurnstile(token, turnstileSecret, clientAddress);
  if (!ok) return jsonResponse(400, { error: "Captcha verification failed" });

  const subject = `Work with us — ${name}${organization ? ` (${organization})` : ""}`;
  const bodyText =
    `New "Work with us" submission\n\n` +
    `Name: ${name}\n` +
    `Email: ${email}\n` +
    `Organization: ${organization || "—"}\n` +
    `Title: ${position || "—"}\n\n` +
    `Message:\n${message}\n`;

  const result = await sendSesEmail({
    region: awsRegion,
    accessKeyId: awsKey,
    secretAccessKey: awsSecret,
    fromAddress: fromAddr,
    toAddress: toAddr,
    replyTo: email,
    subject,
    bodyText,
  });

  if (!result.ok) {
    console.error("SES send failed", result.status, result.body);
    return jsonResponse(502, { error: "Failed to send message" });
  }

  return jsonResponse(200, { ok: true });
};
