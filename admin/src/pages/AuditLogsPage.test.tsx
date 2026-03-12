import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import type { AuditLogItem } from '../types';
import AuditLogsPage from './AuditLogsPage';

vi.mock('@ant-design/icons', () => ({
  AuditOutlined: () => null,
}));

vi.mock('antd', async () => {
  return {
    Card: ({ children }: PropsWithChildren) => <div>{children}</div>,
    Input: ({
      value,
      onChange,
      type,
      placeholder,
      'data-testid': testId,
    }: {
      value?: string;
      onChange?: (event: { target: { value: string } }) => void;
      type?: string;
      placeholder?: string;
      'data-testid'?: string;
    }) => (
      <input
        data-testid={testId}
        value={value}
        onChange={(event) => onChange?.({ target: { value: event.target.value } })}
        type={type}
        placeholder={placeholder}
      />
    ),
    Select: ({
      value,
      onChange,
      options,
      'data-testid': testId,
    }: {
      value: string;
      onChange: (value: string) => void;
      options: Array<{ value: string; label: string }>;
      'data-testid'?: string;
    }) => (
      <select data-testid={testId} value={value} onChange={(event) => onChange(event.target.value)}>
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
      locale,
    }: {
      dataSource: AuditLogItem[];
      pagination: {
        current: number;
        pageSize: number;
        total: number;
        onChange: (page: number, pageSize: number) => void;
      };
      locale?: { emptyText?: string };
    }) => (
      <div>
        <div data-testid="row-ids">{dataSource.map((item) => item.id).join(',')}</div>
        <div data-testid="page-info">{`page=${pagination.current} size=${pagination.pageSize}`}</div>
        <div data-testid="empty-text">{locale?.emptyText}</div>
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
    Typography: {
      Title: ({ children }: PropsWithChildren) => <h2>{children}</h2>,
      Text: ({ children, ...props }: PropsWithChildren<Record<string, unknown>>) => (
        <span {...props}>{children}</span>
      ),
    },
  };
});

const useAuditLogs = vi.fn();

vi.mock('../hooks/useAuditLogs', () => ({
  useAuditLogs: (args: unknown) => useAuditLogs(args),
}));

function buildLog(id: number, overrides: Partial<AuditLogItem> = {}): AuditLogItem {
  return {
    id,
    actor_id: `admin-${id}`,
    action: 'create',
    resource_type: 'city',
    resource_id: String(id),
    http_method: 'POST',
    request_path: '/api/v1/admin/cities',
    trace_id: `trace-${id}`,
    status: 'success',
    created_at: `2026-01-${String(id).padStart(2, '0')}T00:00:00Z`,
    ...overrides,
  };
}

function setupMockData() {
  useAuditLogs.mockImplementation(
    ({
      actorId = '',
      action = '',
      resourceType = '',
      createdFrom = '',
      createdTo = '',
      cursor,
      limit,
    }: {
      actorId?: string;
      action?: string;
      resourceType?: string;
      createdFrom?: string;
      createdTo?: string;
      cursor?: string;
      limit?: number;
    }) => {
      const key = `${actorId || 'all'}|${action || 'all'}|${resourceType || 'all'}|${createdFrom || 'all'}|${createdTo || 'all'}|${cursor ?? 'root'}|${limit ?? 20}`;
      const dataByKey: Record<
        string,
        { items: AuditLogItem[]; next_cursor: string; has_more: boolean }
      > = {
        'all|all|all|all|all|root|20': {
          items: [buildLog(1), buildLog(2)],
          next_cursor: 'cursor-2',
          has_more: true,
        },
        'all|all|all|all|all|cursor-2|20': {
          items: [buildLog(3)],
          next_cursor: '',
          has_more: false,
        },
        'all|create|all|all|all|root|20': {
          items: [buildLog(11, { action: 'create' })],
          next_cursor: '',
          has_more: false,
        },
        'all|all|poi|all|all|root|20': {
          items: [buildLog(21, { resource_type: 'poi' })],
          next_cursor: '',
          has_more: false,
        },
        'admin-77|all|all|all|all|root|20': {
          items: [buildLog(77, { actor_id: 'admin-77' })],
          next_cursor: '',
          has_more: false,
        },
        'all|all|all|2026-01-01T00:00:00Z|2026-01-31T23:59:59Z|root|20': {
          items: [buildLog(31)],
          next_cursor: '',
          has_more: false,
        },
        'all|all|all|all|all|root|50': {
          items: [buildLog(1), buildLog(2), buildLog(3)],
          next_cursor: '',
          has_more: false,
        },
        'all|create|all|all|all|root|50': {
          items: [buildLog(11, { action: 'create' })],
          next_cursor: '',
          has_more: false,
        },
      };

      return {
        logs: {
          data: dataByKey[key] ?? { items: [], next_cursor: '', has_more: false },
          isLoading: false,
        },
      };
    },
  );
}

function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="location">{location.pathname + location.search}</div>;
}

function renderWithRouter(initialEntry = '/audit-logs') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <AuditLogsPage />
      <LocationDisplay />
    </MemoryRouter>,
  );
}

