import { useEffect, useRef } from 'react';

/**
 * useAdminEvents listens for real-time Server-Sent Events (SSE) from the Go backend.
 * 
 * @param {Record<string, (data: any) => void>} handlers - Object mapping event names to handler callbacks.
 */
export default function useAdminEvents(handlers) {
  const handlersRef = useRef(handlers);

  useEffect(() => {
    handlersRef.current = handlers;
  }, [handlers]);

  useEffect(() => {
    let es = null;

    try {
      es = new EventSource('/admin/api/events');

      // Register custom event listeners
      const eventNames = Object.keys(handlersRef.current || {});
      const cleanupListeners = eventNames.map(eventName => {
        const listener = (event) => {
          try {
            const parsed = JSON.parse(event.data);
            if (handlersRef.current && handlersRef.current[eventName]) {
              handlersRef.current[eventName](parsed);
            }
          } catch (err) {
            console.error(`[SSE] Failed to parse event "${eventName}":`, err);
          }
        };

        es.addEventListener(eventName, listener);
        return () => es.removeEventListener(eventName, listener);
      });

      es.onerror = (err) => {
        // EventSource will automatically attempt reconnection
        console.warn('[SSE] EventSource connection issue, auto-reconnecting...', err);
      };

      return () => {
        cleanupListeners.forEach(cleanup => cleanup());
        if (es) {
          es.close();
        }
      };
    } catch (err) {
      console.error('[SSE] Failed to initialize EventSource:', err);
    }
  }, []);
}
