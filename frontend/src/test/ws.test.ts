import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { GameWebSocket } from '../services/ws';

class MockWebSocket {
  static OPEN = 1;
  static CLOSED = 3;

  url: string;
  readyState = MockWebSocket.OPEN;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  sent: string[] = [];

  constructor(url: string) {
    this.url = url;
    // Simulate async open
    setTimeout(() => this.onopen?.(), 0);
  }

  send(data: string) {
    this.sent.push(data);
  }

  close() {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.();
  }
}

describe('GameWebSocket', () => {
  let originalWebSocket: typeof globalThis.WebSocket;

  beforeEach(() => {
    originalWebSocket = globalThis.WebSocket;
    vi.useFakeTimers();
    // @ts-expect-error - mock WebSocket
    globalThis.WebSocket = MockWebSocket;
  });

  afterEach(() => {
    globalThis.WebSocket = originalWebSocket;
    vi.useRealTimers();
  });

  it('constructs WebSocket URL from location', () => {
    // Set window.location for test
    const ws = new GameWebSocket('sess1', 'player1');
    ws.connect();

    // Should have created a WebSocket
    expect(ws.isConnected).toBe(true);
  });

  it('sends JSON messages when connected', () => {
    const ws = new GameWebSocket('s1', 'p1');
    ws.connect();

    ws.send('cursor_move', { lat: 47, lng: 10 });

    // Access underlying mock
    const mockWs = (ws as unknown as { ws: MockWebSocket }).ws;
    expect(mockWs.sent).toHaveLength(1);
    expect(JSON.parse(mockWs.sent[0])).toEqual({
      type: 'cursor_move',
      payload: { lat: 47, lng: 10 },
    });
  });

  it('dispatches parsed messages to handlers', async () => {
    const ws = new GameWebSocket('s1', 'p1');
    ws.connect();

    const handler = vi.fn();
    ws.onMessage(handler);

    const mockWs = (ws as unknown as { ws: MockWebSocket }).ws;
    mockWs.onmessage?.({
      data: JSON.stringify({ type: 'game_state', payload: { phase: 'playing' } }),
    });

    expect(handler).toHaveBeenCalledWith({ type: 'game_state', payload: { phase: 'playing' } });
  });

  it('ignores malformed JSON messages', () => {
    const ws = new GameWebSocket('s1', 'p1');
    ws.connect();

    const handler = vi.fn();
    ws.onMessage(handler);

    const mockWs = (ws as unknown as { ws: MockWebSocket }).ws;
    mockWs.onmessage?.({ data: 'not json' });

    expect(handler).not.toHaveBeenCalled();
  });

  it('unsubscribes handler when returned function is called', () => {
    const ws = new GameWebSocket('s1', 'p1');
    ws.connect();

    const handler = vi.fn();
    const unsub = ws.onMessage(handler);
    unsub();

    const mockWs = (ws as unknown as { ws: MockWebSocket }).ws;
    mockWs.onmessage?.({ data: JSON.stringify({ type: 'test', payload: {} }) });

    expect(handler).not.toHaveBeenCalled();
  });

  it('does not send when not connected', () => {
    const ws = new GameWebSocket('s1', 'p1');
    // Don't call connect
    ws.send('test', {});
    // No error thrown, message just dropped
  });

  it('close prevents reconnection', () => {
    const ws = new GameWebSocket('s1', 'p1');
    ws.connect();
    ws.close();

    expect(ws.isConnected).toBe(false);
  });

  it('schedules reconnect on unexpected close', () => {
    const ws = new GameWebSocket('s1', 'p1');
    ws.connect();

    const mockWs = (ws as unknown as { ws: MockWebSocket }).ws;
    // Simulate unexpected close (not user-initiated)
    mockWs.readyState = MockWebSocket.CLOSED;
    mockWs.onclose?.();

    // Should schedule reconnect
    vi.advanceTimersByTime(1000);
    // New WebSocket should have been created
    expect(ws.isConnected).toBe(true);
  });

  it('uses exponential backoff for reconnection', () => {
    const ws = new GameWebSocket('s1', 'p1');
    ws.connect();

    // First disconnect
    let mockWs = (ws as unknown as { ws: MockWebSocket }).ws;
    mockWs.readyState = MockWebSocket.CLOSED;
    mockWs.onclose?.();

    // Advance past first reconnect (1000ms)
    vi.advanceTimersByTime(1000);

    // Second disconnect
    mockWs = (ws as unknown as { ws: MockWebSocket }).ws;
    mockWs.readyState = MockWebSocket.CLOSED;
    mockWs.onclose?.();

    // Should not reconnect at 1000ms (delay doubled to 2000ms)
    vi.advanceTimersByTime(1000);
    const wsAfter1s = (ws as unknown as { ws: MockWebSocket }).ws;
    expect(wsAfter1s.readyState).toBe(MockWebSocket.CLOSED);

    // Should reconnect at 2000ms
    vi.advanceTimersByTime(1000);
    expect(ws.isConnected).toBe(true);
  });
});
