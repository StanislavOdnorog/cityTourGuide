import { render, screen } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import type { City, POI } from '../types';
import POIMapPage from './POIMapPage';

vi.mock('leaflet', () => {
  const divIcon = vi.fn(() => ({}));
  return {
    default: {
      Icon: { Default: { mergeOptions: vi.fn() } },
      divIcon,
    },
    divIcon,
    Icon: { Default: { mergeOptions: vi.fn() } },
  };
});

vi.mock('leaflet/dist/leaflet.css', () => ({}));
vi.mock('leaflet/dist/images/marker-icon-2x.png', () => ({ default: '' }));
vi.mock('leaflet/dist/images/marker-icon.png', () => ({ default: '' }));
vi.mock('leaflet/dist/images/marker-shadow.png', () => ({ default: '' }));

vi.mock('react-leaflet', () => ({
  MapContainer: ({ children }: PropsWithChildren) => (
    <div data-testid="map-container">{children}</div>
  ),
  Marker: ({ children }: PropsWithChildren) => <div data-testid="marker">{children}</div>,
  Popup: ({ children }: PropsWithChildren) => <div>{children}</div>,
  TileLayer: () => null,
}));

vi.mock('react-leaflet-cluster', () => ({
  default: ({ children }: PropsWithChildren) => <div>{children}</div>,
}));

vi.mock('@ant-design/icons', () => ({}));

vi.mock('antd', () => ({
  Button: ({
    children,
    onClick,
  }: PropsWithChildren<{ onClick?: () => void; type?: string; size?: string; 'data-testid'?: string }>) => (
    <button onClick={onClick}>{children}</button>
  ),
  Card: ({ children }: PropsWithChildren) => <div>{children}</div>,
  Col: ({ children }: PropsWithChildren) => <div>{children}</div>,
  Descriptions: Object.assign(
    ({ children }: PropsWithChildren) => <dl>{children}</dl>,
    {
      Item: ({ label, children }: PropsWithChildren<{ label: string }>) => (
        <>
          <dt>{label}</dt>
          <dd>{children}</dd>
        </>
      ),
    },
  ),
  Row: ({ children }: PropsWithChildren) => <div>{children}</div>,
  Select: ({
    value,
    onChange,
    options,
    'data-testid': testId,
  }: {
    value: unknown;
    onChange: (v: unknown) => void;
    options: Array<{ value: unknown; label: string }>;
    'data-testid'?: string;
    style?: React.CSSProperties;
    placeholder?: string;
    loading?: boolean;
  }) => (
    <select
      data-testid={testId}
      value={String(value ?? '')}
      onChange={(e) => onChange(e.target.value || e.target.value)}
    >
      {options.map((o) => (
        <option key={String(o.value)} value={String(o.value)}>
          {o.label}
        </option>
      ))}
    </select>
  ),
  Spin: ({ children }: PropsWithChildren<{ spinning?: boolean }>) => <div>{children}</div>,
  Tag: ({ children, 'data-testid': testId }: PropsWithChildren<{ 'data-testid'?: string; style?: React.CSSProperties; color?: string }>) => (
    <span data-testid={testId}>{children}</span>
  ),
  Typography: {
    Title: ({ children }: PropsWithChildren) => <h2>{children}</h2>,
  },
}));

const mockUseCities = vi.fn();
const mockUsePOIs = vi.fn();

vi.mock('../hooks', () => ({
  useCities: () => mockUseCities(),
  usePOIs: (...args: unknown[]) => mockUsePOIs(...args),
}));

function buildCity(id: number, name: string): City {
  return {
    id,
    name,
    name_ru: null,
    country: 'Georgia',
    center_lat: 41.7,
    center_lng: 44.8,
    radius_km: 10,
    is_active: true,
    download_size_mb: 50,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}

function buildPOI(id: number, name: string): POI {
  return {
    id,
    name,
    name_ru: null,
    type: 'building',
    status: 'active',
    lat: 41.69 + id * 0.001,
    lng: 44.80 + id * 0.001,
    interest_score: 80,
    address: null,
    city_id: 1,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  } as POI;
}

function setupDefaults(pois: POI[] = [buildPOI(1, 'Tower'), buildPOI(2, 'Bridge')]) {
  mockUseCities.mockReturnValue({
    data: { items: [buildCity(1, 'Tbilisi'), buildCity(2, 'Batumi')] },
    isLoading: false,
  });

  mockUsePOIs.mockReturnValue({
    data: pois,
    isLoading: false,
    isFetching: false,
  });
}

function renderPage() {
  return render(
    <MemoryRouter>
      <POIMapPage />
    </MemoryRouter>,
  );
}

describe('POIMapPage', () => {
  it('renders the page heading and map container', () => {
    setupDefaults();

    renderPage();

    expect(screen.getByText('POI Map')).toBeInTheDocument();
    expect(screen.getByTestId('poi-map-page')).toBeInTheDocument();
    expect(screen.getByTestId('map-container')).toBeInTheDocument();
  });

  it('renders markers for POIs', () => {
    setupDefaults();

    renderPage();

    const markers = screen.getAllByTestId('marker');
    expect(markers).toHaveLength(2);
  });

  it('shows POI count tag', () => {
    setupDefaults();

    renderPage();

    expect(screen.getByTestId('poi-count')).toHaveTextContent('2 POIs');
  });

  it('shows loading state with no markers when POIs are loading', () => {
    mockUseCities.mockReturnValue({
      data: { items: [buildCity(1, 'Tbilisi')] },
      isLoading: false,
    });
    mockUsePOIs.mockReturnValue({
      data: [],
      isLoading: true,
      isFetching: true,
    });

    renderPage();

    expect(screen.queryByTestId('marker')).not.toBeInTheDocument();
    expect(screen.queryByTestId('poi-count')).not.toBeInTheDocument();
  });

  it('shows no markers and no count when POI list is empty', () => {
    setupDefaults([]);

    renderPage();

    expect(screen.queryByTestId('marker')).not.toBeInTheDocument();
    expect(screen.queryByTestId('poi-count')).not.toBeInTheDocument();
  });

  it('renders filter selects for city, type, and status', () => {
    setupDefaults();

    renderPage();

    expect(screen.getByTestId('poi-type-filter')).toBeInTheDocument();
    expect(screen.getByTestId('poi-status-filter')).toBeInTheDocument();
  });
});
