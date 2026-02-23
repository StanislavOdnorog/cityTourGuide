import { useDownloadStore } from '../useDownloadStore';

describe('useDownloadStore', () => {
  beforeEach(() => {
    useDownloadStore.getState().reset();
    // Clear downloaded city IDs
    useDownloadStore.setState({ downloadedCityIds: new Set<number>() });
  });

  it('has correct initial state', () => {
    const state = useDownloadStore.getState();
    expect(state.status).toBe('idle');
    expect(state.progress.completedBytes).toBe(0);
    expect(state.progress.totalBytes).toBe(0);
    expect(state.progress.completedFiles).toBe(0);
    expect(state.progress.totalFiles).toBe(0);
    expect(state.error).toBeNull();
  });

  it('setStatus updates status', () => {
    useDownloadStore.getState().setStatus('downloading');
    expect(useDownloadStore.getState().status).toBe('downloading');
  });

  it('setProgress updates progress partially', () => {
    useDownloadStore.getState().setProgress({ totalBytes: 1000, totalFiles: 10 });
    const { progress } = useDownloadStore.getState();
    expect(progress.totalBytes).toBe(1000);
    expect(progress.totalFiles).toBe(10);
    expect(progress.completedBytes).toBe(0);
    expect(progress.completedFiles).toBe(0);

    useDownloadStore.getState().setProgress({ completedFiles: 5, completedBytes: 500 });
    const updated = useDownloadStore.getState().progress;
    expect(updated.completedFiles).toBe(5);
    expect(updated.completedBytes).toBe(500);
    expect(updated.totalBytes).toBe(1000);
  });

  it('setError stores error message', () => {
    useDownloadStore.getState().setError('Network error');
    expect(useDownloadStore.getState().error).toBe('Network error');
  });

  it('markCityDownloaded adds city to set', () => {
    useDownloadStore.getState().markCityDownloaded(1);
    expect(useDownloadStore.getState().downloadedCityIds.has(1)).toBe(true);

    useDownloadStore.getState().markCityDownloaded(2);
    expect(useDownloadStore.getState().downloadedCityIds.has(1)).toBe(true);
    expect(useDownloadStore.getState().downloadedCityIds.has(2)).toBe(true);
  });

  it('removeCityDownloaded removes city from set', () => {
    useDownloadStore.getState().markCityDownloaded(1);
    useDownloadStore.getState().markCityDownloaded(2);

    useDownloadStore.getState().removeCityDownloaded(1);
    expect(useDownloadStore.getState().downloadedCityIds.has(1)).toBe(false);
    expect(useDownloadStore.getState().downloadedCityIds.has(2)).toBe(true);
  });

  it('reset clears status, progress, and error but not downloadedCityIds', () => {
    useDownloadStore.getState().setStatus('downloading');
    useDownloadStore.getState().setProgress({ completedFiles: 5, totalFiles: 10 });
    useDownloadStore.getState().setError('error');
    useDownloadStore.getState().markCityDownloaded(1);

    useDownloadStore.getState().reset();

    const state = useDownloadStore.getState();
    expect(state.status).toBe('idle');
    expect(state.progress.completedFiles).toBe(0);
    expect(state.error).toBeNull();
    expect(state.downloadedCityIds.has(1)).toBe(true);
  });
});
