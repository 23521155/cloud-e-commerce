import { app, InvocationContext, output } from "@azure/functions";
import { BlobServiceClient } from "@azure/storage-blob";
import { imageSize } from "image-size";

type BlobCreatedEvent = {
    eventType: string;
    subject: string;
    eventTime: string;
    data?: { contentType?: string; contentLength?: number };
};

type PhotoRow = {
    blob_name: string;
    request_id: number;
    position: number;
    content_type: string;
    size_bytes: number;
    width: number | null;
    height: number | null;
    processed_at: string;
};

const CONTAINER = process.env.SELL_PHOTOS_CONTAINER ?? "sell-photos";
const SUBJECT = new RegExp(`^/blobServices/default/containers/${CONTAINER}/blobs/((\\d+)/(\\d+)\\.(?:jpg|png|webp|gif))$`);
const HEADER_BYTES = 64 * 1024;

const photoRows = output.sql({
    commandText: "dbo.sell_request_photos",
    connectionStringSetting: "CatalogSqlConnectionString",
});

let blobService: BlobServiceClient | undefined;
function container() {
    blobService ??= BlobServiceClient.fromConnectionString(process.env.AzureWebJobsStorage ?? "");
    return blobService.getContainerClient(CONTAINER);
}

function events(message: unknown): BlobCreatedEvent[] {
    const parsed = typeof message === "string" ? JSON.parse(message) : message;
    return (Array.isArray(parsed) ? parsed : [parsed]).filter(
        (e): e is BlobCreatedEvent => typeof e?.eventType === "string" && typeof e?.subject === "string",
    );
}

async function dimensions(blobName: string, length: number): Promise<{ width: number | null; height: number | null }> {
    if (length <= 0) return { width: null, height: null };
    const head = await container().getBlobClient(blobName).downloadToBuffer(0, Math.min(HEADER_BYTES, length));
    try {
        const { width, height } = imageSize(head);
        return { width: width ?? null, height: height ?? null };
    } catch {
        return { width: null, height: null };
    }
}

export async function ingestSellPhoto(message: unknown, context: InvocationContext): Promise<void> {
    const rows: PhotoRow[] = [];

    for (const event of events(message)) {
        if (event.eventType !== "Microsoft.Storage.BlobCreated") continue;

        const match = SUBJECT.exec(event.subject);
        if (!match) {
            context.warn(`Skipping ${event.subject}: not a sell photo`);
            continue;
        }
        const [, blobName, requestId, position] = match;

        const props = await container().getBlobClient(blobName).getProperties();
        const size = await dimensions(blobName, props.contentLength ?? 0);
        const processedAt = new Date();

        rows.push({
            blob_name: blobName,
            request_id: Number(requestId),
            position: Number(position),
            content_type: props.contentType ?? event.data?.contentType ?? "application/octet-stream",
            size_bytes: props.contentLength ?? event.data?.contentLength ?? 0,
            width: size.width,
            height: size.height,
            processed_at: processedAt.toISOString(),
        });

        const lagMs = processedAt.getTime() - new Date(event.eventTime).getTime();
        context.log(`Ingested ${blobName} (${size.width}x${size.height}) ${lagMs} ms after upload`);
    }

    if (rows.length > 0) context.extraOutputs.set(photoRows, rows);
}

app.storageQueue("ingestSellPhoto", {
    queueName: "sell-photo-events",
    connection: "AzureWebJobsStorage",
    extraOutputs: [photoRows],
    handler: ingestSellPhoto,
});
