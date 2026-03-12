import { fireEvent, render, screen } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import NotFoundPage from './NotFoundPage';

vi.mock('antd', () => ({
  Button: ({ children, onClick }: PropsWithChildren<{ onClick?: () => void }>) => (
    <button onClick={onClick}>{children}</button>
  ),
  Result: ({
    status,
    title,
    subTitle,
    extra,
  }: {
    status: string;
    title: string;
    subTitle: string;
    extra: React.ReactNode;
  }) => (
    <div data-testid="result" data-status={status}>
      <h1>{title}</h1>
      <p>{subTitle}</p>
      {extra}
    </div>
  ),
}));

function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="location">{location.pathname}</div>;
}

function renderWithRouter(initialEntry = '/unknown') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <NotFoundPage />
      <LocationDisplay />
    </MemoryRouter>,
  );
}

describe('NotFoundPage', () => {
  it('renders 404 result with correct status and text', () => {
    renderWithRouter();

    const result = screen.getByTestId('result');
    expect(result).toHaveAttribute('data-status', '404');
    expect(screen.getByText('404')).toBeInTheDocument();
    expect(screen.getByText('Page not found')).toBeInTheDocument();
  });

  it('renders a button to navigate back to dashboard', () => {
    renderWithRouter();

    expect(screen.getByRole('button', { name: 'Back to Dashboard' })).toBeInTheDocument();
  });

  it('navigates to / when the button is clicked', () => {
    renderWithRouter();

    fireEvent.click(screen.getByRole('button', { name: 'Back to Dashboard' }));

    expect(screen.getByTestId('location')).toHaveTextContent('/');
  });
});
