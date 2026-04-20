import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import Home from '../components/Home';

// Mock maplibre-gl to avoid canvas issues in jsdom
vi.mock('maplibre-gl', () => ({
  default: {
    Map: vi.fn(),
    Marker: vi.fn(() => ({ setLngLat: vi.fn().mockReturnThis(), setPopup: vi.fn().mockReturnThis(), addTo: vi.fn().mockReturnThis(), remove: vi.fn() })),
    Popup: vi.fn(() => ({ setText: vi.fn().mockReturnThis() })),
    LngLatBounds: vi.fn(),
  },
}));

// Mock api
vi.mock('../services/api', () => ({
  listMaps: vi.fn().mockResolvedValue([]),
  getMap: vi.fn().mockResolvedValue(null),
  createSoloSession: vi.fn(),
  startGame: vi.fn(),
}));

describe('Home page', () => {
  it('renders the title and navigation buttons', () => {
    render(
      <MemoryRouter>
        <Home />
      </MemoryRouter>
    );

    expect(screen.getByText('Sluff')).toBeInTheDocument();
    expect(screen.getByText('Solo Play')).toBeInTheDocument();
    expect(screen.getByText('Join Game')).toBeInTheDocument();
    expect(screen.getByText('Create Session')).toBeInTheDocument();
    expect(screen.getByText('Guide Panel')).toBeInTheDocument();
  });

  it('renders all buttons as clickable', () => {
    render(
      <MemoryRouter>
        <Home />
      </MemoryRouter>
    );

    const buttons = screen.getAllByRole('button');
    expect(buttons).toHaveLength(4);
    buttons.forEach((btn) => {
      expect(btn).toBeEnabled();
    });
  });

  it('displays the subtitle', () => {
    render(
      <MemoryRouter>
        <Home />
      </MemoryRouter>
    );

    expect(screen.getByText('Draw a safe backcountry ski tour')).toBeInTheDocument();
  });
});
