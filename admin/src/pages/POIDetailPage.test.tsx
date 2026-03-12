import { fireEvent, render, screen } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import POIDetailPage from './POIDetailPage';

const mockNavigate = vi.fn();
let mockParams: Record<string, string> = { id: '1' };
const mockSearchParams = new URLSearchParams();

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useParams: () => mockParams,
    useSearchParams: () => [mockSearchParams],
  };
});

vi.mock('@ant-design/icons', () => ({
  ArrowLeftOutlined: () => null,
  PauseCircleOutlined: () => null,
  PlayCircleOutlined: () => null,
  ThunderboltOutlined: () => null,
}));

vi.mock('antd', () => ({
  App: {
    useApp: () => ({
      message: { success: vi.fn(), error: vi.fn() },
    }),
  },
  Badge: ({ children }: PropsWithChildren) => <span>{children}</span>,
  Button: ({
    children,
    onClick,
    loading,
  }: PropsWithChildren<{ onClick?: () => void; loading?: boolean; icon?: React.ReactNode; type?: string; size?: string; block?: boolean }>) => (
    <button onClick={onClick} disabled={loading}>
      {children}
    </button>
  ),
  Card: ({ children, title }: PropsWithChildren<{ title?: React.ReactNode; extra?: React.ReactNode; style?: React.CSSProperties }>) => (
    <div>
      {title && <div>{title}</div>}
      {children}
    </div>
  ),
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
  Empty: ({ description }: { description: string }) => (
    <div data-testid="empty">{description}</div>
  ),
  Row: ({ children }: PropsWithChildren) => <div>{children}</div>,
  Space: ({ children }: PropsWithChildren) => <div>{children}</div>,
  Spin: ({ size }: { size: string }) => <div data-testid="spinner" data-size={size} />,
  Switch: ({
    checked,
    loading,
    onChange,
  }: {
    checked?: boolean;
    loading?: boolean;
    onChange?: (checked: boolean) => void;
    size?: string;
  }) => (
    <input
      type="checkbox"
      checked={checked}
      disabled={loading}
      onChange={(e) => onChange?.(e.target.checked)}
      data-testid="poi-status-switch"
    />
  ),
  Table: ({
    dataSource,
    loading,
  }: {
    dataSource: unknown[];
    loading: boolean;
    columns?: unknown[];
    rowKey?: string;
    size?: string;
    pagination?: unknown;
    scroll?: unknown;
    locale?: unknown;
    rowClassName?: unknown;
  }) => (
    <div data-testid="table" data-loading={loading}>
      {dataSource.length} items
    </div>
  ),
  Tabs: ({
    items,
  }: {
    items: Array<{ key: string; label: React.ReactNode; children: React.ReactNode }>;
  }) => (
    <div data-testid="tabs">
      {items.map((item) => (
        <div key={item.key} data-testid={`tab-${item.key}`}>
          <span>{item.label}</span>
          {item.children}
        </div>
      ))}
    </div>
  ),
  Tag: ({ children }: PropsWithChildren) => <span>{children}</span>,
  Typography: {
    Title: ({ children }: PropsWithChildren) => <h4>{children}</h4>,
    Text: ({ children }: PropsWithChildren) => <span>{children}</span>,
    Paragraph: ({ children }: PropsWithChildren) => <p>{children}</p>,
  },
}));

const mockUsePOIDetail = vi.fn();

vi.mock('../hooks', () => ({
  usePOIDetail: (...args: unknown[]) => mockUsePOIDetail(...args),
}));

