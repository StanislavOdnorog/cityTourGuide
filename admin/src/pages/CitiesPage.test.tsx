import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { describe, expect, it, vi } from 'vitest';
import type { City } from '../types';
import CitiesPage from './CitiesPage';

vi.mock('@ant-design/icons', () => ({
  PlusOutlined: () => null,
  EditOutlined: () => null,
  DeleteOutlined: () => null,
  UndoOutlined: () => null,
  ExclamationCircleOutlined: () => null,
}));

vi.mock('react-leaflet', () => ({
  MapContainer: ({ children }: PropsWithChildren) => <div data-testid="map">{children}</div>,
  TileLayer: () => null,
  Circle: () => null,
  useMap: () => ({ setView: vi.fn() }),
}));

const mockConfirm = vi.fn();

vi.mock('antd', () => {
  let formValues: Record<string, unknown> = {};
  const formInstance = {
    validateFields: () => Promise.resolve(formValues),
    setFieldsValue: (vals: Record<string, unknown>) => {
      formValues = { ...formValues, ...vals };
    },
    resetFields: () => {
      formValues = {};
    },
  };

  return {
    App: {
      useApp: () => ({
        message: { success: vi.fn(), error: vi.fn() },
      }),
    },
    Button: ({
      children,
      onClick,
      loading,
    }: PropsWithChildren<{ onClick?: () => void; loading?: boolean }>) => (
      <button onClick={onClick} disabled={loading}>
        {children}
      </button>
    ),
    Drawer: ({
      children,
      open,
      title,
    }: PropsWithChildren<{ open: boolean; title: string; onClose?: () => void; width?: number; extra?: React.ReactNode }>) =>
      open ? (
        <div data-testid="drawer">
          <span>{title}</span>
          {children}
        </div>
      ) : null,
    Form: Object.assign(
      ({ children }: PropsWithChildren) => <div>{children}</div>,
      {
        Item: ({ children, label }: PropsWithChildren<{ name?: string; label?: string; rules?: unknown[]; valuePropName?: string; style?: React.CSSProperties }>) => (
          <div>
            {label && <label>{label}</label>}
            {children}
          </div>
        ),
        useForm: () => [formInstance],
        useWatch: () => undefined,
      },
    ),
    Input: Object.assign(
      () => <input />,
      {
        Search: ({
          onSearch,
          placeholder,
        }: {
          onSearch?: (value: string) => void;
          placeholder?: string;
          allowClear?: boolean;
          style?: React.CSSProperties;
          onChange?: (e: { target: { value: string } }) => void;
        }) => (
          <input
            data-testid="search-input"
            placeholder={placeholder}
            onChange={(e) => onSearch?.(e.target.value)}
          />
        ),
      },
    ),
    InputNumber: () => <input type="number" />,
    Modal: { confirm: (...args: unknown[]) => mockConfirm(...args) },
    Select: ({
      value,
      onChange,
      options,
    }: {
      value: string;
      onChange: (value: string) => void;
      options: Array<{ value: string; label: string }>;
      style?: React.CSSProperties;
    }) => (
      <select
        data-testid="visibility-filter"
        value={value}
        onChange={(e) => onChange(e.target.value as 'all' | 'visible' | 'deleted')}
      >
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    ),
    Space: ({ children }: PropsWithChildren) => <div>{children}</div>,
    Switch: () => <input type="checkbox" />,
    Table: ({
      dataSource,
      loading,
      pagination,
    }: {
      dataSource: City[];
      loading: boolean;
      columns: unknown[];
      rowKey: string;
      rowClassName?: (city: City) => string;
      pagination: {
        current: number;
        pageSize: number;
        total: number;
        onChange: (page: number, pageSize: number) => void;
      };
    }) => (
      <div data-testid="cities-table" data-loading={loading}>
        <div data-testid="row-names">
          {dataSource.map((city) => city.name).join(',')}
        </div>
        <div data-testid="page-info">
          {`page=${pagination.current} size=${pagination.pageSize}`}
        </div>
        {dataSource.map((city) => (
          <div key={city.id} data-testid={`city-row-${city.id}`}>
            {city.name}
            {city.is_active ? ' Active' : ' Inactive'}
          </div>
        ))}
      </div>
    ),
    Tag: ({ children }: PropsWithChildren) => <span>{children}</span>,
  };
});

