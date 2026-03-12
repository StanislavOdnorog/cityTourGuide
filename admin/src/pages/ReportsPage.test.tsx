import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import type { AdminReportListItem } from '../types';
import ReportsPage from './ReportsPage';

vi.mock('@ant-design/icons', () => ({
  CheckCircleOutlined: () => null,
  CloseCircleOutlined: () => null,
  StopOutlined: () => null,
}));

vi.mock('antd', async () => {
  return {
    App: {
      useApp: () => ({
        message: {
          success: vi.fn(),
          error: vi.fn(),
        },
      }),
    },
    Button: ({ children, onClick, disabled }: PropsWithChildren<{ onClick?: () => void; disabled?: boolean }>) => (
      <button onClick={onClick} disabled={disabled}>
        {children}
      </button>
    ),
    Card: ({ children }: PropsWithChildren) => <div>{children}</div>,
    Select: ({
      value,
      onChange,
      options,
    }: {
      value: string;
      onChange: (value: string) => void;
      options: Array<{ value: string; label: string }>;
    }) => (
      <select data-testid="status-filter" value={value} onChange={(event) => onChange(event.target.value)}>
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    ),
    Space: ({ children }: PropsWithChildren) => <div>{children}</div>,
    Table: ({
      dataSource,
      pagination,
    }: {
      dataSource: AdminReportListItem[];
      pagination: {
        current: number;
        pageSize: number;
        total: number;
        onChange: (page: number, pageSize: number) => void;
      };
    }) => (
      <div>
        <div data-testid="row-ids">{dataSource.map((item) => item.id).join(',')}</div>
        <div data-testid="page-info">{`page=${pagination.current} size=${pagination.pageSize}`}</div>
        <button onClick={() => pagination.onChange(1, pagination.pageSize)}>page-1</button>
        <button onClick={() => pagination.onChange(2, pagination.pageSize)}>page-2</button>
        <button
          onClick={() => pagination.onChange(pagination.current + 1, pagination.pageSize)}
          disabled={pagination.current * pagination.pageSize >= pagination.total}
        >
          next
        </button>
        <button onClick={() => pagination.onChange(1, 50)}>change-size-50</button>
      </div>
    ),
    Tag: ({ children }: PropsWithChildren) => <span>{children}</span>,
    Tooltip: ({ children }: PropsWithChildren) => <>{children}</>,
    Typography: {
      Title: ({ children }: PropsWithChildren) => <h2>{children}</h2>,
      Text: ({ children }: PropsWithChildren) => <span>{children}</span>,
    },
  };
});

const useReports = vi.fn();

vi.mock('../hooks/useReports', () => ({
  useReports: (args: unknown) => useReports(args),
}));

function buildReport(id: number): AdminReportListItem {
  return {
    id,
    story_id: id * 10,
    user_id: `user-${id}`,
    poi_id: id * 100,
    poi_name: `POI ${id}`,
    story_language: 'en',
    story_status: 'active',
    type: 'wrong_fact',
    comment: null,
    status: 'new',
    created_at: `2026-01-${String(id).padStart(2, '0')}T00:00:00Z`,
  };
}

function setupMockData() {
  useReports.mockImplementation(
    ({
      status = '',
      cursor,
      limit,
    }: {
      status?: string;
      cursor?: string;
      limit?: number;
    }) => {
      const key = `${status || 'all'}|${cursor ?? 'root'}|${limit ?? 20}`;
      const dataByKey: Record<string, { items: AdminReportListItem[]; next_cursor: string; has_more: boolean }> = {
        'all|root|20': {
          items: [buildReport(1), buildReport(2)],
          next_cursor: 'cursor-2',
          has_more: true,
        },
        'all|cursor-2|20': {
          items: [buildReport(3)],
          next_cursor: '',
          has_more: false,
        },
        'new|root|20': {
          items: [buildReport(11)],
          next_cursor: '',
          has_more: false,
        },
        'all|root|50': {
          items: [buildReport(1), buildReport(2), buildReport(3)],
          next_cursor: '',
          has_more: false,
        },
        'new|root|50': {
          items: [buildReport(11)],
          next_cursor: '',
          has_more: false,
        },
      };

      return {
        reports: {
          data: dataByKey[key] ?? { items: [], next_cursor: '', has_more: false },
          isLoading: false,
        },
        updateStatus: {
          mutate: vi.fn(),
          isPending: false,
        },
        disableStory: {
          mutate: vi.fn(),
          isPending: false,
        },
      };
    },
  );
}