describe('AuditLogsPage', () => {
  it('renders the page and displays log entries', async () => {
    setupMockData();

    renderWithRouter();

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('1,2');
    });
    expect(useAuditLogs).toHaveBeenLastCalledWith({
      actorId: '',
      action: '',
      resourceType: '',
      createdFrom: '',
      createdTo: '',
      cursor: undefined,
      limit: 20,
    });
  });

  it('navigates pages using cursor pagination', async () => {
    setupMockData();

    renderWithRouter();

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('1,2');
    });

    fireEvent.click(screen.getByRole('button', { name: 'next' }));

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('3');
    });
    expect(useAuditLogs).toHaveBeenLastCalledWith({
      actorId: '',
      action: '',
      resourceType: '',
      createdFrom: '',
      createdTo: '',
      cursor: 'cursor-2',
      limit: 20,
    });

    fireEvent.click(screen.getByRole('button', { name: 'page-1' }));

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('1,2');
    });
  });

  it('resets pagination when the action filter changes', async () => {
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

    // Change action filter — should reset to page 1
    fireEvent.change(screen.getByTestId('action-filter'), { target: { value: 'create' } });

    await waitFor(() => {
      expect(screen.getByTestId('page-info')).toHaveTextContent('page=1 size=20');
    });
    expect(screen.getByTestId('row-ids')).toHaveTextContent('11');
  });

  it('filters by resource type', async () => {
    setupMockData();

    renderWithRouter();

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('1,2');
    });

    fireEvent.change(screen.getByTestId('resource-type-filter'), { target: { value: 'poi' } });

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('21');
    });
    expect(useAuditLogs).toHaveBeenLastCalledWith({
      actorId: '',
      action: '',
      resourceType: 'poi',
      createdFrom: '',
      createdTo: '',
      cursor: undefined,
      limit: 20,
    });
  });

  it('filters by actor id and resets pagination', async () => {
    setupMockData();

    renderWithRouter();

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('1,2');
    });

    fireEvent.click(screen.getByRole('button', { name: 'next' }));
    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('3');
    });

    fireEvent.change(screen.getByTestId('actor-id-filter'), { target: { value: 'admin-77' } });

    await waitFor(() => {
      expect(screen.getByTestId('page-info')).toHaveTextContent('page=1 size=20');
    });
    expect(screen.getByTestId('row-ids')).toHaveTextContent('77');
    expect(useAuditLogs).toHaveBeenLastCalledWith({
      actorId: 'admin-77',
      action: '',
      resourceType: '',
      createdFrom: '',
      createdTo: '',
      cursor: undefined,
      limit: 20,
    });
  });

  it('filters by created-at window', async () => {
    setupMockData();

    renderWithRouter();

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('1,2');
    });

    fireEvent.change(screen.getByTestId('created-from-filter'), {
      target: { value: '2026-01-01' },
    });
    fireEvent.change(screen.getByTestId('created-to-filter'), { target: { value: '2026-01-31' } });

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('31');
    });
    expect(useAuditLogs).toHaveBeenLastCalledWith({
      actorId: '',
      action: '',
      resourceType: '',
      createdFrom: '2026-01-01T00:00:00Z',
      createdTo: '2026-01-31T23:59:59Z',
      cursor: undefined,
      limit: 20,
    });
  });

  it('resets to page 1 when page size changes', async () => {
    setupMockData();

    renderWithRouter();

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('1,2');
    });

    fireEvent.click(screen.getByRole('button', { name: 'next' }));
    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('3');
    });

    fireEvent.click(screen.getByRole('button', { name: 'change-size-50' }));

    await waitFor(() => {
      expect(screen.getByTestId('page-info')).toHaveTextContent('page=1 size=50');
    });
    expect(screen.getByTestId('row-ids')).toHaveTextContent('1,2,3');
  });

  it('shows empty state when no logs match', async () => {
    useAuditLogs.mockReturnValue({
      logs: {
        data: { items: [], next_cursor: '', has_more: false },
        isLoading: false,
      },
    });

    renderWithRouter();

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('');
    });
    expect(screen.getByTestId('empty-text')).toHaveTextContent(
      'No audit logs match the current filters.',
    );
  });

  it('shows loading state while logs are being fetched', () => {
    useAuditLogs.mockReturnValue({
      logs: {
        data: undefined,
        isLoading: true,
      },
    });

    renderWithRouter();

    expect(screen.getByTestId('audit-logs-loading')).toHaveTextContent('Loading audit logs...');
    expect(screen.getByTestId('empty-text')).toHaveTextContent('Loading audit logs...');
  });

  it('falls back to page 1 when URL has page=2 but no cursor is available', async () => {
    setupMockData();

    renderWithRouter('/audit-logs?page=2');

    await waitFor(() => {
      expect(screen.getByTestId('page-info')).toHaveTextContent('page=1 size=20');
    });
    expect(screen.getByTestId('row-ids')).toHaveTextContent('1,2');
  });

  it('updates URL when action filter changes', async () => {
    setupMockData();

    renderWithRouter();

    await waitFor(() => {
      expect(screen.getByTestId('row-ids')).toHaveTextContent('1,2');
    });

    fireEvent.change(screen.getByTestId('action-filter'), { target: { value: 'create' } });

    await waitFor(() => {
      expect(screen.getByTestId('location')).toHaveTextContent('/audit-logs?action=create');
    });
  });
});