const mockUseCities = vi.fn();
const mockUseCityManagement = vi.fn();

vi.mock('../hooks', () => ({
  useCities: (...args: unknown[]) => mockUseCities(...args),
  useCityManagement: () => mockUseCityManagement(),
}));

vi.mock('../hooks/useCursorTableState', () => ({
  useCursorTableState: () => ({
    page: 1,
    pageSize: 20,
    cursor: undefined,
    setPageAndSize: vi.fn(),
    recordNextCursor: vi.fn(),
    resetPagination: vi.fn(),
  }),
}));

function buildCity(id: number, overrides: Partial<City> = {}): City {
  return {
    id,
    name: `City ${id}`,
    name_ru: null,
    country: 'Country',
    center_lat: 41.0,
    center_lng: 44.0,
    radius_km: 10,
    is_active: true,
    download_size_mb: 50,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

function setupDefaults(cities: City[] = [buildCity(1), buildCity(2)]) {
  mockUseCities.mockReturnValue({
    data: { items: cities, next_cursor: '', has_more: false },
    isLoading: false,
  });

  mockUseCityManagement.mockReturnValue({
    createCity: { mutate: vi.fn(), isPending: false },
    updateCity: { mutate: vi.fn(), isPending: false },
    deleteCity: { mutate: vi.fn(), isPending: false },
    restoreCity: { mutate: vi.fn(), isPending: false },
    toUpdateRequest: (city: City) => city,
  });
}

describe('CitiesPage', () => {
  it('renders the table with city data', () => {
    setupDefaults();

    render(<CitiesPage />);

    expect(screen.getByTestId('row-names')).toHaveTextContent('City 1,City 2');
    expect(screen.getByRole('button', { name: 'Create City' })).toBeInTheDocument();
  });

  it('shows loading state', () => {
    mockUseCities.mockReturnValue({
      data: undefined,
      isLoading: true,
    });
    mockUseCityManagement.mockReturnValue({
      createCity: { mutate: vi.fn(), isPending: false },
      updateCity: { mutate: vi.fn(), isPending: false },
      deleteCity: { mutate: vi.fn(), isPending: false },
      restoreCity: { mutate: vi.fn(), isPending: false },
      toUpdateRequest: vi.fn(),
    });

    render(<CitiesPage />);

    expect(screen.getByTestId('cities-table')).toHaveAttribute('data-loading', 'true');
  });

  it('shows empty table when no cities exist', () => {
    setupDefaults([]);

    render(<CitiesPage />);

    expect(screen.getByTestId('row-names')).toHaveTextContent('');
  });

  it('filters cities by search text', async () => {
    setupDefaults([buildCity(1, { name: 'Tbilisi' }), buildCity(2, { name: 'Batumi' })]);

    render(<CitiesPage />);

    expect(screen.getByTestId('row-names')).toHaveTextContent('Tbilisi,Batumi');

    fireEvent.change(screen.getByTestId('search-input'), { target: { value: 'Tbil' } });

    await waitFor(() => {
      expect(screen.getByTestId('row-names')).toHaveTextContent('Tbilisi');
    });
    expect(screen.getByTestId('row-names')).not.toHaveTextContent('Batumi');
  });

  it('opens create drawer when Create City button is clicked', () => {
    setupDefaults();

    render(<CitiesPage />);

    expect(screen.queryByTestId('drawer')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Create City' }));

    expect(screen.getByTestId('drawer')).toBeInTheDocument();
    expect(screen.getByText('Create City', { selector: '[data-testid="drawer"] span' })).toBeInTheDocument();
  });
});
