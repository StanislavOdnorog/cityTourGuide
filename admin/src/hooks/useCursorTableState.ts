import { useCallback, useEffect, useMemo, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';

export interface CursorTableParams<F extends string = string> {
  filterKey: string;
  filterValues: readonly F[];
  defaultFilter: F | '';
  defaultPageSize: number;
  pageSizeOptions: readonly number[];
}

export interface CursorTableState<F extends string = string> {
  filter: F | '';
  page: number;
  pageSize: number;
  cursor: string | undefined;
  pageCursors: Record<number, string | null>;
  setFilter: (value: F | '') => void;
  setPageAndSize: (page: number, pageSize: number) => void;
  recordNextCursor: (nextCursor: string) => void;
  resetPagination: () => void;
}

/**
 * Syncs cursor-based table pagination state with URL search params.
 * Changing filter or page size resets to page 1 with a fresh cursor chain.
 */
export function useCursorTableState<F extends string = string>({
  filterKey,
  filterValues,
  defaultFilter,
  defaultPageSize,
  pageSizeOptions,
}: CursorTableParams<F>): CursorTableState<F> {
  const [searchParams, setSearchParams] = useSearchParams();
  const pageCursorsRef = useRef<Record<number, string | null>>({ 1: null });

  const filter = useMemo(() => {
    const raw = searchParams.get(filterKey) ?? '';
    if (raw === '') return defaultFilter;
    return (filterValues as readonly string[]).includes(raw) ? (raw as F) : defaultFilter;
  }, [searchParams, filterKey, filterValues, defaultFilter]);

  const page = useMemo(() => {
    const raw = searchParams.get('page');
    if (!raw) return 1;
    const n = parseInt(raw, 10);
    return Number.isFinite(n) && n >= 1 ? n : 1;
  }, [searchParams]);

  const pageSize = useMemo(() => {
    const raw = searchParams.get('pageSize');
    if (!raw) return defaultPageSize;
    const n = parseInt(raw, 10);
    return pageSizeOptions.includes(n) ? n : defaultPageSize;
  }, [searchParams, defaultPageSize, pageSizeOptions]);

  // Deterministically rebuild cursor chain: if the URL says page > 1 but we
  // don't have the cursor for that page, fall back to page 1.
  const effectivePage = useMemo(() => {
    if (page === 1) return 1;
    return page in pageCursorsRef.current ? page : 1;
  }, [page]);

  // Sync effective page back to URL if it differs (e.g. cursor not available)
  useEffect(() => {
    if (effectivePage !== page) {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          if (effectivePage === 1) {
            next.delete('page');
          } else {
            next.set('page', String(effectivePage));
          }
          return next;
        },
        { replace: true },
      );
    }
  }, [effectivePage, page, setSearchParams]);

  const cursor = pageCursorsRef.current[effectivePage] ?? undefined;

  const setFilter = useCallback(
    (value: F | '') => {
      pageCursorsRef.current = { 1: null };
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          if (value === '' || value === defaultFilter) {
            next.delete(filterKey);
          } else {
            next.set(filterKey, value);
          }
          next.delete('page');
          return next;
        },
        { replace: false },
      );
    },
    [setSearchParams, filterKey, defaultFilter],
  );

  const setPageAndSize = useCallback(
    (newPage: number, newPageSize: number) => {
      const sizeChanged = newPageSize !== pageSize;
      if (sizeChanged) {
        pageCursorsRef.current = { 1: null };
        setSearchParams(
          (prev) => {
            const next = new URLSearchParams(prev);
            next.delete('page');
            if (newPageSize === defaultPageSize) {
              next.delete('pageSize');
            } else {
              next.set('pageSize', String(newPageSize));
            }
            return next;
          },
          { replace: false },
        );
        return;
      }

      if (newPage === effectivePage) return;

      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          if (newPage === 1) {
            next.delete('page');
          } else {
            next.set('page', String(newPage));
          }
          return next;
        },
        { replace: false },
      );
    },
    [pageSize, effectivePage, setSearchParams, defaultPageSize],
  );

  const recordNextCursor = useCallback(
    (nextCursor: string) => {
      const nextPage = effectivePage + 1;
      if (pageCursorsRef.current[nextPage] !== nextCursor) {
        pageCursorsRef.current = {
          ...pageCursorsRef.current,
          [nextPage]: nextCursor,
        };
      }
    },
    [effectivePage],
  );

  const resetPagination = useCallback(() => {
    pageCursorsRef.current = { 1: null };
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        next.delete('page');
        return next;
      },
      { replace: false },
    );
  }, [setSearchParams]);

  return {
    filter,
    page: effectivePage,
    pageSize,
    cursor,
    pageCursors: pageCursorsRef.current,
    setFilter,
    setPageAndSize,
    recordNextCursor,
    resetPagination,
  };
}
