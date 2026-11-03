let accessToken: string | null = null;
let sessionChannel: BroadcastChannel | undefined;
const authExpiredListeners = new Set<() => void>();
const crossTabListeners = new Set<
  (event: "session_available" | "signed_out") => void
>();

function getSessionChannel(): BroadcastChannel | undefined {
  if (typeof window === "undefined" || !("BroadcastChannel" in window)) {
    return undefined;
  }
  sessionChannel ??= new BroadcastChannel("zaminflow-session");
  return sessionChannel;
}

export function getAccessToken(): string | null {
  return accessToken;
}

export function setAccessToken(token: string): void {
  accessToken = token;
}

export function clearAccessToken(): void {
  accessToken = null;
}

export function broadcastSessionAvailable(): void {
  getSessionChannel()?.postMessage({ type: "session_available" });
}

export function broadcastSignedOut(): void {
  accessToken = null;
  getSessionChannel()?.postMessage({ type: "signed_out" });
  for (const listener of authExpiredListeners) listener();
}

export function subscribeAuthExpired(listener: () => void): () => void {
  authExpiredListeners.add(listener);
  return () => authExpiredListeners.delete(listener);
}

export function subscribeCrossTabSession(
  listener: (event: "session_available" | "signed_out") => void,
): () => void {
  crossTabListeners.add(listener);
  const channel = getSessionChannel();
  if (channel) {
    channel.onmessage = (message: MessageEvent<unknown>) => {
      if (
        typeof message.data === "object" &&
        message.data !== null &&
        "type" in message.data &&
        (message.data.type === "session_available" ||
          message.data.type === "signed_out")
      ) {
        const type = message.data.type;
        if (type === "signed_out") accessToken = null;
        for (const subscriber of crossTabListeners) subscriber(type);
        if (type === "signed_out") {
          for (const subscriber of authExpiredListeners) subscriber();
        }
      }
    };
  }
  return () => {
    crossTabListeners.delete(listener);
    if (crossTabListeners.size === 0 && sessionChannel) {
      sessionChannel.onmessage = null;
      sessionChannel.close();
      sessionChannel = undefined;
    }
  };
}

export async function withCrossTabRefreshLock<T>(
  operation: () => Promise<T>,
): Promise<T> {
  if (typeof navigator !== "undefined" && navigator.locks) {
    return navigator.locks.request(
      "zaminflow-refresh-token",
      { mode: "exclusive" },
      operation,
    );
  }
  return operation();
}
