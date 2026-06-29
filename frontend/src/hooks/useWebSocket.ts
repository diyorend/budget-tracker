import { useEffect, useRef } from "react";
import toast from "react-hot-toast";

interface AlertMessage {
  type: string;
  category: string;
  spent: number;
  limit: number;
  percentage: number;
  message: string;
}

export function useAlertWebSocket() {
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (!token) return;

    const wsUrl = `${import.meta.env.VITE_WS_URL || "ws://localhost:8080"}/ws?token=${token}`;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      console.log("WebSocket connected");
    };

    ws.onmessage = (event) => {
      try {
        const msg: AlertMessage = JSON.parse(event.data);
        if (msg.type === "budget_alert") {
          const isExceeded = msg.percentage >= 100;
          toast(msg.message, {
            duration: 6000,
            icon: isExceeded ? "🚨" : "⚡",
            style: {
              background: isExceeded ? "#fee2e2" : "#fef3c7",
              color: "#1f2937",
              border: `1px solid ${isExceeded ? "#fca5a5" : "#fcd34d"}`,
            },
          });
        }
      } catch (e) {
        console.error("ws parse error", e);
      }
    };

    ws.onclose = () => {
      console.log("WebSocket disconnected");
    };

    ws.onerror = (err) => {
      console.error("WebSocket error", err);
    };

    return () => {
      ws.close();
    };
  }, []);

  return wsRef;
}
