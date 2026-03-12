import { render, screen } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { describe, expect, it, vi } from 'vitest';
import type { DashboardStats } from '../hooks/useDashboardStats';
import type { City } from '../types';
import DashboardPage from './DashboardPage';

vi.mock('@ant-design/icons', () => ({
  AlertOutlined: () => null,
  EnvironmentOutlined: () => null,
  GlobalOutlined: () => null,
  ReadOutlined: () => null,
  WarningOutlined: () => null,
}));

vi.mock('antd', () => ({
  Alert: ({ message, description }: { message: string; description: string }) => (
    <div role="alert">
      <span>{message}</span>
      <span>{description}</span>
    </div>
  ),
  Card: ({ children, title }: PropsWithChildren<{ title?: string }>) => (
    <div>
      {title && <span>{title}</span>}
      {children}
    </div>
  ),
  Col: ({ children }: PropsWithChildren) => <div>{children}</div>,
  Progress: ({ percent, format }: { percent: number; format?: () => string }) => (
    <div data-testid="progress" data-percent={percent}>
      {format?.()}
    </div>
  ),
  Row: ({ children }: PropsWithChildren) => <div>{children}</div>,
  Skeleton: ({ active }: { active: boolean }) => (
    <div data-testid="skeleton" data-active={active} />
  ),
  Statistic: ({ title, value }: { title: string; value: string | number }) => (
    <div data-testid={`stat-${title}`}>
      <span>{title}</span>
      <span data-testid={`stat-value-${title}`}>{value}</span>
    </div>
  ),
  Table: ({
    dataSource,
    loading,
    locale,
  }: {
    dataSource: City[];
    loading: boolean;
    locale?: { emptyText: string };
  }) => (
    <div data-testid="cities-table" data-loading={loading}>
      {dataSource.length > 0
        ? dataSource.map((city) => <div key={city.id}>{city.name}</div>)
        : <div>{locale?.emptyText}</div>}
    </div>
  ),
  Typography: {
    Title: ({ children }: PropsWithChildren) => <h2>{children}</h2>,
  },
}));

const mockUseDashboardStats = vi.fn();

vi.mock('../hooks', () => ({
  useDashboardStats: () => mockUseDashboardStats(),
}));

const sampleStats: DashboardStats = {
  citiesCount: 3,
  poisCount: 42,
  storiesCount: 128,
  reportsCount: 5,
  newReportsCount: 2,
};

const sampleCities: City[] = [
  {
    id: 1,
    name: 'Tbilisi',
    name_ru: null,
    country: 'Georgia',
    center_lat: 41.7151,
    center_lng: 44.8271,
    radius_km: 15,
    is_active: true,
    download_size_mb: 120.5,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  },
];

describe('DashboardPage', () => {
  it('renders stats and cities on success', () => {
    mockUseDashboardStats.mockReturnValue({
      stats: sampleStats,
      isLoading: false,
      isError: false,
      error: null,
      cities: sampleCities,
    });

    render(<DashboardPage />);

    expect(screen.getByText('Dashboard')).toBeInTheDocument();
    expect(screen.getByTestId('stat-value-Cities')).toHaveTextContent('3');
    expect(screen.getByTestId('stat-value-Points of Interest')).toHaveTextContent('42');
    expect(screen.getByTestId('stat-value-Stories')).toHaveTextContent('128');
    expect(screen.getByTestId('stat-value-Reports')).toHaveTextContent('5');
    expect(screen.getByTestId('stat-value-New Reports')).toHaveTextContent('2');
    expect(screen.getByText('Tbilisi')).toBeInTheDocument();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('shows skeletons when data is loading', () => {
    mockUseDashboardStats.mockReturnValue({
      stats: null,
      isLoading: true,
      isError: false,
      error: null,
      cities: [],
    });

    render(<DashboardPage />);

    const skeletons = screen.getAllByTestId('skeleton');
    expect(skeletons.length).toBe(5);
    // No stat values should be rendered while loading
    expect(screen.queryByTestId('stat-value-Cities')).not.toBeInTheDocument();
  });

  it('renders error alert when API call fails', () => {
    mockUseDashboardStats.mockReturnValue({
      stats: null,
      isLoading: false,
      isError: true,
      error: new Error('Failed to fetch admin stats'),
      cities: [],
    });

    render(<DashboardPage />);

    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.getByText('Failed to load dashboard')).toBeInTheDocument();
    expect(screen.getByText('Failed to fetch admin stats')).toBeInTheDocument();
  });

  it('shows zero values correctly without visual bugs', () => {
    mockUseDashboardStats.mockReturnValue({
      stats: { citiesCount: 0, poisCount: 0, storiesCount: 0, reportsCount: 0, newReportsCount: 0 },
      isLoading: false,
      isError: false,
      error: null,
      cities: [],
    });

    render(<DashboardPage />);

    expect(screen.getByTestId('stat-value-Cities')).toHaveTextContent('0');
    expect(screen.getByTestId('stat-value-Points of Interest')).toHaveTextContent('0');
    expect(screen.getByTestId('stat-value-Stories')).toHaveTextContent('0');
    expect(screen.getByTestId('stat-value-Reports')).toHaveTextContent('0');
    expect(screen.getByTestId('stat-value-New Reports')).toHaveTextContent('0');
    expect(screen.getByText('No cities found')).toBeInTheDocument();
  });

  it('renders report resolution progress card', () => {
    mockUseDashboardStats.mockReturnValue({
      stats: sampleStats,
      isLoading: false,
      isError: false,
      error: null,
      cities: sampleCities,
    });

    render(<DashboardPage />);

    expect(screen.getByText('Report Resolution')).toBeInTheDocument();
    const progress = screen.getByTestId('progress');
    expect(progress).toHaveAttribute('data-percent', '60');
    expect(progress).toHaveTextContent('3 / 5 resolved');
  });

  it('does not render report resolution card while loading', () => {
    mockUseDashboardStats.mockReturnValue({
      stats: null,
      isLoading: true,
      isError: false,
      error: null,
      cities: [],
    });

    render(<DashboardPage />);

    expect(screen.queryByText('Report Resolution')).not.toBeInTheDocument();
  });
});
