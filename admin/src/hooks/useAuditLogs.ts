import { useQuery } from '@tanstack/react-query';
import { listAuditLogs } from '../api';

interface UseAuditLogsOptions {
  actorId?: string;
  action?: string;
  resourceType?: string;
  createdFrom?: string;
  createdTo?: string;
  cursor?: string;
  limit?: number;
}

export function useAuditLogs({
  actorId = '',
  action = '',
  resourceType = '',
  createdFrom = '',
  createdTo = '',
  cursor,
  limit = 20,
}: UseAuditLogsOptions = {}) {
  const logs = useQuery({
    queryKey: [
      'audit-logs',
      actorId,
      action,
      resourceType,
      createdFrom,
      createdTo,
      cursor ?? null,
      limit,
    ],
    queryFn: () =>
      listAuditLogs({
        limit,
        ...(cursor ? { cursor } : {}),
        ...(actorId ? { actor_id: actorId } : {}),
        ...(action ? { action } : {}),
        ...(resourceType ? { resource_type: resourceType } : {}),
        ...(createdFrom ? { created_from: createdFrom } : {}),
        ...(createdTo ? { created_to: createdTo } : {}),
      }),
    staleTime: 15_000,
  });

  return { logs };
}
