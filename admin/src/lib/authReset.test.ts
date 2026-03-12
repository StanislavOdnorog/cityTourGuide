import { QueryClient } from '@tanstack/react-query';
import { describe, expect, it, vi } from 'vitest';
import { useAuthStore } from '../store/authStore';
import { registerQueryClient, resetAuth } from './authReset';

describe('resetAuth', () => {
  it('clears auth store and localStorage', () => {
    // Seed auth state
    useAuthStore.getState().setAuth('tok', 'ref', { id: '1', email: 'a@b.com' } as any);
    expect(useAuthStore.getState().isAuthenticated).toBe(true);
    expect(localStorage.getItem('csg_admin_token')).toBe('tok');

    resetAuth();

    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(useAuthStore.getState().token).toBeNull();
    expect(localStorage.getItem('csg_admin_token')).toBeNull();
  });

  it('cancels queries and clears query cache when queryClient registered', () => {
    const qc = new QueryClient();
    const cancelSpy = vi.spyOn(qc, 'cancelQueries');
    const clearSpy = vi.spyOn(qc, 'clear');

    registerQueryClient(qc);
    resetAuth();

    expect(cancelSpy).toHaveBeenCalled();
    expect(clearSpy).toHaveBeenCalled();
  });
});