const samplePOI = {
  id: 1,
  name: 'Old Tower',
  name_ru: null,
  type: 'building',
  status: 'active' as const,
  lat: 41.69411,
  lng: 44.80139,
  interest_score: 85,
  address: '1 Main St',
  city_id: 1,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

const sampleStories = [
  { id: 10, language: 'en', layer_type: 'atmosphere', text: 'Story text', audio_url: null, duration_sec: null, confidence: 90, status: 'active', is_inflation: false },
];

const sampleReports = [
  { id: 20, story_id: 10, type: 'wrong_fact', comment: null, status: 'new', created_at: '2026-01-05T00:00:00Z' },
];

const sampleJobs = [
  { id: 30, status: 'completed', trigger_type: 'manual', segments_count: 2, max_segments: 3, created_at: '2026-01-03T00:00:00Z', error_log: null },
];

function setupDefaults() {
  mockUsePOIDetail.mockReturnValue({
    poi: { data: samplePOI, isLoading: false },
    stories: { data: sampleStories, isLoading: false },
    reports: { data: sampleReports, isLoading: false },
    inflationJobs: { data: sampleJobs, isLoading: false },
    updatePOI: { mutate: vi.fn(), isPending: false },
    toggleStoryStatus: { mutate: vi.fn(), isPending: false },
    triggerInflation: { mutate: vi.fn(), isPending: false },
  });
}

function renderPage() {
  return render(
    <MemoryRouter>
      <POIDetailPage />
    </MemoryRouter>,
  );
}

describe('POIDetailPage', () => {
  it('renders POI details on success', () => {
    mockParams = { id: '1' };
    setupDefaults();

    renderPage();

    expect(screen.getByText('Old Tower')).toBeInTheDocument();
    expect(screen.getByText('building')).toBeInTheDocument();
    expect(screen.getByText('85')).toBeInTheDocument();
    expect(screen.getByText('1 Main St')).toBeInTheDocument();
  });

  it('shows spinner during loading', () => {
    mockParams = { id: '1' };
    mockUsePOIDetail.mockReturnValue({
      poi: { data: undefined, isLoading: true },
      stories: { data: [], isLoading: false },
      reports: { data: [], isLoading: false },
      inflationJobs: { data: [], isLoading: false },
      updatePOI: { mutate: vi.fn(), isPending: false },
      toggleStoryStatus: { mutate: vi.fn(), isPending: false },
      triggerInflation: { mutate: vi.fn(), isPending: false },
    });

    renderPage();

    expect(screen.getByTestId('spinner')).toBeInTheDocument();
  });

  it('shows empty state when POI is not found', () => {
    mockParams = { id: '999' };
    mockUsePOIDetail.mockReturnValue({
      poi: { data: null, isLoading: false },
      stories: { data: [], isLoading: false },
      reports: { data: [], isLoading: false },
      inflationJobs: { data: [], isLoading: false },
      updatePOI: { mutate: vi.fn(), isPending: false },
      toggleStoryStatus: { mutate: vi.fn(), isPending: false },
      triggerInflation: { mutate: vi.fn(), isPending: false },
    });

    renderPage();

    expect(screen.getByTestId('empty')).toHaveTextContent('POI not found');
  });

  it('renders tabs for stories, reports, and inflation jobs', () => {
    mockParams = { id: '1' };
    setupDefaults();

    renderPage();

    expect(screen.getByTestId('tab-stories')).toBeInTheDocument();
    expect(screen.getByTestId('tab-reports')).toBeInTheDocument();
    expect(screen.getByTestId('tab-inflation')).toBeInTheDocument();
  });

  it('navigates back when the Back button is clicked', () => {
    mockParams = { id: '1' };
    setupDefaults();

    renderPage();

    fireEvent.click(screen.getByRole('button', { name: 'Back' }));

    expect(mockNavigate).toHaveBeenCalledWith(-1);
  });

  it('calls triggerInflation when Trigger Inflation button is clicked', () => {
    mockParams = { id: '1' };
    const mutateFn = vi.fn();
    mockUsePOIDetail.mockReturnValue({
      poi: { data: samplePOI, isLoading: false },
      stories: { data: sampleStories, isLoading: false },
      reports: { data: sampleReports, isLoading: false },
      inflationJobs: { data: sampleJobs, isLoading: false },
      updatePOI: { mutate: vi.fn(), isPending: false },
      toggleStoryStatus: { mutate: vi.fn(), isPending: false },
      triggerInflation: { mutate: mutateFn, isPending: false },
    });

    renderPage();

    fireEvent.click(screen.getByRole('button', { name: 'Trigger Inflation' }));

    expect(mutateFn).toHaveBeenCalled();
  });
});
