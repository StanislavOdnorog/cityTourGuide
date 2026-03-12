import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import LoginPage from './LoginPage';

const mockNavigate = vi.fn();
const mockLocation = { state: null, pathname: '/login', search: '', hash: '', key: 'default' };

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useLocation: () => mockLocation,
  };
});

vi.mock('@ant-design/icons', () => ({
  LockOutlined: () => null,
  UserOutlined: () => null,
}));

const mockMessageSuccess = vi.fn();
const mockMessageError = vi.fn();

vi.mock('antd', () => {
  // Minimal form state for testing
  let formValues: Record<string, string> = {};
  const formInstance = {
    validateFields: () => Promise.resolve(formValues),
    setFieldsValue: (vals: Record<string, string>) => { formValues = { ...formValues, ...vals }; },
    resetFields: () => { formValues = {}; },
  };

  return {
    App: {
      useApp: () => ({
        message: {
          success: mockMessageSuccess,
          error: mockMessageError,
        },
      }),
    },
    Button: ({
      children,
      onClick,
      loading,
      htmlType,
      ...rest
    }: PropsWithChildren<{
      onClick?: () => void;
      loading?: boolean;
      htmlType?: string;
      type?: string;
      block?: boolean;
    }>) => (
      <button
        onClick={onClick}
        disabled={loading}
        type={htmlType === 'submit' ? 'submit' : 'button'}
        data-testid={rest['data-testid' as keyof typeof rest] as string}
      >
        {children}
      </button>
    ),
    Card: ({ children, 'data-testid': testId }: PropsWithChildren<{ 'data-testid'?: string }>) => (
      <div data-testid={testId}>{children}</div>
    ),
    Form: Object.assign(
      ({
        children,
        onFinish,
        'data-testid': testId,
      }: PropsWithChildren<{
        onFinish?: (values: Record<string, string>) => void;
        name?: string;
        autoComplete?: string;
        layout?: string;
        size?: string;
        'data-testid'?: string;
      }>) => (
        <form
          data-testid={testId}
          onSubmit={(e) => {
            e.preventDefault();
            onFinish?.(formValues);
          }}
        >
          {children}
        </form>
      ),
      {
        Item: ({ children }: PropsWithChildren<{ name?: string; rules?: unknown[] }>) => (
          <div>{children}</div>
        ),
        useForm: () => [formInstance],
        useWatch: () => undefined,
      },
    ),
    Input: Object.assign(
      ({
        'data-testid': testId,
        onChange,
        placeholder,
      }: {
        'data-testid'?: string;
        onChange?: (e: { target: { value: string } }) => void;
        placeholder?: string;
        prefix?: React.ReactNode;
      }) => (
        <input
          data-testid={testId}
          placeholder={placeholder}
          onChange={(e) => {
            formValues[testId === 'login-email' ? 'email' : testId ?? ''] = e.target.value;
            onChange?.(e);
          }}
        />
      ),
      {
        Password: ({
          'data-testid': testId,
          onChange,
          placeholder,
        }: {
          'data-testid'?: string;
          onChange?: (e: { target: { value: string } }) => void;
          placeholder?: string;
          prefix?: React.ReactNode;
        }) => (
          <input
            data-testid={testId}
            type="password"
            placeholder={placeholder}
            onChange={(e) => {
              formValues.password = e.target.value;
              onChange?.(e);
            }}
          />
        ),
      },
    ),
    Typography: {
      Title: ({ children }: PropsWithChildren) => <h3>{children}</h3>,
      Text: ({ children }: PropsWithChildren) => <span>{children}</span>,
    },
  };
});

const mockLogin = vi.fn();

vi.mock('../api/client', () => ({
  login: (...args: unknown[]) => mockLogin(...args),
}));

vi.mock('../api/errors', () => ({
  normalizeApiError: (err: unknown, fallback: string) => {
    const status = (err as { status?: number })?.status ?? 500;
    return { status, message: fallback };
  },
  formatApiErrorMessage: (err: { message: string }) => err.message,
}));

const mockSetAuth = vi.fn();

vi.mock('../store/authStore', () => ({
  useAuthStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({ setAuth: mockSetAuth }),
}));

function renderLogin() {
  return render(
    <MemoryRouter>
      <LoginPage />
    </MemoryRouter>,
  );
}

describe('LoginPage', () => {
  it('renders login form with email, password, and submit button', () => {
    renderLogin();

    expect(screen.getByText('CSG Admin')).toBeInTheDocument();
    expect(screen.getByTestId('login-email')).toBeInTheDocument();
    expect(screen.getByTestId('login-password')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Log in' })).toBeInTheDocument();
  });

  it('calls login API and navigates on successful admin login', async () => {
    const tokens = { access_token: 'at', refresh_token: 'rt' };
    const userData = { is_admin: true, email: 'admin@test.com' };
    mockLogin.mockResolvedValue({ data: userData, tokens });

    renderLogin();

    fireEvent.change(screen.getByTestId('login-email'), { target: { value: 'admin@test.com' } });
    fireEvent.change(screen.getByTestId('login-password'), { target: { value: 'pass123' } });
    fireEvent.submit(screen.getByTestId('login-form'));

    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledWith('admin@test.com', 'pass123');
    });

    await waitFor(() => {
      expect(mockSetAuth).toHaveBeenCalledWith('at', 'rt', userData);
    });
    expect(mockMessageSuccess).toHaveBeenCalledWith('Login successful');
    expect(mockNavigate).toHaveBeenCalledWith('/', { replace: true });
  });

  it('shows error when user is not an admin', async () => {
    mockLogin.mockResolvedValue({
      data: { is_admin: false },
      tokens: { access_token: 'at', refresh_token: 'rt' },
    });

    renderLogin();

    fireEvent.change(screen.getByTestId('login-email'), { target: { value: 'user@test.com' } });
    fireEvent.change(screen.getByTestId('login-password'), { target: { value: 'pass' } });
    fireEvent.submit(screen.getByTestId('login-form'));

    await waitFor(() => {
      expect(mockMessageError).toHaveBeenCalledWith('Access denied: admin privileges required');
    });
    expect(mockSetAuth).not.toHaveBeenCalled();
  });

  it('shows error message on 401 response', async () => {
    mockLogin.mockRejectedValue({ status: 401 });

    renderLogin();

    fireEvent.change(screen.getByTestId('login-email'), { target: { value: 'a@b.com' } });
    fireEvent.change(screen.getByTestId('login-password'), { target: { value: 'wrong' } });
    fireEvent.submit(screen.getByTestId('login-form'));

    await waitFor(() => {
      expect(mockMessageError).toHaveBeenCalledWith('Invalid email or password');
    });
  });

  it('shows rate limit message on 429 response', async () => {
    mockLogin.mockRejectedValue({ status: 429 });

    renderLogin();

    fireEvent.change(screen.getByTestId('login-email'), { target: { value: 'a@b.com' } });
    fireEvent.change(screen.getByTestId('login-password'), { target: { value: 'x' } });
    fireEvent.submit(screen.getByTestId('login-form'));

    await waitFor(() => {
      expect(mockMessageError).toHaveBeenCalledWith(
        'Too many login attempts. Please try again later.',
      );
    });
  });
});
