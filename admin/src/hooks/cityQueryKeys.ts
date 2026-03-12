export const cityQueryKeys = {
  all: ['cities'] as const,
  list: (cursor: string | undefined, limit: number, includeDeleted: boolean) =>
    ['cities', 'list', cursor ?? null, limit, includeDeleted] as const,
};
