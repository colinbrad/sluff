import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import MapOverlayControls from '../components/map/MapOverlayControls';

describe('MapOverlayControls', () => {
  it('renders 3D and Slope buttons', () => {
    render(
      <MapOverlayControls
        terrain3d={false}
        slopeShading={false}
        onToggleTerrain={() => {}}
        onToggleSlope={() => {}}
      />
    );

    expect(screen.getByText('3D')).toBeInTheDocument();
    expect(screen.getByText('Slope')).toBeInTheDocument();
  });

  it('calls onToggleTerrain when 3D button clicked', () => {
    const onToggle = vi.fn();
    render(
      <MapOverlayControls
        terrain3d={false}
        slopeShading={false}
        onToggleTerrain={onToggle}
        onToggleSlope={() => {}}
      />
    );

    fireEvent.click(screen.getByText('3D'));
    expect(onToggle).toHaveBeenCalledOnce();
  });

  it('calls onToggleSlope when Slope button clicked', () => {
    const onToggle = vi.fn();
    render(
      <MapOverlayControls
        terrain3d={false}
        slopeShading={false}
        onToggleTerrain={() => {}}
        onToggleSlope={onToggle}
      />
    );

    fireEvent.click(screen.getByText('Slope'));
    expect(onToggle).toHaveBeenCalledOnce();
  });

  it('applies active styling when terrain3d is true', () => {
    render(
      <MapOverlayControls
        terrain3d={true}
        slopeShading={false}
        onToggleTerrain={() => {}}
        onToggleSlope={() => {}}
      />
    );

    const btn = screen.getByTitle('Toggle 3D terrain');
    expect(btn.className).toContain('bg-blue-600');
    expect(btn.className).toContain('text-white');
  });

  it('applies inactive styling when terrain3d is false', () => {
    render(
      <MapOverlayControls
        terrain3d={false}
        slopeShading={false}
        onToggleTerrain={() => {}}
        onToggleSlope={() => {}}
      />
    );

    const btn = screen.getByTitle('Toggle 3D terrain');
    expect(btn.className).toContain('bg-white');
    expect(btn.className).toContain('border-gray-300');
  });

  it('applies active styling when slopeShading is true', () => {
    render(
      <MapOverlayControls
        terrain3d={false}
        slopeShading={true}
        onToggleTerrain={() => {}}
        onToggleSlope={() => {}}
      />
    );

    const btn = screen.getByTitle('Toggle slope angle shading');
    expect(btn.className).toContain('bg-blue-600');
    expect(btn.className).toContain('text-white');
  });

  it('applies inactive styling when slopeShading is false', () => {
    render(
      <MapOverlayControls
        terrain3d={false}
        slopeShading={false}
        onToggleTerrain={() => {}}
        onToggleSlope={() => {}}
      />
    );

    const btn = screen.getByTitle('Toggle slope angle shading');
    expect(btn.className).toContain('bg-white');
    expect(btn.className).toContain('border-gray-300');
  });

  it('both buttons can be active simultaneously', () => {
    render(
      <MapOverlayControls
        terrain3d={true}
        slopeShading={true}
        onToggleTerrain={() => {}}
        onToggleSlope={() => {}}
      />
    );

    expect(screen.getByTitle('Toggle 3D terrain').className).toContain('bg-blue-600');
    expect(screen.getByTitle('Toggle slope angle shading').className).toContain('bg-blue-600');
  });
});
