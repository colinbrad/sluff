import { describe, it, expect, vi, beforeEach } from 'vitest';

// Mock fetch globally before importing api module
const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

// Import after mocking
const api = await import('../services/api');

function jsonResponse(data: unknown, status = 200) {
  return Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    statusText: 'OK',
    json: () => Promise.resolve(data),
  });
}

function errorResponse(error: string, status = 400) {
  return Promise.resolve({
    ok: false,
    status,
    statusText: 'Bad Request',
    json: () => Promise.resolve({ error }),
  });
}

describe('API client', () => {
  beforeEach(() => {
    mockFetch.mockReset();
  });

  describe('listMaps', () => {
    it('fetches maps from /api/admin/maps', async () => {
      const maps = [{ id: 'm1', name: 'Test Map' }];
      mockFetch.mockReturnValueOnce(jsonResponse(maps));

      const result = await api.listMaps();
      expect(result).toEqual(maps);
      expect(mockFetch).toHaveBeenCalledWith(
        '/api/admin/maps',
        expect.objectContaining({ headers: { 'Content-Type': 'application/json' } })
      );
    });
  });

  describe('createMap', () => {
    it('sends POST with name and description', async () => {
      const map = { id: 'm1', name: 'New Map', description: 'Desc' };
      mockFetch.mockReturnValueOnce(jsonResponse(map, 201));

      const result = await api.createMap('New Map', 'Desc');
      expect(result).toEqual(map);
      expect(mockFetch).toHaveBeenCalledWith(
        '/api/admin/maps',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ name: 'New Map', description: 'Desc' }),
        })
      );
    });
  });

  describe('getMap', () => {
    it('fetches a single map by id', async () => {
      const map = { id: 'm1', name: 'Map', rounds: [] };
      mockFetch.mockReturnValueOnce(jsonResponse(map));

      const result = await api.getMap('m1');
      expect(result).toEqual(map);
      expect(mockFetch).toHaveBeenCalledWith('/api/admin/maps/m1', expect.anything());
    });
  });

  describe('deleteMap', () => {
    it('sends DELETE and returns undefined for 204', async () => {
      mockFetch.mockReturnValueOnce(Promise.resolve({
        ok: true,
        status: 204,
        statusText: 'No Content',
        json: () => Promise.reject(new Error('no body')),
      }));

      const result = await api.deleteMap('m1');
      expect(result).toBeUndefined();
      expect(mockFetch).toHaveBeenCalledWith(
        '/api/admin/maps/m1',
        expect.objectContaining({ method: 'DELETE' })
      );
    });
  });

  describe('createRound', () => {
    it('sends round data with GeoJSON geometries', async () => {
      const roundData = {
        round_number: 1,
        name: 'Round 1',
        start_point: { type: 'Point' as const, coordinates: [10, 47] },
        end_point: { type: 'Point' as const, coordinates: [10.1, 47.1] },
        corridor: { type: 'Polygon' as const, coordinates: [[[10, 47], [10.1, 47], [10.1, 47.1], [10, 47.1], [10, 47]]] },
      };
      mockFetch.mockReturnValueOnce(jsonResponse({ id: 'r1', ...roundData }, 201));

      const result = await api.createRound('m1', roundData);
      expect(result.id).toBe('r1');
      expect(mockFetch).toHaveBeenCalledWith(
        '/api/admin/maps/m1/rounds',
        expect.objectContaining({ method: 'POST' })
      );
    });
  });

  describe('createSession', () => {
    it('creates a session with map_id', async () => {
      const sess = { id: 's1', map_id: 'm1', code: 'ABC123', phase: 'waiting' };
      mockFetch.mockReturnValueOnce(jsonResponse(sess, 201));

      const result = await api.createSession('m1', 300);
      expect(result).toEqual(sess);
    });
  });

  describe('createSoloSession', () => {
    it('creates solo session with player', async () => {
      const resp = {
        session: { id: 's1', is_solo: true },
        player: { id: 'p1', name: 'Player' },
        team: { id: 't1', name: 'Solo' },
      };
      mockFetch.mockReturnValueOnce(jsonResponse(resp, 201));

      const result = await api.createSoloSession('m1', 'Player', 300);
      expect(result.session.id).toBe('s1');
      expect(result.player.name).toBe('Player');
      expect(result.team.name).toBe('Solo');

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(body).toEqual({ map_id: 'm1', player_name: 'Player', time_limit_sec: 300 });
    });
  });

  describe('joinSession', () => {
    it('joins with player name', async () => {
      const player = { id: 'p1', name: 'Alice' };
      mockFetch.mockReturnValueOnce(jsonResponse(player, 201));

      const result = await api.joinSession('s1', 'Alice');
      expect(result).toEqual(player);
    });
  });

  describe('submitRoute', () => {
    it('submits a route for scoring', async () => {
      const route = { id: 'tr1', score: 850 };
      mockFetch.mockReturnValueOnce(jsonResponse(route, 201));

      const path: GeoJSON.LineString = { type: 'LineString', coordinates: [[10, 47], [10.1, 47.1]] };
      const result = await api.submitRoute('s1', 'r1', 't1', path);
      expect(result).toEqual(route);
    });
  });

  describe('error handling', () => {
    it('throws Error with server error message', async () => {
      mockFetch.mockReturnValueOnce(errorResponse('map not found', 404));

      await expect(api.getMap('nonexistent')).rejects.toThrow('map not found');
    });

    it('uses statusText when body has no error field', async () => {
      mockFetch.mockReturnValueOnce(Promise.resolve({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        json: () => Promise.reject(new Error('no json')),
      }));

      await expect(api.listMaps()).rejects.toThrow('Internal Server Error');
    });
  });
});
