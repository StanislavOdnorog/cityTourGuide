import { act, renderHook } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { createElement } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { useCursorTableState } from './useCursorTableState';

const FILTER_VALUES = ['new', 'reviewed', 'resolved', 'dismissed'] as const;
const PAGE_SIZE_OPTIONS = [10, 20, 50] as const;

const defaultParams = {
  filterKey: 'status',
  filterValues: FILTER_VALUES,
  defaultFilter: '' as const,
  defaultPageSize: 20,
  pageSizeOptions: PAGE_SIZE_OPTIONS,
};

function createWrapper(initialEntry = '/reports') {
  return function Wrapper({ children }: PropsWithChildren) {
    return createElement(MemoryRouter, { initialEntries: [initialEntry] }, children);
  };
}

describe('useCursorTableState', () => {
  it('returns default values when URL has no params', () => {
    const { result } = renderHook(() => useCursorTableState(defaultParams), {
      wrapper: createWrapper(),
    });

    expect(result.current.filter).toBe('');
    expect(result.current.page).toBe(1);
    expect(result.current.pageSize).toBe(20);
    expect(result.current.cursor).toBeUndefined();
  });

  it('reads filter from URL', () => {
    const { result } = renderHook(() => useCursorTableState(defaultParams), {
      wrapper: createWrapper('/reports?status=new'),
    });

    expect(result.current.filter).toBe('new');
  });

  it('reads pageSize from URL', () => {
    const { result } = renderHook(() => useCursorTableState(defaultParams), {
      wrapper: createWrapper('/reports?pageSize=50'),
    });

    expect(result.current.pageSize).toBe(50);
  });

  it('ignores invalid filter values', () => {
    const { result } = renderHook(() => useCursorTableState(defaultParams), {
      wrapper: createWrapper('/reports?status=invalid'),
    });

    expect(result.current.filter).toBe('');
  });

  it('ignores invalid pageSize values', () => {
    const { result } = renderHook(() => useCursorTableState(defaultParams), {
      wrapper: createWrapper('/reports?pageSize=999'),
    });

    expect(result.current.pageSize).toBe(20);
  });

  it('falls back to page 1 when page param exists but no cursor', () => {
    const { result } = renderHook(() => useCursorTableState(defaultParams), {
      wrapper: createWrapper('/reports?page=3'),
    });

    expect(result.current.page).toBe(1);
  });

  it('setFilter resets page cursors and updates filter', () => {
    const { result } = renderHook(() => useCursorTableState(defaultParams), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.setFilter('new');
    });

    expect(result.current.filter).toBe('new');
    expect(result.current.page).toBe(1);
    expect(result.current.cursor).toBeUndefined();
  });

  it('recordNextCursor stores cursor for next page (verified via navigation)', () => {
    const { result } = renderHook(() => useCursorTableState(defaultParams), {
      wrapper: createWrapper(),
    });

    // Record cursor for page 2
    act(() => {
      result.current.recordNextCursor('abc123');
    });

    // Navigate to page 2 to verify cursor was stored
    act(() => {
      result.current.setPageAndSize(2, 20);
    });

    expect(result.current.page).toBe(2);
    expect(result.current.cursor).toBe('abc123');
  });

  it('setPageAndSize navigates to a page with known cursor', () => {
    const { result } = renderHook(() => useCursorTableState(defaultParams), {
      wrapper: createWrapper(),
    });

    // Record cursor for page 2
    act(() => {
      result.current.recordNextCursor('abc123');
    });

    // Navigate to page 2
    act(() => {
      result.current.setPageAndSize(2, 20);
    });

    expect(result.current.page).toBe(2);
    expect(result.current.cursor).toBe('abc123');
  });

  it('setPageAndSize with new size resets to page 1', () => {
    const { result } = renderHook(() => useCursorTableState(defaultParams), {
      wrapper: createWrapper(),
    });

    // Record cursor and go to page 2
    act(() => {
      result.current.recordNextCursor('abc123');
    });
    act(() => {
      result.current.setPageAndSize(2, 20);
    });

    expect(result.current.page).toBe(2);

    // Change page size
    act(() => {
      result.current.setPageAndSize(1, 50);
    });

    expect(result.current.page).toBe(1);
    expect(result.current.pageSize).toBe(50);
    expect(result.current.cursor).toBeUndefined();
  });
});
