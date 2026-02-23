import { useCallback, useRef } from 'react';
import { fetchCityDownloadManifest } from '@/api/endpoints';
import { StoryCacheManager } from '@/services/cache';
import { useDownloadStore } from '@/store/useDownloadStore';
import type { DownloadManifestItem, NearbyStoryCandidate } from '@/types';

/**
 * Convert a DownloadManifestItem to a NearbyStoryCandidate shape
 * so it can be passed to StoryCacheManager.getAudioPath().
 */
function toCandidate(item: DownloadManifestItem): NearbyStoryCandidate {
  return {
    poi_id: item.poi_id,
    poi_name: item.poi_name,
    story_id: item.story_id,
    story_text: '',
    audio_url: item.audio_url,
    duration_sec: item.duration_sec,
    distance_m: 0,
    score: 0,
  };
}

/**
 * Hook for downloading all stories for a city for offline use.
 */
export function useDownloadCity(cityId: number, language = 'en') {
  const {
    status,
    progress,
    error,
    downloadedCityIds,
    setStatus,
    setProgress,
    setError,
    markCityDownloaded,
    reset,
  } = useDownloadStore();

  const cancelledRef = useRef(false);
  const cacheManagerRef = useRef<StoryCacheManager | null>(null);

  const isDownloaded = downloadedCityIds.has(cityId);

  const startDownload = useCallback(async () => {
    cancelledRef.current = false;
    reset();
    setStatus('fetching_manifest');

    try {
      const manifest = await fetchCityDownloadManifest(cityId, language);

      if (cancelledRef.current) return;

      const items = manifest.data.filter((item) => item.audio_url !== null);
      if (items.length === 0) {
        setStatus('completed');
        markCityDownloaded(cityId);
        return;
      }

      setStatus('downloading');
      setProgress({
        totalBytes: manifest.total_size_bytes,
        totalFiles: items.length,
        completedBytes: 0,
        completedFiles: 0,
      });

      // Initialize cache manager
      if (!cacheManagerRef.current) {
        cacheManagerRef.current = new StoryCacheManager();
        await cacheManagerRef.current.init();
      }
      const cacheManager = cacheManagerRef.current;

      let completedBytes = 0;
      let completedFiles = 0;

      for (const item of items) {
        if (cancelledRef.current) {
          setStatus('idle');
          return;
        }

        const candidate = toCandidate(item);
        await cacheManager.getAudioPath(candidate);

        completedFiles++;
        completedBytes += item.file_size_bytes;
        setProgress({ completedFiles, completedBytes });
      }

      setStatus('completed');
      markCityDownloaded(cityId);
    } catch (err) {
      if (!cancelledRef.current) {
        setStatus('error');
        setError(err instanceof Error ? err.message : 'Download failed');
      }
    }
  }, [cityId, language, reset, setStatus, setProgress, setError, markCityDownloaded]);

  const cancelDownload = useCallback(() => {
    cancelledRef.current = true;
    reset();
  }, [reset]);

  return {
    status,
    progress,
    error,
    isDownloaded,
    startDownload,
    cancelDownload,
  };
}
