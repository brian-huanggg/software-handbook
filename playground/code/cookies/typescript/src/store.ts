import { randomBytes } from "node:crypto";

export const SESSION_TTL = 3600

interface Session {
    status: string;
    createdAt: number;
    expiresIn: number;
}

const sessions = new Map<string, Session>();

export function create_session(): string {
    const sessionId = randomBytes(32).toString("base64url"); // randomBytes(32) 回傳 Buffer，是 32 個原始位元組
    sessions.set(sessionId, {
        status: "unauthenticated",
        createdAt: Date.now(),
        expiresIn: SESSION_TTL,
    });

    return sessionId;
}