/**
 * Helper to capture the current URL from the MemoryRouter.
 * We render a small LocationDisplay component alongside ReportsPage.
 */
function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="location">{location.pathname + location.search}</div>;
}

function renderWithRouter(initialEntry = '/reports') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <ReportsPage />
      <LocationDisplay />
    </MemoryRouter>,
  );
}

describe('ReportsPage', () => {
  it('navigates with cached cursors and resets pagination when the status filter changes', async () => {
    setupMockData();

    renderWithRouter();

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('1,2');
    });
    expect(useReports).toHaveBeenLastCalledWith({ status: '', cursor: undefined, limit: 20 });

    fireEvent.click(screen.getByRole('button', { name: 'next' }));

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('3');
    });
    expect(useReports).toHaveBeenLastCalledWith({ status: '', cursor: 'cursor-2', limit: 20 });

    fireEvent.click(screen.getByRole('button', { name: 'page-1' }));

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('1,2');
    });
    expect(useReports).toHaveBeenLastCalledWith({ status: '', cursor: undefined, limit: 20 });

    fireEvent.change(screen.getByTestId('status-filter'), { target: { value: 'new' } });

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('11');
    });
    expect(useReports).toHaveBeenLastCalledWith({ status: 'new', cursor: undefined, limit: 20 });
  });

  it('initializes from URL search params: status=new&pageSize=50', async () => {
    setupMockData();

    renderWithRouter('/reports?status=new&pageSize=50');

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('11');
    });
    expect(useReports).toHaveBeenLastCalledWith({ status: 'new', cursor: undefined, limit: 50 });
    expect(screen.getByTestId('page-info')).toHaveTextContent('page=1 size=50');
    expect(screen.getByTestId('status-filter')).toHaveValue('new');
  });

  it('updates the URL when the status filter changes', async () => {
    setupMockData();

    renderWithRouter();

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('1,2');
    });

    fireEvent.change(screen.getByTestId('status-filter'), { target: { value: 'new' } });

    await waitFor(() => {
      expect(screen.getByTestId('location')).toHaveTextContent('/reports?status=new');
    });
  });

  it('resets to page 1 when page size changes', async () => {
    setupMockData();

    renderWithRouter();

    // Go to page 2
    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('1,2');
    });
    fireEvent.click(screen.getByRole('button', { name: 'next' }));
    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('3');
    });

    // Change page size
    fireEvent.click(screen.getByRole('button', { name: 'change-size-50' }));

    await waitFor(() => {
      expect(screen.getByTestId('page-info')).toHaveTextContent('page=1 size=50');
    });
    expect(screen.getByTestId('row-ids')).toHaveTextContent('1,2,3');
    expect(useReports).toHaveBeenLastCalledWith({ status: '', cursor: undefined, limit: 50 });
  });

  it('resets to page 1 when status filter changes while on page 2', async () => {
    setupMockData();

    renderWithRouter();

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('1,2');
    });

    // Navigate to page 2
    fireEvent.click(screen.getByRole('button', { name: 'next' }));
    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('3');
    });

    // Change filter — should go back to page 1
    fireEvent.change(screen.getByTestId('status-filter'), { target: { value: 'new' } });

    await waitFor(() => {
      expect(screen.getByTestId('page-info')).toHaveTextContent('page=1 size=20');
    });
    expect(screen.getByTestId('row-ids')).toHaveTextContent('11');
  });

  it('falls back to page 1 when URL has page=2 but no cursor is available', async () => {
    setupMockData();

    // Start at page=2 but we have no cursor chain
    renderWithRouter('/reports?page=2');

    await waitFor(() => {
      // Should fall back to page 1 since cursor for page 2 is unknown
      expect(screen.getByTestId('page-info')).toHaveTextContent('page=1 size=20');
    });
    expect(screen.getByTestId('row-ids')).toHaveTextContent('1,2');
  });

  it('updates URL query string when navigating pages', async () => {
    setupMockData();

    renderWithRouter();

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('1,2');
    });

    // Navigate to page 2
    fireEvent.click(screen.getByRole('button', { name: 'next' }));

    await waitFor(() => {
      expect(screen.getByTestId('location')).toHaveTextContent('/reports?page=2');
    });

    // Navigate back to page 1 — page param should be removed
    fireEvent.click(screen.getByRole('button', { name: 'page-1' }));

    await waitFor(() => {
      expect(screen.getByTestId('location')).toHaveTextContent('/reports');
    });
  });
});
