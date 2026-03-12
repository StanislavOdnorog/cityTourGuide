import { StoryCacheManager } from '@/services/cache';
import { useDownloadStore } from '@/store/useDownloadStore';

/**
 * Reconcile persisted download state with the actual cache on disk.
 * Removes download records whose cached audio files have been deleted
 * outside the normal flow (e.g. OS storage pressure, manual clear).
 *
 * Should be called once after the download store has hydrated.
 */
export async function reconcileDownloadState(): Promise<void> {
  const cacheManager = new StoryCacheManager();
  try {
    await cacheManager.init();

    // First: clean up stale DB rows and orphaned files on disk
    await cacheManager.reconcile();

    const { downloadedCities, removeCityDownloaded } = useDownloadStore.getState();
    const cityKeys = Object.keys(downloadedCities);
    if (cityKeys.length === 0) return;

    for (const key of cityKeys) {
      const meta = downloadedCities[key];
      if (!meta || meta.storyIds.length === 0) {
        removeCityDownloaded(meta?.cityId ?? Number(key));
        continue;
      }

      // Check if at least one story file is still cached.
      // If none remain, the city download is stale.
      let anyPresent = false;
      for (const storyId of meta.storyIds) {
        if (await cacheManager.isCached(storyId)) {
          anyPresent = true;
          break;
        }
      }

      if (!anyPresent) {
        removeCityDownloaded(meta.cityId);
      }
    }
  } finally {
    await cacheManager.destroy();
  }
}
