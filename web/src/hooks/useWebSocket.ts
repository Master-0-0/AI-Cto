import { useEffect, useRef, useState, useCallback } from 'react';

type ConnectionStatus = 'connecting' | 'connected' | 'disconnected';

interface UseWebSocketOptions {
  onMessage?: (event: MessageEvent) => void;
}

export function useWebSocket(agentId: string | null, options?: UseWebSocketOptions) {
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>('disconnected');
  const [lastMessage, setLastMessage] = useState<MessageEvent | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<number>();
  const reconnectAttemptsRef = useRef(0);
  const optionsRef = useRef(options);
  optionsRef.current = options; // always keep latest without triggering re-render

  const connect = useCallback(() => {
    if (!agentId) return;

    const token = localStorage.getItem('access_token');
    if (!token) return;

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws/agents/${agentId}?token=${token}`;

    setConnectionStatus('connecting');
    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      setConnectionStatus('connected');
      reconnectAttemptsRef.current = 0;
    };

    ws.onmessage = (event) => {
      setLastMessage(event);
      optionsRef.current?.onMessage?.(event);
    };

    ws.onclose = () => {
      setConnectionStatus('disconnected');
      wsRef.current = null;

      // Auto-reconnect with exponential backoff
      const delay = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000);
      reconnectAttemptsRef.current++;
      reconnectTimeoutRef.current = window.setTimeout(connect, delay);
    };

    ws.onerror = () => {
      ws.close();
    };

    wsRef.current = ws;
  }, [agentId]); // options removed from deps — uses ref instead

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  }, []);

  useEffect(() => {
    connect();
    return disconnect;
  }, [agentId, connect, disconnect]);

  const sendMessage = useCallback((data: unknown) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data));
    }
  }, []);

  return { sendMessage, lastMessage, connectionStatus };
}
